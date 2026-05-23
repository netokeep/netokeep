package main

import (
	"fmt"
	"log"
	"netokeep/internal/local"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
)

func createStartCmd() *cobra.Command {
	var portSsh uint16
	var remoteAddr string
	var name string
	var forwardTraffic bool
	var useProxy bool

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep client.",
		Run: func(cmd *cobra.Command, args []string) {
			// Check if the server is already running
			if pid, err := local.ReadPID(name); err == nil {
				if process, err := os.FindProcess(pid); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						fmt.Printf("Netokeep server is already running (PID: %d)\n", pid)
						fmt.Printf("If you want to start one new instance, \n\tplease run the command with '--name' flag and specify a different name.\n")
						return
					}
				}
			}

			// Start the client
			executable, _ := os.Executable()
			argArr := []string{
				"core",
				"-r", remoteAddr,
				"-s", fmt.Sprintf("%d", portSsh),
				"-n", name,
				"-f", fmt.Sprintf("%t", forwardTraffic),
				"-p", fmt.Sprintf("%t", useProxy),
			}
			newCmd := exec.Command(executable, argArr...)
			newCmd.Stdout = nil
			newCmd.Stderr = nil
			newCmd.Stdin = nil

			if err := newCmd.Start(); err != nil {
				log.Fatalf("Failed to start background process: %v", err)
			}

			if err := local.WritePID(name, newCmd.Process.Pid); err != nil {
				log.Printf("Failed to write PID file: %v", err)
			}
			fmt.Printf("Netokeep client started in background (PID: %d)\n", newCmd.Process.Pid)
		},
	}

	startCmd.Flags().StringVarP(&remoteAddr, "remote-address", "r", "", "NKS server address")
	startCmd.Flags().Uint16VarP(&portSsh, "ssh-port", "s", 2222, "SSH port")
	startCmd.Flags().StringVarP(&name, "name", "n", "default", "name of the netokeep client instance")
	startCmd.Flags().BoolVarP(&forwardTraffic, "forward-traffic", "f", false, "forward SSH traffic")
	startCmd.Flags().BoolVarP(&useProxy, "use-proxy", "p", false, "use proxy for outgoing traffic")
	startCmd.MarkFlagRequired("remote-address")

	return startCmd
}
