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

func TestCheckState(t *testing.T) {
	setupTestPaths(t)

	name := "test"
	stateData, err := CheckState(name)
	if err != nil {
		t.Fatalf("CheckState failed: %v", err)
	}
	if stateData.State != "unknown" {
		t.Errorf("Expected state 'unknown', got '%s'", stateData.State)
	}
}

func TestWriteState(t *testing.T) {
	setupTestPaths(t)

	name := "test"
	stateData := &StateData{State: "active", Pid: 123, SshPort: 2222}
	err := WriteState(name, stateData)
	if err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	retrievedStateData, err := CheckState(name)
	if err != nil {
		t.Fatalf("CheckState failed: %v", err)
	}
	if retrievedStateData.State != stateData.State || retrievedStateData.Pid != stateData.Pid || retrievedStateData.SshPort != stateData.SshPort {
		t.Errorf("Expected %+v, got %+v", stateData, retrievedStateData)
	}
}
