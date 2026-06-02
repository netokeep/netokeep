package main

import (
	"fmt"
	"netokeep/internal/local"
	"time"

	"github.com/spf13/cobra"
)

// CreateStopCmd creates the cobra command to stop the background client
func createStopCmd() *cobra.Command {
	var stopCmd = &cobra.Command{
		Use:   "stop [name]",
		Short: "Stop the netokeep client.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			}

			// 1. Read the PID from the file
			pid, err := local.ReadPID(name)
			if err != nil {
				fmt.Printf("Error: Could not read PID file for client '%s'.\n", name)
				return
			}

			// 2. Terminate the process (platform-specific)
			fmt.Printf("Stopping netokeep (PID %d)... ", pid)
			if err := local.Terminate(pid); err != nil {
				fmt.Printf("\nError: Failed to stop process: %v\n", err)
				return
			}

			// 3. Wait for the process to exit (poll PID for up to 5 seconds)
			success := false
			for range 10 {
				if !local.IsPIDAlive(pid) {
					success = true
					break
				}
				time.Sleep(500 * time.Millisecond)
			}

			if success {
				// // Cleanup the PID file after successful stop
				// if err := local.RemovePID(name); err != nil {
				// 	fmt.Printf("Warning: Failed to remove PID file: %v\n", err)
				// }
				// if err := local.RemovePort(name + "ssh"); err != nil {
				// 	fmt.Printf("Warning: Failed to remove Port file: %v\n", err)
				// }
				fmt.Println("\033[32mstopped\033[0m")
			} else {
				fmt.Println("\nWarning: Process is taking too long to stop. You might need to kill it manually.")
			}
		},
	}

	return stopCmd
}
