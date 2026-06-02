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
				portIn, errIn := local.ReadPort("nks-in")
				portOut, errOut := local.ReadPort("nks-out")
				if errIn == nil && errOut == nil {
					fmt.Printf("NetoKeep server is already running (PID: %d, In Port: %d, Out Port: %d).\n", pid, portIn, portOut)
				} else {
					fmt.Printf("NetoKeep server is already running (PID: %d).\n", pid)
				}
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
			utils.SetupDetachedProcess(newCmd)
			newCmd.Stdout = nil
			newCmd.Stderr = nil
			newCmd.Stdin = nil

			if err := newCmd.Start(); err != nil {
				log.Fatalf("Failed to start background process: %v", err)
			}

			if err := local.WritePID("nks", newCmd.Process.Pid); err != nil {
				log.Printf("Failed to write PID file: %v", err)
			}
			if err := local.WritePort("nks-in", portIn); err != nil {
				log.Printf("Failed to write Port file: %v", err)
			}
			if err := local.WritePort("nks-out", portOut); err != nil {
				log.Printf("Failed to write Port file: %v", err)
			}
			fmt.Printf("NetoKeep server started in background (PID: %d)\n", newCmd.Process.Pid)
		},
	}

	startCmd.Flags().Uint16VarP(&portIn, "in-port", "i", 7890, "port to proxy incoming traffic (HTTP protocol).")
	startCmd.Flags().Uint16VarP(&portOut, "out-port", "o", 7222, "port to forward outgoing traffic.")

	return startCmd
}
