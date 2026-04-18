package nks

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func CreateStatusCmd() *cobra.Command {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check the status of the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			runDir, err := getXDGDir()
			if err != nil {
				fmt.Printf("● netokeep.service - Netokeep Proxy Server\n")
				fmt.Printf("   Status: Error (Could not locate data directory: %v)\n", err)
				return
			}

			pidPath := filepath.Join(runDir, "netokeep.pid")
			logPath := filepath.Join(runDir, "netokeep.log")

			fmt.Printf("● netokeep.service - Netokeep Proxy Server\n")

			// 1. Check the PID and process
			data, err := os.ReadFile(pidPath)
			isRunning := false
			var pid int

			if err == nil {
				pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
				if process, err := os.FindProcess(pid); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						isRunning = true
					}
				}
			}

			// 2. Pring the systemctl like status information
			if isRunning {
				fmt.Printf("   Active: \033[32mactive (running)\033[0m since %s\n", getFileModTime(pidPath))
				fmt.Printf("     Main PID: %d (netokeep)\n", pid)
			} else {
				fmt.Printf("   Active: \033[31minactive (dead)\033[0m\n")
				if err == nil {
					fmt.Printf("   Notice: Found stale PID file (PID: %d), but process is gone.\n", pid)
				}
			}

			// 3. Log the recent logs (simulating Journal tail output)
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

	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	for _, line := range allLines[start:] {
		if line != "" {
			// Simulate the space like system logging
			fmt.Printf("   %s\n", line)
		}
	}
}
