package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/syslog"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/docker/docker/pkg/integration"
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
			l.logger.Err(fmt.Sprintf("error checking if origin volume is thin, proceeding as if it is not: %s", err))
		}
	}
	s, ok := req.Options["size"]
	hasSize := ok && s != ""

	if !hasSize && !isThinSnap {
		return resp(fmt.Errorf("Please specify a size with --size"))
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
		return resp(fmt.Errorf("error creating volume"))
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
			return resp(fmt.Errorf("error partitioning volume"))
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

	v := &vol{req.Name, mp}
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
		return resp(fmt.Errorf("no such volume"))
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

	if isOrigin, err := isSnapshotOrigin(vgName, req.Name); err != nil {
		l.logger.Err(fmt.Sprintf("isSnapshotOrigin error: %s", err))
		return resp(fmt.Errorf("error removing volume, unable to determine if volume is a snapshot origin"))
	} else if isOrigin {
		return resp(fmt.Errorf("error removing volume, all snapshot destinations must be removed before removing a volume"))
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

func removeLogicalVolume(name, vgName string) ([]byte, error) {
	cmd := exec.Command("lvremove", "--force", fmt.Sprintf("%s/%s", vgName, name))
	if out, err := cmd.CombinedOutput(); err != nil {
		return out, err
	}
	return nil, nil
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
	if l.count[req.Name] == 1 {
		mountArgs := []string{fmt.Sprintf("/dev/%s/%s", vgName, req.Name), getMountpoint(l.home, req.Name)}
		if isSnap, err := isSnapshot(vgName, req.Name); err != nil {
			l.logger.Err(fmt.Sprintf("error checking if volume is a snapshot, proceeding as if it is not: %s", err))
		} else if isSnap {
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

var allowedConfKeys = map[string]bool{
	"VOLUME_GROUP": true,
}

func getVolumegroupName(vgConfig string) (string, error) {
	vgName := ""
	inFile, err := os.Open(vgConfig)
	if err != nil {
		return "", err
	}
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		str := scanner.Text()
		if strings.HasPrefix(str, "#") {
			continue
		}
		vgSlice := strings.SplitN(str, "=", 2)
		if !allowedConfKeys[vgSlice[0]] || len(vgSlice) == 1 {
			continue
		}
		vgName = vgSlice[1]
		break
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	if vgName == "" {
		return "", fmt.Errorf("Volume group name must be provided for volume creation. Please update the config file %s with volume group name.", vgConfig)
	}

	return strings.TrimSpace(vgName), nil
}

func getMountpoint(home, name string) string {
	return path.Join(home, name)
}

func saveToDisk(volumes map[string]*vol, count map[string]int) error {
	// Save volume store metadata.
	fhVolumes, err := os.Create(lvmVolumesConfigPath)
	if err != nil {
		return err
	}
	defer fhVolumes.Close()

	if err := json.NewEncoder(fhVolumes).Encode(&volumes); err != nil {
		return err
	}

	// Save count store metadata.
	fhCount, err := os.Create(lvmCountConfigPath)
	if err != nil {
		return err
	}
	defer fhCount.Close()

	return json.NewEncoder(fhCount).Encode(&count)
}

func loadFromDisk(l *lvmDriver) error {
	// Load volume store metadata
	jsonVolumes, err := os.Open(lvmVolumesConfigPath)
	if err != nil {
		return err
	}
	defer jsonVolumes.Close()

	if err := json.NewDecoder(jsonVolumes).Decode(&l.volumes); err != nil {
		return err
	}

	// Load count store metadata
	jsonCount, err := os.Open(lvmCountConfigPath)
	if err != nil {
		return err
	}
	defer jsonCount.Close()

	return json.NewDecoder(jsonCount).Decode(&l.count)
}

func lvdisplayGrep(vgName, lvName, keyword string) (bool, error) {
	var b2 bytes.Buffer

	cmd1 := exec.Command("lvdisplay", fmt.Sprintf("/dev/%s/%s", vgName, lvName))
	cmd2 := exec.Command("grep", keyword)

	r, w := io.Pipe()
	cmd1.Stdout = w
	cmd2.Stdin = r
	cmd2.Stdout = &b2

	if err := cmd1.Start(); err != nil {
		return false, err
	}
	if err := cmd2.Start(); err != nil {
		return false, err
	}
	if err := cmd1.Wait(); err != nil {
		return false, err
	}
	w.Close()
	if err := cmd2.Wait(); err != nil {
		exitCode := integration.ProcessExitCode(err)
		if exitCode != 1 {
			return false, err
		}
	}
	if b2.Len() != 0 {
		return true, nil
	}
	return false, nil
}

func isThinlyProvisioned(vgName, lvName string) (bool, error) {
	return lvdisplayGrep(vgName, lvName, "LV Pool")
}

func isSnapshot(vgName, lvName string) (bool, error) {
	regSnap, err := lvdisplayGrep(vgName, lvName, "destination of")
	thinSnap, err := lvdisplayGrep(vgName, lvName, "LV Thin origin")
	return (regSnap || thinSnap), err
}

func isSnapshotOrigin(vgName, lvName string) (bool, error) {
	return lvdisplayGrep(vgName, lvName, "source of")
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

func (l *lvmDriver) Capabilities(req volume.Request) volume.Response {
    var res volume.Response
    res.Capabilities = volume.Capability{Scope: "local"}
    return res
}
