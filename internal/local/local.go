package local

import (
	"os"
	"path/filepath"

	"runtime"
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

func RemoveStateDir() error {
	return os.RemoveAll(stateDir())
}

func RemoveConfigDir() error {
	return os.RemoveAll(configDir())
}

func RemovePrograms() error {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	nkBin := filepath.Join(binDir(), "nk"+ext)
	nksBin := filepath.Join(binDir(), "nks"+ext)
	if err := os.Remove(nkBin); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(nksBin); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
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
