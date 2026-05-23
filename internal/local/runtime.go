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

func portPath(name string) string { return filepath.Join(stateDir(), name+".port") }

func WritePort(name string, port uint16) error {
	return os.WriteFile(portPath(name), []byte(strconv.Itoa(int(port))), 0644)
}

func ReadPort(name string) (uint16, error) {
	data, err := os.ReadFile(portPath(name))
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(string(data))
	return uint16(port), err
}

func RemovePort(name string) error { return os.Remove(portPath(name)) }
