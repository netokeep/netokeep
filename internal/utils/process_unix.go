//go:build !windows

package utils

import (
	"os"
	"os/exec"
	"syscall"
)

// SetupDetachedProcess configures the given command to run as a detached process.
func SetupDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

// TerminateProcess asks the process to stop; core handles SIGINT and shuts down gracefully.
func TerminateProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGINT)
}
