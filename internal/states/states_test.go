package states

import (
	"errors"
	"os"
	"path/filepath"
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
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}
	if stateData == nil || stateData.State != StateUnknown {
		t.Errorf("Expected an unknown state, got %+v", stateData)
	}
	// Loading must not create the state file
	if names, _ := ListClients(); len(names) != 0 {
		t.Errorf("Expected no clients after loading a missing state, got %v", names)
	}
}

func TestLoadStateCorrupt(t *testing.T) {
	setupTestPaths(t)

	path := filepath.Join(homeDir, ".local", "state", "netokeep", "bad-state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{oops"), 0644); err != nil {
		t.Fatal(err)
	}
	stateData, err := LoadState("bad")
	if err != nil || stateData.State != StateUnknown {
		t.Fatalf("Expected unknown state and no error, got %+v, %v", stateData, err)
	}
	// The corrupt file is left for `nk clear` to remove
	if contents, _ := os.ReadFile(path); string(contents) != "{oops" {
		t.Errorf("Corrupt state file was rewritten: %q", contents)
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
