package hw

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ListMechanicalDisks returns device paths like /dev/sda for rotational disks
func ListMechanicalDisks() ([]string, error) {
	entries, err := filepath.Glob("/sys/block/*")
	if err != nil {
		return nil, err
	}
	var disks []string
	for _, e := range entries {
		base := filepath.Base(e)
		// ignore loop, ram, dm-* and nvme by rotational check
		rotPath := filepath.Join(e, "queue/rotational")
		data, err := os.ReadFile(rotPath)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == "1" {
			dev := filepath.Join("/dev", base)
			if _, err := os.Stat(dev); err == nil {
				disks = append(disks, dev)
			}
		}
	}
	return disks, nil
}

// GetDriveState uses hdparm -C /dev/sdX to query state; returns the trailing state string (e.g. "standby" or "active/idle")
func GetDriveState(dev string) (string, error) {
	out, err := exec.Command("hdparm", "-C", dev).CombinedOutput()
	if err != nil {
		// hdparm may exit non-zero; still try to parse output
	}
	// parse lines for "drive state is:"
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(strings.ToLower(line), "drive state is:"); idx != -1 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	// fallback: return full output
	return strings.TrimSpace(string(out)), fmt.Errorf("unexpected hdparm output")
}

// SetStandbyTimeout sets hdparm -S <value> for device. If value == 0, disables spindown timer.
func SetStandbyTimeout(dev string, value int) error {
	cmd := exec.Command("hdparm", "-S", fmt.Sprintf("%d", value), dev)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return cmd.Run()
}
