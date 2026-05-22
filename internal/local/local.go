package local

import (
	"os"
	"path/filepath"
)

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "netokeep")
}

func stateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "netokeep")
}

func binDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

func InitializeDirs() error {
	for _, d := range []string{configDir(), stateDir(), binDir()} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

func InitialAll() error {
	// Create directories if they don't exist
	if err := InitializeDirs(); err != nil {
		return err
	}

	// Create the config files if not exist
	if _, err := LoadNksConfig(); err != nil {
		return err
	}
	if _, err := LoadNkConfig(); err != nil {
		return err
	}
	return nil
}
