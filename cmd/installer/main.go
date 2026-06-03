package main

import (
	_ "embed"
	"fmt"
	"netokeep/internal/local"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

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
	yellow := color.RGB(255, 204, 0).SprintFunc()

	fmt.Printf("%s\n\n", gray("Setting up NetoKeep..."))
	// Just a small delay to let the user see the message before the installation starts
	time.Sleep(1 * time.Second)

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
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	nkName := "nk" + suffix
	nksName := "nks" + suffix
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
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("%s %s\n\n", gray(" Version: "), orange(version))
	fmt.Printf("%s %s\n\n", gray(" Location:"), orange("~/.local/bin"))
	fmt.Printf("%s %s %s %s %s\n\n",
		gray(" Next: Run"), orange("nk --help"),
		gray("or"), orange("nks --help"),
		gray("to get started."))

	// Verify the installation by checking if the files exist and are on the PATH
	verifyInstalled := func(name string) {
		path := filepath.Join(binDir, name)
		if _, err := os.Stat(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: installed file missing for %s: %v\n", name, err)
			os.Exit(1)
		}
		if _, err := exec.LookPath(name); err != nil {
			fmt.Printf("%s\n",
				yellow(fmt.Sprintf("Warning: %s exists in %s, but is not on your PATH yet.", name, binDir)))
		}
	}
	verifyInstalled(nkName)
	verifyInstalled(nksName)

	// For windows, add 'Press any key to continue...' at the end
	if runtime.GOOS == "windows" {
		fmt.Printf("%s\n", gray("Press any key to continue..."))
		fmt.Scanln()
		os.Exit(0)
	}
	os.Exit(0)
}
