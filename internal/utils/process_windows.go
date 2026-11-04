//go:build windows

package utils

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
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

// TerminateProcess kills the process. A detached process has no console to receive
// Ctrl+C, so this is a hard kill: deferred cleanup does not run and the state file
// may still say "running" (CheckState detects the dead PID).
func TerminateProcess(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.TerminateProcess(h, 0)
}
