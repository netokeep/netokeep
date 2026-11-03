//go:build windows

package utils

import (
	"os/exec"
	"syscall"
)

const (
	DETACHED_PROCESS         = 0x00000008
	CREATE_NEW_PROCESS_GROUP = 0x00000200
)

// SetupDetachedProcess configures the given command to run as a detached process.
func SetupDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
}
