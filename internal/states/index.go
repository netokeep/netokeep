package states

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type StateData struct {
	State   string
	Pid     int
	SshPort uint16
}

var homeDir, _ = os.UserHomeDir()
var defaultStateData = &StateData{State: "unknown"}

func CheckState(name string) (*StateData, error) {
	// read state file
	statePath := filepath.Join(homeDir, ".local", "state", name+"-state.json")
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
	statePath := filepath.Join(homeDir, ".local", "state", name+"-state.json")
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
