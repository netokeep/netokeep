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
		Use:   "stop",
		Short: "Stop the running NetoKeep server",
		Run: func(cmd *cobra.Command, args []string) {
			data, err := states.Stop(states.ServerName, 15*time.Second)
			switch {
			case errors.Is(err, states.ErrNotRunning):
				fmt.Printf("\033[33mNetoKeep server is not running.\033[0m\n")
			case err != nil:
				fmt.Printf("\033[31mFailed to stop NetoKeep server: %v\033[0m\n", err)
			default:
				fmt.Printf("\033[32mNetoKeep server stopped (PID: %d).\033[0m\n", data.Pid)
			}
		},
	}
}
