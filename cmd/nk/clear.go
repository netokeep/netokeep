package main

import (
	"errors"
	"fmt"
	"netokeep/internal/logging"
	"netokeep/internal/states"
	"slices"

	"github.com/spf13/cobra"
)

func createClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear <name>",
		Short: "Remove the state and logs of a stopped NetoKeep client",
		Long:  "Remove the state and logs of a stopped client. A running client is refused.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			clients, err := states.ListClients()
			if err != nil {
				fmt.Printf("\033[31mFailed to list clients: %v\033[0m\n", err)
				return
			}
			if !slices.Contains(clients, name) {
				fmt.Printf("\033[33mNetoKeep client '%s' not found. Use 'nk ls' to see all clients.\033[0m\n", name)
				return
			}

			data, err := states.LoadState(name)
			if errors.Is(err, states.ErrNotFound) {
				fmt.Printf("\033[33mNetoKeep client '%s' not found. Use 'nk ls' to see all clients.\033[0m\n", name)
				return
			}
			if err != nil {
				fmt.Printf("\033[31mFailed to load state for '%s': %v\033[0m\n", name, err)
				return
			}
			if states.CheckRunning(data) {
				fmt.Printf("\033[33m'%s' is running (PID: %d). Stop it first with: nk stop %s\033[0m\n", name, data.Pid, name)
				return
			}
			if err := states.ClearState(name); err != nil {
				fmt.Printf("\033[31mFailed to clear state for '%s': %v\033[0m\n", name, err)
				return
			}
			if err := logging.ClearLogs(name); err != nil {
				fmt.Printf("\033[31mFailed to clear logs for '%s': %v\033[0m\n", name, err)
				return
			}
			fmt.Printf("\033[32mCleared '%s'.\033[0m\n", name)
		},
	}
}
