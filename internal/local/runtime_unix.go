//go:build !windows

package local

import (
	"os"
	"syscall"
)

// IsPIDAlive checks whether a process with the given PID is still running.
func IsPIDAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func terminateProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.SIGINT)
}
