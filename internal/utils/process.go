package utils

import (
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/v4/process"
)

func IsProcessRunning(pid int) bool {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return false
	}
	args, err := p.CmdlineSlice()
	if err != nil || len(args) < 2 {
		return false
	}
	if args[1] != "core" {
		return false
	}

	exe, err1 := p.Exe()
	self, err2 := os.Executable()

	if err1 != nil || err2 != nil {
		return true
	}
	return filepath.Clean(exe) == filepath.Clean(self)
}
