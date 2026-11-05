package states

import (
	"encoding/json"
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
var defaultStateData = &StateData{State: StateUnknown}

func LoadState(name string) (*StateData, error) {
	// read state file
	statePath := filepath.Join(homeDir, ".local", "state", "netokeep", name+"-state.json")
	contents, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := WriteState(name, defaultStateData); err != nil {
				log.Printf("Failed to write default state data: %v\n", err)
				return nil, err
			}
			return defaultStateData, nil
		}
		return nil, err
	}

	// decode the JSON content from the file into a StateData struct
	stateData := new(StateData)
	if err := json.Unmarshal(contents, stateData); err != nil {
		log.Println("Failed to unmarshal state data. Overwriting with default.")
		if err := WriteState(name, defaultStateData); err != nil {
			log.Printf("Failed to write default state data: %v\n", err)
			return nil, err
		}
		return defaultStateData, nil
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
