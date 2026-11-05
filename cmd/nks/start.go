package main

import (
	"fmt"
	"netokeep/internal/states"
	"netokeep/internal/utils"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

func createStartCmd() *cobra.Command {
	var inPort uint16
	var outPort uint16

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the NetoKeep server.",
		Long:  "Start the NetoKeep server, listening for incoming and outgoing traffic on specified ports.",
		Run: func(cmd *cobra.Command, args []string) {
			if os.Geteuid() != 0 {
				fmt.Printf("\033[31mThis command must be run as root.\033[0m\n")
				return
			}

			data, err := states.LoadState(states.ServerName)
			if err != nil {
				fmt.Printf("\033[31mFailed to load current server state: %v\033[0m\n", err)
				return
			}
			if states.CheckRunning(data) {
				if data.InPort == 0 {
					fmt.Printf("\033[32mNetoKeep server is already running (PID: %d, Out Port: %d).\033[0m\n", data.Pid, data.OutPort)
					return
				} else {
					fmt.Printf("\033[32mNetoKeep server is already running (PID: %d, In Port: %d, Out Port: %d).\033[0m\n", data.Pid, data.InPort, data.OutPort)
					return
				}
			}

			// Create core command args
			executable, _ := os.Executable()
			argv := []string{
				"core",
				"-i", fmt.Sprintf("%d", inPort),
				"-o", fmt.Sprintf("%d", outPort),
			}

			// Execute the core command
			newCmd := exec.Command(executable, argv...)
			utils.SetupDetachedProcess(newCmd)

			if err := newCmd.Start(); err != nil {
				fmt.Printf("\033[31mFailed to start background process: %v\033[0m\n", err)
				return
			}

			// Check the state after starting the background process
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			timer := time.NewTimer(10 * time.Second)
			defer timer.Stop()

			for {
				select {
				case <-cmd.Context().Done():
					return
				case <-timer.C:
					fmt.Printf("\033[33mTimeout waiting for NetoKeep server to start.\033[0m\n")
					return
				case <-ticker.C:
					// Check the state periodically
					newData, err := states.LoadState(states.ServerName)
					if err != nil {
						fmt.Printf("\033[31mError checking state while waiting: %v\033[0m\n", err)
						return
					}
					if newData.Pid != newCmd.Process.Pid {
						continue
					}

					switch newData.State {
					case states.StateRunning:
						fmt.Printf("\033[32mNetoKeep server started successfully (PID: %d, In Port: %d, Out Port: %d).\033[0m\n", newData.Pid, newData.InPort, newData.OutPort)
						return
					case states.StateFailed:
						fmt.Printf("\033[31mNetoKeep server failed to start.\033[0m\n")
						return
					default:
						// Unknown state, continue checking
						continue
					}
				}
			}
		},
	}

	startCmd.Flags().Uint16VarP(&inPort, "in-port", "i", 0, "port to proxy incoming traffic (HTTP protocol).")
	startCmd.Flags().Uint16VarP(&outPort, "out-port", "o", 7222, "port to forward outgoing traffic.")

	return startCmd
}
