//go:build !darwin && !linux

package localruntime

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func configureProcessGroup(_ *exec.Cmd) {}

func signalProcessTree(pid int, signal os.Signal) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	if runtime.GOOS == "windows" {
		args := []string{"/PID", strconv.Itoa(pid), "/T"}
		if isForceSignal(signal) {
			args = append(args, "/F")
		}
		return exec.Command("taskkill", args...).Run()
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
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		_, ok := windowsImageByPID(pid)
		return ok
	}
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid=").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func processLooksLikeBinary(pid int, binary string) bool {
	if pid <= 0 {
		return false
	}
	base := strings.ToLower(strings.TrimSpace(filepath.Base(binary)))
	if base == "" {
		base = defaultBinary
	}

	if runtime.GOOS == "windows" {
		image, ok := windowsImageByPID(pid)
		if !ok {
			return false
		}
		normalizedBase := base
		if !strings.HasSuffix(normalizedBase, ".exe") {
			normalizedBase += ".exe"
		}
		normalizedImage := strings.ToLower(strings.TrimSpace(image))
		return normalizedImage == normalizedBase
	}

	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return false
	}
	commandLine := strings.ToLower(strings.TrimSpace(string(output)))
	if commandLine == "" {
		return false
	}
	return strings.Contains(commandLine, base)
}

func windowsImageByPID(pid int) (string, bool) {
	output, err := exec.Command(
		"tasklist",
		"/FI",
		fmt.Sprintf("PID eq %d", pid),
		"/FO",
		"CSV",
		"/NH",
	).Output()
	if err != nil {
		return "", false
	}

	line := strings.TrimSpace(string(output))
	if line == "" || strings.HasPrefix(strings.ToUpper(line), "INFO:") {
		return "", false
	}

	reader := csv.NewReader(strings.NewReader(line))
	record, err := reader.Read()
	if err != nil || len(record) == 0 {
		return "", false
	}

	image := strings.TrimSpace(record[0])
	if image == "" {
		return "", false
	}
	return image, true
}

func isForceSignal(signal os.Signal) bool {
	return signal == os.Kill
}
