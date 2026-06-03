package main

import (
	"fmt"
	"log"
	"netokeep/internal/local"
	"netokeep/pkg/utils"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

func createRestartCmd() *cobra.Command {
	var restartCmd = &cobra.Command{
		Use:     "restart [name]",
		Aliases: []string{"reconnect"},
		Short:   "Restart the netokeep client.",
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "default"
			if len(args) > 0 {
				name = args[0]
			}

			// 1. Read the previous args
			argArr, err := local.ReadArgs(name)
			if err != nil {
				fmt.Printf("Error: Could not read previous arguments for '%s'. Is it configured? (%v)\n", name, err)
				return
			}

			// Extract SSH port directly from the arguments
			var portSsh uint16
			for i, arg := range argArr {
				if arg == "-s" && i+1 < len(argArr) {
					p, _ := strconv.Atoi(argArr[i+1])
					portSsh = uint16(p)
					break
				}
			}

			// 2. Stop the current instance if it's running
			pid, alive := local.IsAlive(name)
			if alive {
				fmt.Printf("Stopping netokeep (PID %d)... ", pid)
				if err := local.Terminate(pid); err != nil {
					fmt.Printf("\nError: Failed to stop process: %v\n", err)
					return
				}

				// Wait for the process to exit
				success := false
				for range 10 {
					if !local.IsPIDAlive(pid) {
						success = true
						break
					}
					time.Sleep(500 * time.Millisecond)
				}

				if success {
					_ = local.RemovePID(name)
					fmt.Println("\033[32mstopped\033[0m")

					time.Sleep(1 * time.Second)
				} else {
					fmt.Println("\nWarning: Process is taking too long to stop. Continuing anyway...")
				}
			}

			// 3. Start the client with previous args
			executable, _ := os.Executable()
			newCmd := exec.Command(executable, argArr...)
			utils.SetupDetachedProcess(newCmd)
			newCmd.Stdout = nil
			newCmd.Stderr = nil
			newCmd.Stdin = nil

			if err := newCmd.Start(); err != nil {
				log.Fatalf("Failed to restart background process: %v", err)
			}

			if err := local.WritePID(name, newCmd.Process.Pid); err != nil {
				log.Printf("Failed to write PID file: %v", err)
			}
			if err := local.WritePort(name+"ssh", portSsh); err != nil {
				log.Printf("Failed to write Port file: %v", err)
			}

			fmt.Printf("Netokeep client restarted in background (PID: %d)\n", newCmd.Process.Pid)
		},
	}

	return restartCmd
}
