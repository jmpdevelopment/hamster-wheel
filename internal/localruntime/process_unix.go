//go:build darwin || linux

package localruntime

import (
	"os"
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessTree(pid int, signal os.Signal) error {
	sig, ok := signal.(syscall.Signal)
	if !ok {
		return syscall.EINVAL
	}
	if pid <= 0 {
		return syscall.EINVAL
	}
	return syscall.Kill(-pid, sig)
}

func killProcessTree(pid int) error {
	if pid <= 0 {
		return syscall.EINVAL
	}
	return syscall.Kill(-pid, syscall.SIGKILL)
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
