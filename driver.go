package main

import (
	"fmt"
	"log/syslog"
	"os"
	"os/exec"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
)

type lvmDriver struct {
	home     string
	vgConfig string
	volumes  map[string]*vol
	count    map[string]int
	mu       sync.RWMutex
	logger   *syslog.Writer
}

type vol struct {
	Name       string `json:"name"`
	MountPoint string `json:"mountpoint"`
	Type       string `json:"type"`
	Source     string `json:"source"`
}

func newDriver(home, vgConfig string) (*lvmDriver, error) {
	logger, err := syslog.New(syslog.LOG_ERR, "docker-lvm-plugin")
	if err != nil {
		return nil, err
	}

	return &lvmDriver{
		home:     home,
		vgConfig: vgConfig,
		volumes:  make(map[string]*vol),
		count:    make(map[string]int),
		logger:   logger,
	}, nil
}

func (l *lvmDriver) Create(req volume.Request) volume.Response {
	l.mu.Lock()
	defer l.mu.Unlock()

	if v, exists := l.volumes[req.Name]; exists {
		return resp(v.MountPoint)
	}

	vgName, err := getVolumegroupName(l.vgConfig)
	if err != nil {
		return resp(err)
	}

	cmdArgs := []string{"-n", req.Name, "--setactivationskip", "n"}
	snap, ok := req.Options["snapshot"]
	isSnapshot := ok && snap != ""
	isThinSnap := false
	if isSnapshot {
		if isThinSnap, err = isThinlyProvisioned(vgName, snap); err != nil {
			l.logger.Err(fmt.Sprintf("Create: lvdisplayGrep error: %s", err))
			return resp(fmt.Errorf("Error creating volume"))
		}
	}
	s, ok := req.Options["size"]
	hasSize := ok && s != ""

	if !hasSize && !isThinSnap {
		return resp(fmt.Errorf("Please specify a size with --opt size="))
	}

	if hasSize && isThinSnap {
		return resp(fmt.Errorf("Please don't specify --size for thin snapshots"))
	}

	if isSnapshot {
		cmdArgs = append(cmdArgs, "--snapshot")
		if hasSize {
			cmdArgs = append(cmdArgs, "--size", s)
		}
		cmdArgs = append(cmdArgs, vgName+"/"+snap)
	} else if thin, ok := req.Options["thinpool"]; ok && thin != "" {
		cmdArgs = append(cmdArgs, "--virtualsize", s)
		cmdArgs = append(cmdArgs, "--thin")
		cmdArgs = append(cmdArgs, vgName+"/"+thin)
	} else {
		cmdArgs = append(cmdArgs, "--size", s)
		cmdArgs = append(cmdArgs, vgName)
	}
	cmd := exec.Command("lvcreate", cmdArgs...)
	if out, err := cmd.CombinedOutput(); err != nil {
		l.logger.Err(fmt.Sprintf("Create: lvcreate error: %s output %s", err, string(out)))
		return resp(fmt.Errorf("Error creating volume"))
	}

	defer func() {
		if err != nil {
			removeLogicalVolume(req.Name, vgName)
		}
	}()

	if !isSnapshot {
		cmd = exec.Command("mkfs.xfs", fmt.Sprintf("/dev/%s/%s", vgName, req.Name))
		out, err := cmd.CombinedOutput()
		if err != nil {
			l.logger.Err(fmt.Sprintf("Create: mkfs.xfs error: %s output %s", err, string(out)))
			return resp(fmt.Errorf("Error partitioning volume"))
		}
	}

	mp := getMountpoint(l.home, req.Name)
	err = os.MkdirAll(mp, 0700)
	if err != nil {
		return resp(err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(mp)
		}
	}()

	v := &vol{Name: req.Name, MountPoint: mp}
	if isSnapshot {
		v.Type = "Snapshot"
		v.Source = snap
	}
	l.volumes[v.Name] = v
	l.count[v.Name] = 0
	err = saveToDisk(l.volumes, l.count)
	if err != nil {
		return resp(err)
	}
	return resp(v.MountPoint)
}

func (l *lvmDriver) List(req volume.Request) volume.Response {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var ls []*volume.Volume
	for _, vol := range l.volumes {
		v := &volume.Volume{
			Name:       vol.Name,
			Mountpoint: vol.MountPoint,
		}
		ls = append(ls, v)
	}
	return volume.Response{Volumes: ls}
}

func (l *lvmDriver) Get(req volume.Request) volume.Response {
	l.mu.RLock()
	defer l.mu.RUnlock()
	v, exists := l.volumes[req.Name]
	if !exists {
		return resp(fmt.Errorf("No such volume"))
	}
	var res volume.Response
	res.Volume = &volume.Volume{
		Name:       v.Name,
		Mountpoint: v.MountPoint,
	}
	return res
}

func (l *lvmDriver) Remove(req volume.Request) volume.Response {
	l.mu.Lock()
	defer l.mu.Unlock()

	vgName, err := getVolumegroupName(l.vgConfig)
	if err != nil {
		return resp(err)
	}

	isOrigin := func() bool {
		for _, vol := range l.volumes {
			if vol.Name == req.Name {
				continue
			}
			if vol.Type == "Snapshot" && vol.Source == req.Name {
				return true
			}
		}
		return false
	}()

	if isOrigin {
		return resp(fmt.Errorf("Error removing volume, all snapshot destinations must be removed before removing the original volume"))
	}

	if err := os.RemoveAll(getMountpoint(l.home, req.Name)); err != nil {
		return resp(err)
	}

	if out, err := removeLogicalVolume(req.Name, vgName); err != nil {
		l.logger.Err(fmt.Sprintf("Remove: removeLogicalVolume error %s output %s", err, string(out)))
		return resp(fmt.Errorf("error removing volume"))
	}

	delete(l.count, req.Name)
	delete(l.volumes, req.Name)
	if err := saveToDisk(l.volumes, l.count); err != nil {
		return resp(err)
	}
	return resp(getMountpoint(l.home, req.Name))
}

func (l *lvmDriver) Path(req volume.Request) volume.Response {
	return resp(getMountpoint(l.home, req.Name))
}

func (l *lvmDriver) Mount(req volume.MountRequest) volume.Response {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.count[req.Name]++
	vgName, err := getVolumegroupName(l.vgConfig)
	if err != nil {
		return resp(err)
	}

	isSnap := func() bool {
		if v, ok := l.volumes[req.Name]; ok {
			if v.Type == "Snapshot" {
				return true
			}
		}
		return false
	}()

	if l.count[req.Name] == 1 {
		mountArgs := []string{fmt.Sprintf("/dev/%s/%s", vgName, req.Name), getMountpoint(l.home, req.Name)}
		if isSnap {
			mountArgs = append([]string{"-o", "nouuid"}, mountArgs...)
		}
		cmd := exec.Command("mount", mountArgs...)
		if out, err := cmd.CombinedOutput(); err != nil {
			l.logger.Err(fmt.Sprintf("Mount: mount error: %s output %s", err, string(out)))
			return resp(fmt.Errorf("error mouting volume"))
		}
	}
	if err := saveToDisk(l.volumes, l.count); err != nil {
		return resp(err)
	}
	return resp(getMountpoint(l.home, req.Name))
}

func (l *lvmDriver) Unmount(req volume.UnmountRequest) volume.Response {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.count[req.Name]--
	if l.count[req.Name] == 0 {
		cmd := exec.Command("umount", getMountpoint(l.home, req.Name))
		if out, err := cmd.CombinedOutput(); err != nil {
			l.logger.Err(fmt.Sprintf("Unmount: unmount error: %s output %s", err, string(out)))
			return resp(fmt.Errorf("error unmounting volume"))
		}
	}
	if err := saveToDisk(l.volumes, l.count); err != nil {
		return resp(err)
	}
	return resp(getMountpoint(l.home, req.Name))
}

func (l *lvmDriver) Capabilities(req volume.Request) volume.Response {
	var res volume.Response
	res.Capabilities = volume.Capability{Scope: "local"}
	return res
}

func resp(r interface{}) volume.Response {
	switch t := r.(type) {
	case error:
		return volume.Response{Err: t.Error()}
	case string:
		return volume.Response{Mountpoint: t}
	default:
		return volume.Response{Err: "bad value writing response"}
	}
}
