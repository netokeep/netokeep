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
	if stateData.State != StateUnknown {
		t.Errorf("Expected state '%s', got '%s'", StateUnknown, stateData.State)
	}
}

func TestWriteState(t *testing.T) {
	setupTestPaths(t)

	name := "test"
	stateData := &StateData{State: StateRunning, Pid: 123, SSHPort: 2222, InPort: 7890, OutPort: 7222}
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
		retrievedStateData.InPort != stateData.InPort ||
		retrievedStateData.OutPort != stateData.OutPort {
		t.Errorf("Expected %+v, got %+v", stateData, retrievedStateData)
	}
}
