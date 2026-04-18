package nks

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

func getXDGDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "netokeep")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func CreateStartCmd() *cobra.Command {
	var sshPort uint16
	var tcpPort uint16
	var outPort uint16

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the netokeep server.",
		Run: func(cmd *cobra.Command, args []string) {
			runDir, err := getXDGDir()
			if err != nil {
				log.Fatalf("Failed to get XDG directory: %v", err)
			}
			pidPath := filepath.Join(runDir, "netokeep.pid")
			// Check if the PID file exists
			if data, err := os.ReadFile(pidPath); err == nil {
				pid, _ := strconv.Atoi(string(data))
				if process, err := os.FindProcess(pid); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						fmt.Printf("Netokeep server is already running (PID: %d)\n", pid)
						return
					}
				}
			}
			// Start the server
			executable, _ := os.Executable()
			argArr := []string{
				"run",
				"-s", fmt.Sprintf("%d", sshPort),
				"-t", fmt.Sprintf("%d", tcpPort),
				"-o", fmt.Sprintf("%d", outPort),
			}
			newCmd := exec.Command(executable, argArr...)
			newCmd.Stdout = nil
			newCmd.Stderr = nil
			newCmd.Stdin = nil

			if err := newCmd.Start(); err != nil {
				log.Fatalf("Failed to start background process: %v", err)
			}

			os.WriteFile(pidPath, []byte(strconv.Itoa(newCmd.Process.Pid)), 0644)
			fmt.Printf("Netokeep server started in background (PID: %d)\n", newCmd.Process.Pid)
		},
	}

	startCmd.Flags().Uint16VarP(&sshPort, "sshPort", "s", 22, "port to proxy SSH traffic.")
	startCmd.Flags().Uint16VarP(&tcpPort, "tcpPort", "t", 7890, "port to proxy TCP traffic.")
	startCmd.Flags().Uint16VarP(&outPort, "outPort", "o", 7222, "port to forward traffic.")

	return startCmd
}
