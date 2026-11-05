package main

import (
	"context"
	"fmt"
	"netokeep/internal/states"
	"netokeep/internal/utils"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// clientArgs are the settings a client is started with; they are saved in its state file.
type clientArgs struct {
	SSHPort        uint16
	RemoteAddr     string
	ForwardTraffic bool
	UseProxy       bool
}

func createStartCmd() *cobra.Command {
	var c clientArgs

	startCmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start the NetoKeep client",
		Long:  "Start the NetoKeep client, listening for SSH connections and forwarding TCP traffic.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if name == states.ServerName {
				// The server keeps its state under this name in the same directory
				fmt.Printf("\033[31m'%s' is reserved for the NetoKeep server; choose another name.\033[0m\n", name)
				return
			}

			data, err := states.LoadState(name)
			if err != nil {
				fmt.Printf("\033[31mError checking state for %s: %v\033[0m\n", name, err)
				return
			}
			if states.CheckRunning(data) {
				fmt.Printf("\033[32mNetoKeep client '%s' is already running (PID: %d, SSH Port: %d).\033[0m\n", name, data.Pid, data.SSHPort)
				return
			}
			launchClient(cmd.Context(), name, c)
		},
	}

	startCmd.Flags().Uint16VarP(&c.SSHPort, "ssh-port", "s", 0, "ssh port to use")
	startCmd.Flags().StringVarP(&c.RemoteAddr, "remote-addr", "r", "", "remote address to connect to")
	startCmd.Flags().BoolVarP(&c.ForwardTraffic, "forward-traffic", "f", false, "enable traffic forwarding (default false)")
	startCmd.Flags().BoolVarP(&c.UseProxy, "use-proxy", "p", false, "use proxy for connection (default false)")
	startCmd.MarkFlagRequired("remote-addr")
	startCmd.MarkFlagRequired("ssh-port")

	return startCmd
}

// launchClient starts `nk core` in the background and waits until it reports running or failed.
func launchClient(ctx context.Context, name string, c clientArgs) {
	// Create core command args
	executable, _ := os.Executable()
	argv := []string{
		"core",
		name,
		"--ssh-port", fmt.Sprintf("%d", c.SSHPort),
		"--remote-addr", c.RemoteAddr,
	}
	if c.ForwardTraffic {
		argv = append(argv, "--forward-traffic")
	}
	if c.UseProxy {
		argv = append(argv, "--use-proxy")
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
		case <-ctx.Done():
			return
		case <-timer.C:
			fmt.Printf("\033[33mTimeout waiting for NetoKeep client '%s' to start.\033[0m\n", name)
			return
		case <-ticker.C:
			// Check the state periodically
			newData, err := states.LoadState(name)
			if err != nil {
				fmt.Printf("\033[31mError checking state for %s while waiting: %v\033[0m\n", name, err)
				return
			}
			if newData.Pid != newCmd.Process.Pid {
				continue
			}

			switch newData.State {
			case states.StateRunning:
				fmt.Printf("\033[32mNetoKeep client '%s' started successfully (PID: %d, SSH Port: %d).\033[0m\n", name, newData.Pid, newData.SSHPort)
				return
			case states.StateFailed:
				fmt.Printf("\033[31mNetoKeep client '%s' failed to start. Run 'nk status %s' to see the logs.\033[0m\n", name, name)
				return
			default:
				// Unknown state, continue checking
				continue
			}
		}
	}
}
