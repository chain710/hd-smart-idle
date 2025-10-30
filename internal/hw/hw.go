package hw

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

// Drive state constants
const (
	DriveStateActive  = "active"
	DriveStateStandby = "standby"
)

// HDDControl defines an abstraction for HDD operations used by the daemon.
// It allows swapping implementations for testing or platform-specific behavior.
type HDDControl interface {
	// List returns device paths like /dev/sda for rotational disks
	List() ([]string, error)
	// GetState queries device state (e.g. returns string containing "standby" or "active/idle")
	GetState(dev string) (string, error)
	// SetStandbyTimeout sets hdparm -S <value> for device. If value == 0, disables spindown timer.
	SetStandbyTimeout(dev string, value int) error
}

// DefaultHDDControl is the default implementation of HDDControl that
// uses the host filesystem and hdparm binary.
type defaultHDDControl struct {
	fsys fs.FS
}

// NewHDDControl returns the default HDDControl implementation which uses the
// host filesystem and hdparm binary.
func NewHDDControl() HDDControl { return defaultHDDControl{fsys: os.DirFS("/")} }

func (d defaultHDDControl) List() ([]string, error) {
	// operate on the configured fs.FS (allows testing with fstest.MapFS)
	entries, err := fs.Glob(d.fsys, "sys/block/*")
	if err != nil {
		return nil, err
	}
	var disks []string
	for _, e := range entries {
		base := path.Base(e)
		// ignore loop, ram, dm-* and nvme by rotational check
		rotPath := path.Join(e, "queue/rotational")
		data, err := fs.ReadFile(d.fsys, rotPath)
		if err != nil {
			// missing rotational file or unreadable -> skip this entry
			continue
		}
		if strings.TrimSpace(string(data)) == "1" {
			// build dev path as /dev/<base>
			devPath := path.Join("/dev", base)
			// Check existence within provided FS; strip leading / for fs.Stat
			checkPath := path.Join("dev", base)
			if _, err := fs.Stat(d.fsys, checkPath); err == nil {
				disks = append(disks, devPath)
			}
		}
	}
	return disks, nil
}

func (d defaultHDDControl) GetState(dev string) (string, error) {
	out, err := exec.Command(hdparmPath(), "-C", dev).CombinedOutput()
	return d.parseHDParmState(string(out), err)
}

// parseHDParmState parses the output of `hdparm -C` command and returns
// a normalized state: "active" or "standby". Returns os.ErrNotExist if device
// not found, or other error if parsing fails.
func (d defaultHDDControl) parseHDParmState(output string, cmdErr error) (string, error) {
	output = strings.TrimSpace(output)
	if cmdErr != nil {
		if strings.Contains(output, "No such file or directory") {
			return "", os.ErrNotExist
		} else {
			return "", fmt.Errorf("hdparm command error(%w): %s", cmdErr, output)
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader([]byte(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(strings.ToLower(line), "drive state is:"); idx != -1 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				state := strings.TrimSpace(parts[1])
				// Normalize to standard enum values
				if strings.Contains(strings.ToLower(state), "standby") {
					return DriveStateStandby, nil
				}
				if strings.Contains(strings.ToLower(state), "active") || strings.Contains(strings.ToLower(state), "idle") {
					return DriveStateActive, nil
				}
			}
		}
	}

	// parse failed or "drive state is:" not found
	return "", fmt.Errorf("malformed hdparm output: %v", output)
}

// SetStandbyTimeout implements HDDControl.SetStandbyTimeout for the default implementation.
// It delegates to the package-level SetStandbyTimeout function to perform the actual hdparm call.
func (defaultHDDControl) SetStandbyTimeout(dev string, value int) error {
	cmd := exec.Command(hdparmPath(), "-S", fmt.Sprintf("%d", value), dev)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return cmd.Run()
}

// hdparmPath returns the path to the hdparm binary. It checks the HDPARM_PATH
// environment variable and falls back to /sbin/hdparm when not set.
func hdparmPath() string {
	if p, ok := os.LookupEnv("HDPARM_PATH"); ok && p != "" {
		return p
	}
	return "/sbin/hdparm"
}

// NewDryRunHDDControl returns an HDDControl wrapper that logs SetStandbyTimeout
// calls instead of executing them. Useful for dry-run/testing modes.
func NewDryRunHDDControl(inner HDDControl) HDDControl {
	return dryRunHDDControl{inner: inner}
}

type dryRunHDDControl struct {
	inner HDDControl
}

func (d dryRunHDDControl) List() ([]string, error)             { return d.inner.List() }
func (d dryRunHDDControl) GetState(dev string) (string, error) { return d.inner.GetState(dev) }
func (d dryRunHDDControl) SetStandbyTimeout(dev string, value int) error {
	logrus.Infof("dry-run: would run hdparm -S %d %s", value, dev)
	return nil
}
