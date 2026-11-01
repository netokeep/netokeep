//go:build !windows

package utils

import (
	"os/exec"
	"syscall"
)

// SetupDetachedProcess configures the given command to run as a detached process.
func SetupDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}
