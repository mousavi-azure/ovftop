package deploy

import (
	"fmt"
	"os/exec"
)

// LocateOVFTool finds the ovftool binary on PATH, returning a clear error
// if it isn't installed.
func LocateOVFTool() (string, error) {
	path, err := exec.LookPath("ovftool")
	if err != nil {
		return "", fmt.Errorf("ovftool not found on PATH: install VMware OVF Tool to deploy (%w)", err)
	}
	return path, nil
}
