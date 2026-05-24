package main

import (
	"fmt"
	"log"
	"netokeep/internal/local"
	"netokeep/pkg/utils"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func createStartCmd() *cobra.Command {
	var portIn uint16
	var portOut uint16

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep server.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			utils.EnsureRoot()
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Check if the server is already running
			pid, alive := local.IsAlive("nks")
			if alive {
				pidStr := "unknown"
				if pid != 0 {
					pidStr = fmt.Sprintf("%d", pid)
				}
				fmt.Printf("Netokeep server is already running (PID: %s).\n", pidStr)
				return
			}
			// Start the server
			executable, _ := os.Executable()
			argArr := []string{
				"core",
				"-i", fmt.Sprintf("%d", portIn),
				"-o", fmt.Sprintf("%d", portOut),
			}
			newCmd := exec.Command(executable, argArr...)
			newCmd.Stdout = nil
			newCmd.Stderr = nil
			newCmd.Stdin = nil

			if err := newCmd.Start(); err != nil {
				log.Fatalf("Failed to start background process: %v", err)
			}

			if err := local.WritePID("nks", newCmd.Process.Pid); err != nil {
				log.Printf("Failed to write PID file: %v", err)
			}
			fmt.Printf("Netokeep server started in background (PID: %d)\n", newCmd.Process.Pid)
		},
	}

	startCmd.Flags().Uint16VarP(&portIn, "in-port", "i", 7890, "port to proxy incoming traffic (HTTP protocol).")
	startCmd.Flags().Uint16VarP(&portOut, "out-port", "o", 7222, "port to forward outgoing traffic.")

	return startCmd
}
