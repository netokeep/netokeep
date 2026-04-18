package nks

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// CreateStopCmd creates the cobra command to stop the background server
func CreateStopCmd() *cobra.Command {
	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			runDir, err := getXDGDir()
			if err != nil {
				fmt.Printf("Error: Failed to get data directory: %v\n", err)
				return
			}

			pidPath := filepath.Join(runDir, "netokeep.pid")

			// 1. Read the PID from the file
			data, err := os.ReadFile(pidPath)
			if err != nil {
				fmt.Println("Error: Netokeep is not running (PID file not found).")
				return
			}

			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				fmt.Printf("Error: Invalid PID found in %s\n", pidPath)
				return
			}

			// 2. Find the process
			process, err := os.FindProcess(pid)
			if err != nil {
				fmt.Printf("Error: Could not find process with PID %d\n", pid)
				return
			}

			// 3. Send SIGTERM (graceful shutdown)
			fmt.Printf("Stopping netokeep (PID %d)... ", pid)
			err = process.Signal(syscall.SIGTERM)
			if err != nil {
				fmt.Printf("\nError: Failed to send shutdown signal: %v\n", err)
				return
			}

			// 4. Wait a moment for the process to exit and cleanup the PID file
			// We poll the process state for up to 5 seconds
			success := false
			for range 10 {
				// Signal(0) checks if the process still exists
				if err := process.Signal(syscall.Signal(0)); err != nil {
					success = true
					break
				}
				time.Sleep(500 * time.Millisecond)
			}

			if success {
				// Cleanup the PID file after successful stop
				os.Remove(pidPath)
				fmt.Println("\033[32mstopped\033[0m")
			} else {
				fmt.Println("\nWarning: Process is taking too long to stop. You might need to kill it manually.")
			}
		},
	}

	return stopCmd
}
