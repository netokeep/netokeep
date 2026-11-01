package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestPaths(t *testing.T) {
	t.Helper()

	oldClientPath := clientConfPath
	oldServerPath := serverConfPath

	dir := t.TempDir()
	clientConfPath = filepath.Join(dir, "nk_settings.json")
	serverConfPath = filepath.Join(dir, "nk_server.json")

	t.Cleanup(func() {
		clientConfPath = oldClientPath
		serverConfPath = oldServerPath
	})
}

func TestLoadClientConf(t *testing.T) {
	setupTestPaths(t)
	_, err := LoadClientConf()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestLoadServerConf(t *testing.T) {
	setupTestPaths(t)
	_, err := LoadServerConf()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSaveClientConf(t *testing.T) {
	setupTestPaths(t)
	err := SaveClientConf(&ClientConf{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSaveServerConf(t *testing.T) {
	setupTestPaths(t)
	err := SaveServerConf(&ServerConf{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClearClientConf(t *testing.T) {
	setupTestPaths(t)
	if err := SaveClientConf(&ClientConf{}); err != nil {
		t.Fatal(err)
	}

	if err := ClearClientConf(); err != nil {
		t.Fatal(err)
	}

	_, err := os.Stat(clientConfPath)
	if err == nil {
		t.Fatalf("Expected file to be removed, got %v", err)
	}
}

func TestClearServerConf(t *testing.T) {
	setupTestPaths(t)
	if err := SaveServerConf(&ServerConf{}); err != nil {
		t.Fatal(err)
	}

	if err := ClearServerConf(); err != nil {
		t.Fatal(err)
	}

	_, err := os.Stat(serverConfPath)
	if err == nil {
		t.Fatalf("Expected file to be removed, got %v", err)
	}
}
