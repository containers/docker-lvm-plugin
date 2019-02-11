package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
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

func lvdisplayGrep(vgName, lvName, keyword string) (bool, string, error) {
	outByttes, err := exec.Command("lvdisplay", fmt.Sprintf("/dev/%s/%s", vgName, lvName)).Output()
	if err != nil {
		return false, "", err
	}
	var result []string
	outStr := strings.TrimSpace(string(outByttes[:]))

	for _, line := range strings.Split(outStr, "\n") {
		if strings.Contains(line, keyword) {
			result = append(result, line)
		}
	}

	if len(result) > 0 {
		return true, strings.Join(result, "\n"), nil
	}

	return false, "", nil
}

func isThinlyProvisioned(vgName, lvName string) (bool, string, error) {
	return lvdisplayGrep(vgName, lvName, "LV Pool")
}

func getVolumeCreationDateTime(vgName, lvName string) (time.Time, error) {
	_, creationDateTime, err := lvdisplayGrep(vgName, lvName, "LV Creation host")
	if err != nil {
		return time.Time{}, err
	}

	// creationDateTime is in the form "LV Creation host, time localhost, 2018-11-18 13:46:08 -0100"
	tokens := strings.Split(creationDateTime, ",")
	date := strings.TrimSpace(tokens[len(tokens)-1])
	return time.Parse("2006-01-02 15:04:05 -0700", date)
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
