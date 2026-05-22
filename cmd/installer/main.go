package main

import (
	_ "embed"
	"fmt"
	"netokeep/internal/local"
	"os"
	"path/filepath"
)

//go:embed nk
var nkBin []byte

//go:embed nks
var nksBin []byte

func main() {
	fmt.Println("🚀 Starting installation of NetoKeep...")

	// Fetch the bin directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot find home directory: %v\n", err)
		os.Exit(1)
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create bin directory: %v\n", err)
		os.Exit(1)
	}

	// Copy the binaries to the bin directory
	nkName := "nk"
	nksName := "nks"
	install := func(name string, data []byte) {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, data, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to install %s: %v\n", name, err)
			os.Exit(1)
		}
	}
	install(nkName, nkBin)
	install(nksName, nksBin)

	// Initialize the config and state directories
	local.InitialAll()

	fmt.Println("✅ Installation completed successfully!")
	fmt.Println("👉 Now use 'nk' and 'nks' anywhere in your shell.")
}
