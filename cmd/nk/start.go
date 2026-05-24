package main

import (
	"fmt"
	"log"
	"netokeep/internal/local"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func createStartCmd() *cobra.Command {
	var portSsh uint16
	var remoteAddr string
	var forwardTraffic bool
	var useProxy bool

	var startCmd = &cobra.Command{
		Use:   "start [name]",
		Short: "Start the netokeep client.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			}

			// Check if the client is already running
			pid, alive := local.IsAlive(name)
			if alive {
				portSsh, err := local.ReadPort(name + "ssh")
				if err == nil {
					fmt.Printf("Netokeep client is already running (PID: %d, SSH Port: %d)\n", pid, portSsh)
				} else {
					fmt.Printf("Netokeep client is already running (PID: %d)\n", pid)
				}
				fmt.Printf("If you want to start one new instance, \n")
				fmt.Printf("run 'nk start <name> -s <port>' with a different name and port.\n")
				return
			}

			// Start the client
			executable, _ := os.Executable()
			argArr := []string{
				"core",
				name,
				"-r", remoteAddr,
				"-s", fmt.Sprintf("%d", portSsh),
			}
			if forwardTraffic {
				argArr = append(argArr, "-f")
			}
			if useProxy {
				argArr = append(argArr, "-p")
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
			if err := local.WritePort(name+"ssh", portSsh); err != nil {
				log.Printf("Failed to write Port file: %v", err)
			}
			fmt.Printf("Netokeep client started in background (PID: %d)\n", newCmd.Process.Pid)
		},
	}

	startCmd.Flags().StringVarP(&remoteAddr, "remote-address", "r", "", "NKS server address")
	startCmd.Flags().Uint16VarP(&portSsh, "ssh-port", "s", 2222, "SSH port")
	startCmd.Flags().BoolVarP(&forwardTraffic, "forward-traffic", "f", false, "forward SSH traffic")
	startCmd.Flags().BoolVarP(&useProxy, "use-proxy", "p", false, "use proxy for outgoing traffic")
	startCmd.MarkFlagRequired("remote-address")

	return startCmd
}
