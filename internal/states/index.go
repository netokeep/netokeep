package states

import (
	"encoding/json"
	"errors"
	"log"
	"netokeep/internal/utils"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type StateKind string

const (
	KindClient StateKind = "client"
	KindServer StateKind = "server"
)

type State string

const (
	StateRunning State = "running"
	StateFailed  State = "failed"
	StateStopped State = "stopped"
	StateUnknown State = "unknown"
)

type StateData struct {
	State State `json:"state"`
	Pid   int   `json:"pid"`

	SSHPort        uint16 `json:"ssh_port,omitempty"`
	RemoteAddr     string `json:"remote_addr,omitempty"`
	ForwardTraffic bool   `json:"forward_traffic,omitempty"`
	UseProxy       bool   `json:"use_proxy,omitempty"`

	InPort  uint16 `json:"in_port,omitempty"`
	OutPort uint16 `json:"out_port,omitempty"`
}

var homeDir, _ = os.UserHomeDir()

// ErrNotFound means the state file does not exist: the name was never started, or it was cleared.
var ErrNotFound = errors.New("state not found")

// LoadState reads the state of name. It never creates or rewrites the state file:
// a missing file returns ErrNotFound, and an unreadable one is reported as unknown.
// With ErrNotFound the returned state is still usable (unknown, PID 0), so callers
// that only want to know whether name is running can ignore that error.
func LoadState(name string) (*StateData, error) {
	statePath := filepath.Join(homeDir, ".local", "state", "netokeep", name+"-state.json")
	contents, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StateData{State: StateUnknown}, ErrNotFound
		}
		return nil, err
	}

	stateData := new(StateData)
	if err := json.Unmarshal(contents, stateData); err != nil {
		// Leave the file alone so `nk ls` still shows it and `nk clear` can remove it
		log.Printf("Failed to unmarshal state data of '%s': %v\n", name, err)
		return &StateData{State: StateUnknown}, nil
	}
	return stateData, nil
}

func WriteState(name string, stateData *StateData) error {
	statePath := filepath.Join(homeDir, ".local", "state", "netokeep", name+"-state.json")
	tempPath := statePath + ".tmp"
	jsonBytes, err := json.Marshal(stateData)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(tempPath, jsonBytes, 0644); err != nil {
		return err
	}
	return os.Rename(tempPath, statePath)
}

func ClearState(name string) error {
	statePath := filepath.Join(homeDir, ".local", "state", "netokeep", name+"-state.json")
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func CheckRunning(state *StateData) bool {
	if state.State != StateRunning {
		return false
	}

	if utils.IsProcessRunning(state.Pid) {
		return true
	}

	state.State = StateUnknown
	return false
}

// ServerName is the state name of nks; clients must not use it.
const ServerName = "nks"

// ListClients returns the names of all client states (every state file except the server's), sorted.
func ListClients() ([]string, error) {
	pattern := filepath.Join(homeDir, ".local", "state", "netokeep", "*-state.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		name := strings.TrimSuffix(filepath.Base(m), "-state.json")
		if name != ServerName {
			names = append(names, name)
		}
	}
	// Sort by name, not file name: "cpu-2-state.json" sorts before "cpu-state.json"
	sort.Strings(names)
	return names, nil
}
