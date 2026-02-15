//go:build !darwin && !linux

package localruntime

import (
	"errors"
	"os"
	"os/exec"
)

func configureProcessGroup(_ *exec.Cmd) {}

func signalProcessTree(pid int, signal os.Signal) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(signal)
}

func killProcessTree(pid int) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

func isProcessAlive(pid int) bool {
	return pid > 0
}
