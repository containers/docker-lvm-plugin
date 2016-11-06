package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	icmd "github.com/docker/docker/pkg/system"
)

var allowedConfKeys = map[string]bool{
	"VOLUME_GROUP": true,
}

func removeLogicalVolume(name, vgName string) ([]byte, error) {
	cmd := exec.Command("lvremove", "--force", fmt.Sprintf("%s/%s", vgName, name))
	if out, err := cmd.CombinedOutput(); err != nil {
		return out, err
	}
	return nil, nil
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
		exitCode := icmd.ProcessExitCode(err)
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

func keyFileExists(keyFile string) error {
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file does not exist: %s", keyFile)
	}
	return nil
}

func cryptsetupInstalled() error {
	if _, err := exec.LookPath("cryptsetup"); err != nil {
		return fmt.Errorf("'cryptsetup' executable not found")
	}
	return nil
}

func logicalDevice(vgName, lvName string) string {
	return fmt.Sprintf("/dev/%s/%s", vgName, lvName)
}

func luksDevice(lvName string) string {
	return fmt.Sprintf("/dev/mapper/%s", luksDeviceName(lvName))
}

func luksDeviceName(lvName string) string {
	return fmt.Sprintf("luks-%s", lvName)
}
