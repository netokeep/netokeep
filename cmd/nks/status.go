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
		Use:   "status",
		Short: "Show the status and recent logs of the NetoKeep server",
		Run: func(cmd *cobra.Command, args []string) {
			data, err := states.LoadState(states.ServerName)
			if err != nil {
				fmt.Printf("\033[31mFailed to load server state: %v\033[0m\n", err)
				return
			}
			switch {
			case !states.CheckRunning(data):
				fmt.Printf("\033[90m●\033[0m NetoKeep server - %s\n", data.State)
			case data.InPort == 0:
				fmt.Printf("\033[32m●\033[0m NetoKeep server - \033[32mrunning\033[0m (PID: %d, Out Port: %d)\n", data.Pid, data.OutPort)
			default:
				fmt.Printf("\033[32m●\033[0m NetoKeep server - \033[32mrunning\033[0m (PID: %d, In Port: %d, Out Port: %d)\n", data.Pid, data.InPort, data.OutPort)
			}
			printRecentLogs(states.ServerName, lines)
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
