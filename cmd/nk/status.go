package main

import (
	"fmt"
	"netokeep/internal/logging"
	"netokeep/internal/states"

	"github.com/spf13/cobra"
)

func createStatusCmd() *cobra.Command {
	var lines int

	statusCmd := &cobra.Command{
		Use:   "status <nickname>",
		Short: "Show the status and recent logs of a NetoKeep client",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			data, err := states.LoadState(name)
			if err != nil {
				fmt.Printf("\033[31mFailed to load state for '%s': %v\033[0m\n", name, err)
				return
			}
			if states.CheckRunning(data) {
				fmt.Printf("\033[32m●\033[0m NetoKeep client '%s' - \033[32mrunning\033[0m (PID: %d, SSH Port: %d)\n", name, data.Pid, data.SSHPort)
			} else {
				fmt.Printf("\033[90m●\033[0m NetoKeep client '%s' - %s\n", name, data.State)
			}
			printRecentLogs(name, lines)
		},
	}
	statusCmd.Flags().IntVarP(&lines, "lines", "n", 10, "number of recent log lines to show")
	return statusCmd
}

// printRecentLogs prints the tail of the log file, like `systemctl status` does.
func printRecentLogs(name string, n int) {
	if n <= 0 {
		return
	}
	logs, err := logging.Tail(name, n)
	if err != nil {
		fmt.Printf("\n    (no logs: %v)\n", err)
		return
	}
	fmt.Printf("\n\033[90m%s\033[0m\n", logging.LogPath(name))
	for _, line := range logs {
		fmt.Println(line)
	}
}
