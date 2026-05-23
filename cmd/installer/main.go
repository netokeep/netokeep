package main

import (
	_ "embed"
	"fmt"
	"netokeep/internal/local"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

//go:embed nk
var nkBin []byte

//go:embed nks
var nksBin []byte

var version = "dev"

func main() {
	gray := color.RGB(142, 142, 142).SprintFunc()
	greenBold := color.RGB(69, 177, 90).Add(color.Bold).SprintFunc()
	orange := color.RGB(209, 108, 77).SprintFunc()

	fmt.Printf("%s\n\n", gray("Setting up NetoKeep..."))

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

	fmt.Printf("\n%s\n\n", greenBold("✔ NetoKeep successfully installed!"))
	fmt.Printf("%s %s\n\n", gray(" Version: "), orange(version))
	fmt.Printf("%s %s\n\n", gray(" Location:"), orange("~/.local/bin"))
	fmt.Printf("%s %s %s %s %s\n\n",
		gray(" Next: Run"), orange("nk --help"),
		gray("or"), orange("nks --help"),
		gray("to get started."))
}
