//go:build windows

package local

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// IsPIDAlive checks whether a process with the given PID is still running.
func IsPIDAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var exitCode uint32
	err = windows.GetExitCodeProcess(h, &exitCode)
	if err != nil {
		return false
	}
	return exitCode == 259 // STILL_ACTIVE
}

func removeRunningProgram(path string) error {
	tmpFile := filepath.Join(os.TempDir(), "nk_cleanup.bat")
	script := "@echo off\r\n" +
		":loop\r\n" +
		"ping -n 2 127.0.0.1 >nul\r\n" +
		"del /f \"" + path + "\" 2>nul\r\n" +
		"if exist \"" + path + "\" goto loop\r\n" +
		"del \"%~f0\"\r\n"

	if err := os.WriteFile(tmpFile, []byte(script), 0644); err != nil {
		return err
	}

	cmd := exec.Command("cmd.exe", "/c", tmpFile)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return cmd.Start()
}

func terminateProcess(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.TerminateProcess(h, 0)
}
