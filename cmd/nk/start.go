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
	var sshPort uint16
	var remoteAddr string
	var forwardTraffic bool
	var useProxy bool

	startCmd := &cobra.Command{
		Use:   "start <nickname>",
		Short: "Start the NetoKeep client",
		Long:  "Start the NetoKeep client, listening for SSH connections and forwarding TCP traffic.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			data, err := states.CheckState(name)
			if err != nil {
				fmt.Printf("\033[31mError checking state for %s: %v\033[0m\n", name, err)
				return
			}
			if data.State == states.StateRunning {
				fmt.Printf("\033[32mNetoKeep client '%s' is already running (PID: %d, SSH Port: %d).\033[0m\n", name, data.Pid, data.SshPort)
				return
			}

			// Create core command args
			executable, _ := os.Executable()
			argv := []string{
				"core",
				name,
				"--ssh-port", fmt.Sprintf("%d", sshPort),
				"--remote-addr", remoteAddr,
			}
			if forwardTraffic {
				argv = append(argv, "--forward-traffic")
			}
			if useProxy {
				argv = append(argv, "--use-proxy")
			}

			// Execute the core command
			newCmd := exec.Command(executable, argv...)
			utils.SetupDetachedProcess(newCmd)

			if err := newCmd.Start(); err != nil {
				fmt.Printf("\033[31mFailed to start background process: %v\033[0m\n", err)
				return
			}

			// Check the state after the state changes.
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			timer := time.NewTimer(10 * time.Second)
			defer timer.Stop()

			for {
				select {
				case <-cmd.Context().Done():
					return
				case <-timer.C:
					fmt.Printf("\033[33mTimeout waiting for NetoKeep client '%s' to start.\033[0m\n", name)
					return
				case <-ticker.C:
					// Check the state periodically
					newData, err := states.CheckState(name)
					if err != nil {
						fmt.Printf("\033[31mError checking state for %s while waiting: %v\033[0m\n", name, err)
						return
					}
					if newData.Pid != newCmd.Process.Pid {
						continue
					}

					switch newData.State {
					case states.StateRunning:
						fmt.Printf("\033[32mNetoKeep client '%s' started successfully (PID: %d, SSH Port: %d).\033[0m\n", name, newData.Pid, newData.SshPort)
						return
					case states.StateFailed:
						fmt.Printf("\033[31mNetoKeep client '%s' failed to start.\033[0m\n", name)
						return
					default:
						// Unknown state, continue checking
						continue
					}
				}
			}
		},
	}

	startCmd.Flags().Uint16VarP(&sshPort, "ssh-port", "s", 0, "ssh port to use")
	startCmd.Flags().StringVarP(&remoteAddr, "remote-addr", "r", "", "remote address to connect to")
	startCmd.Flags().BoolVarP(&forwardTraffic, "forward-traffic", "f", false, "enable traffic forwarding (default false)")
	startCmd.Flags().BoolVarP(&useProxy, "use-proxy", "p", false, "use proxy for connection (default false)")
	startCmd.MarkFlagRequired("remote-addr")
	startCmd.MarkFlagRequired("ssh-port")

	return startCmd
}
