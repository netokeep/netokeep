package main

import (
	"errors"
	"fmt"
	"netokeep/internal/states"
	"time"

	"github.com/spf13/cobra"
)

func createRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a NetoKeep client with its last settings",
		Long:  "Stop the client if it is running, then start it again with the settings saved from its last start.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			data, err := states.LoadState(name)
			if errors.Is(err, states.ErrNotFound) {
				fmt.Printf("\033[33mNetoKeep client '%s' not found. Use 'nk ls' to see all clients.\033[0m\n", name)
				return
			}
			if err != nil {
				fmt.Printf("\033[31mError checking state for %s: %v\033[0m\n", name, err)
				return
			}
			// Settings come from the state file the core writes on every start
			if data.RemoteAddr == "" || data.SSHPort == 0 {
				fmt.Printf("\033[31mNo saved settings for NetoKeep client '%s'. Start it with: nk start %s -r <remote-link> -s <ssh-port>\033[0m\n", name, name)
				return
			}
			c := clientArgs{
				SSHPort:        data.SSHPort,
				RemoteAddr:     data.RemoteAddr,
				ForwardTraffic: data.ForwardTraffic,
				UseProxy:       data.UseProxy,
			}

			if _, err := states.Stop(name, 15*time.Second); err != nil && !errors.Is(err, states.ErrNotRunning) {
				fmt.Printf("\033[31mFailed to stop NetoKeep client '%s': %v\033[0m\n", name, err)
				return
			} else if err == nil {
				fmt.Printf("NetoKeep client '%s' stopped.\n", name)
			}
			launchClient(cmd.Context(), name, c)
		},
	}
}
