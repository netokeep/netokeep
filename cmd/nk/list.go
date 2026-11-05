package main

import (
	"fmt"
	"netokeep/internal/states"

	"github.com/spf13/cobra"
)

func createListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List all NetoKeep clients",
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			names, err := states.ListClients()
			if err != nil {
				fmt.Printf("\033[31mFailed to list clients: %v\033[0m\n", err)
				return
			}
			if len(names) == 0 {
				fmt.Println("No NetoKeep clients yet. Start one with: nk start <name> -r <remote-link> -s <ssh-port>")
				return
			}

			// Same style as `nk status`, one line per client
			for _, name := range names {
				data, err := states.LoadState(name)
				switch {
				case err != nil:
					fmt.Printf("\033[31m●\033[0m %s - error: %v\n", name, err)
				case states.CheckRunning(data):
					fmt.Printf("\033[32m●\033[0m %s - \033[32mrunning\033[0m (PID: %d, SSH Port: %d)\n", name, data.Pid, data.SSHPort)
				default:
					fmt.Printf("\033[90m●\033[0m %s - %s\n", name, data.State)
				}
			}
		},
	}
}
