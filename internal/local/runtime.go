package local

import (
	"encoding/json"
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

func argsPath(name string) string { return filepath.Join(stateDir(), name+".args") }

func WriteArgs(name string, args []string) error {
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return os.WriteFile(argsPath(name), data, 0644)
}

func ReadArgs(name string) ([]string, error) {
	data, err := os.ReadFile(argsPath(name))
	if err != nil {
		return nil, err
	}
	var args []string
	err = json.Unmarshal(data, &args)
	return args, err
}

func RemoveArgs(name string) error { return os.Remove(argsPath(name)) }

// IsAlive checks whether a named instance is still running by reading its PID
// and verifying the process exists. Works cross-platform.
func IsAlive(name string) (int, bool) {
	pid, err := ReadPID(name)
	if err != nil {
		return 0, false
	}
	return pid, IsPIDAlive(pid)
}

// Terminate tries graceful shutdown first (SIGINT / CTRL_BREAK_EVENT),
// falling back to force kill (SIGKILL / TerminateProcess).
func Terminate(pid int) error {
	return terminateProcess(pid)
}

// ListClients returns a list of all client names that have a PID file in the state directory.
func ListClients() ([]string, error) {
	files, err := os.ReadDir(stateDir())
	if err != nil {
		return nil, err
	}
	var clients []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".pid" {
			name := file.Name()[:len(file.Name())-4] // Remove .pid extension
			if name != "nks" && name != "sshd" {     // Exclude nks server instance
				clients = append(clients, name)
			}
		}
	}
	return clients, nil
}
