//go:build darwin || linux

package localruntime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

func processLooksLikeBinary(pid int, binary string) bool {
	if pid <= 0 {
		return false
	}
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return false
	}
	commandLine := strings.ToLower(strings.TrimSpace(string(output)))
	if commandLine == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(strings.TrimSpace(binary)))
	if base == "" {
		base = defaultBinary
	}
	if strings.Contains(commandLine, base) {
		return true
	}
	return strings.Contains(commandLine, fmt.Sprintf("/%s", base))
}
