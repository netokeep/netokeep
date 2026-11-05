package main

import (
	"errors"
	"fmt"
	"netokeep/internal/states"
	"time"

	"github.com/spf13/cobra"
)

func createStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running NetoKeep client",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			data, err := states.Stop(name, 15*time.Second)
			switch {
			case errors.Is(err, states.ErrNotFound):
				fmt.Printf("\033[33mNetoKeep client '%s' not found. Use 'nk ls' to see all clients.\033[0m\n", name)
			case errors.Is(err, states.ErrNotRunning):
				fmt.Printf("\033[33mNetoKeep client '%s' is not running.\033[0m\n", name)
			case err != nil:
				fmt.Printf("\033[31mFailed to stop NetoKeep client '%s': %v\033[0m\n", name, err)
			default:
				fmt.Printf("\033[32mNetoKeep client '%s' stopped (PID: %d).\033[0m\n", name, data.Pid)
			}
		},
	}
}
