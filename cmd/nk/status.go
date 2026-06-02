package main

import (
	"fmt"
	"netokeep/internal/local"
	"netokeep/internal/logging"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func createStatusCmd() *cobra.Command {
	var statusCmd = &cobra.Command{
		Use:   "status [name]",
		Short: "Check the status of the netokeep client.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			}

			pid, err := local.ReadPID(name)
			if err != nil {
				fmt.Printf("● netokeep.service - NetoKeep Proxy Client\n")
				fmt.Printf("   Status: Error (Could not locate PID file.)\n")
				return
			}
			logPath := logging.LogPath(name)

			fmt.Printf("● netokeep.service - NetoKeep Proxy Client\n")

			// Print the systemctl like status information
			if local.IsPIDAlive(pid) {
				fmt.Printf("   Active: \033[32mactive (running)\033[0m since %s\n", getFileModTime(logPath))
				fmt.Printf("     Main PID: %d (netokeep)\n", pid)
			} else {
				fmt.Printf("   Active: \033[31minactive (dead)\033[0m\n")
			}

			// Log the recent logs (simulating Journal tail output)
			fmt.Printf("\nRecent logs:\n")
			printLastLogs(logPath, 10)
		},
	}

	return statusCmd
}

// Get the modification timestamp for one file and convert it to friendly format.
func getFileModTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	return info.ModTime().Format("Mon 2006-01-02 15:04:05 MST")
}

// Read the last N lines of one file.
func printLastLogs(path string, lines int) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("   (No log messages available)")
		return
	}

	content := strings.TrimSpace(string(data))
	allLines := strings.Split(content, "\n")

	start := max(len(allLines)-lines, 0)

	for _, line := range allLines[start:] {
		if line != "" {
			// Simulate the space like system logging
			fmt.Printf("   %s\n", line)
		}
	}
}
