package utils
import (
	"fmt"
	"os"
	"os/exec"
)

// EnsureRoot checks if the current user is root and auto elevates privileges if not.
func EnsureRoot() {
	// Check if the user is root
	if os.Geteuid() == 0 {
		return
	}

	fmt.Println("Detected non-root user, attempting to auto elevate privileges...")

	// Get the absolute path of the currently executing file
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("Failed to get executable path:", err)
		os.Exit(1)
	}

	// Build the sudo command
	// Equivalent to: sudo USER_CONFIRM=1 /path/to/app arg1 arg2 ...
	// The USER_CONFIRM=1 environment variable indicates that the process has been confirmed by the user
	args := []string{"USER_CONFIRM=1", exe}
	// Preserve original arguments
	args = append(args, os.Args[1:]...)
	cmd := exec.Command("sudo", args...)

	// Connect the new command's input/output to the current terminal
	// This allows the user to enter their password when sudo prompts "Password:"
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the new command and handle the errors
	if err = cmd.Run(); err != nil {
		// If it's an ExitError, we can extract the exit code
		// and exit with the same code
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Println("Failed to elevate privileges:", err)
		os.Exit(1)
	}

	// After the new root process finishes, the old process also exits
	os.Exit(0)
}
