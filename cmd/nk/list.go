package main

import (
	"fmt"
	"netokeep/internal/local"

	"github.com/spf13/cobra"
)

func createListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all recorded clients.",
		Run: func(cmd *cobra.Command, args []string) {
			clients, err := local.ListClients()
			if err != nil {
				fmt.Printf("Error listing clients: %v\n", err.Error())
				return
			}

			maxLen := 0
			for _, client := range clients {
				if len(client) > maxLen {
					maxLen = len(client)
				}
			}

			if len(clients) == 0 {
				fmt.Println("No clients found.")
				return
			}

			fmt.Printf("Clients:\n")
			for _, client := range clients {
				fmt.Printf("  %-*s: ", maxLen, client)
				if pid, alive := local.IsAlive(client); alive {
					fmt.Printf("\033[32mactive (running, PID: %d)\033[0m\n", pid)
				} else {
					fmt.Printf("\033[31minactive (dead)\033[0m\n")
				}
			}

		},
	}
}
