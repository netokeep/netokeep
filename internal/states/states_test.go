package states

import (
	"testing"
)

func setupTestPaths(t *testing.T) {
	t.Helper()

	oldHomeDir := homeDir

	dir := t.TempDir()
	homeDir = dir

	t.Cleanup(func() {
		homeDir = oldHomeDir
	})
}

func TestLoadState(t *testing.T) {
	setupTestPaths(t)

	name := "test"
	stateData, err := LoadState(name)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if stateData.State != "unknown" {
		t.Errorf("Expected state 'unknown', got '%s'", stateData.State)
	}
}

func TestWriteState(t *testing.T) {
	setupTestPaths(t)

	name := "test"
	stateData := &StateData{State: "active", Pid: 123, SSHPort: 2222, Inport: 7890, Outport: 7222}
	err := WriteState(name, stateData)
	if err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	retrievedStateData, err := LoadState(name)
	if err != nil {
		t.Fatalf("Load state failed: %v", err)
	}
	if retrievedStateData.State != stateData.State ||
		retrievedStateData.Pid != stateData.Pid ||
		retrievedStateData.SSHPort != stateData.SSHPort ||
		retrievedStateData.Inport != stateData.Inport ||
		retrievedStateData.Outport != stateData.Outport {
		t.Errorf("Expected %+v, got %+v", stateData, retrievedStateData)
	}
}
