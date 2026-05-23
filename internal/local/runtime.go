package local

import (
	"os"
	"path/filepath"
	"strconv"
)

func pidPath(name string) string { return filepath.Join(stateDir(), name+".pid") }

func WritePID(name string, pid int) error {
	return os.WriteFile(pidPath(name), []byte(strconv.Itoa(pid)), 0644)
}

func ReadPID(name string) (int, error) {
	data, err := os.ReadFile(pidPath(name))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func RemovePID(name string) error { return os.Remove(pidPath(name)) }
