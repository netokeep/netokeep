package main

import (
	"fmt"
	"netokeep/internal/local"
	"netokeep/internal/logging"
	"os"

	"github.com/spf13/cobra"
)

func createClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear [name]",
		Aliases: []string{"remove", "rm", "delete", "del", "clean"},
		Short: "Clear recorded clients and logs.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "__None__"
			if len(args) > 0 {
				name = args[0]
			}
			if name == "__None__" {
				fmt.Printf("You should specify a client name to clear.\n\nUse 'nk ls' to see available clients.\n")
				return
			}

			pid, err := local.ReadPID(name)
			if err != nil {
				fmt.Printf("Client '%s' not found: %v\n", name, err)
				return
			}

			if alive := local.IsPIDAlive(pid); alive {
				fmt.Printf("Error: Client '%s' is currently active (PID %d).\n\nPlease stop it before clearing.\n", name, pid)
				return
			}

			if err := local.RemovePID(name); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: Failed to remove PID file for client '%s': %v\n", name, err)
			}

			if err := local.RemovePort(name + "ssh"); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: Failed to remove SSH port file for client '%s': %v\n", name, err)
			}

			if err := local.RemoveArgs(name); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: Failed to remove Args file for client '%s': %v\n", name, err)
			}

			if err := logging.ClearLogs(name); err != nil {
				fmt.Printf("Warning: Failed to clear logs for client '%s': %v\n", name, err)
			}

			fmt.Printf("'%s' cleared successfully.\n", name)
		},
	}
}
