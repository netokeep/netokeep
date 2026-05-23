package main

import (
	"fmt"
	"netokeep/internal/local"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// CreateStopCmd creates the cobra command to stop the background client
func createStopCmd() *cobra.Command {
	var name string

	var stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {

			// 1. Read the PID from the file
			pid, err := local.ReadPID(name)
			if err != nil {
				fmt.Printf("Error: Could not read PID file: %v\n", err)
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
				// local.RemovePID(name)
				fmt.Println("\033[32mstopped\033[0m")
			} else {
				fmt.Println("\nWarning: Process is taking too long to stop. You might need to kill it manually.")
			}
		},
	}

	stopCmd.Flags().StringVarP(&name, "name", "n", "default", "name of the netokeep client instance")

	return stopCmd
}
