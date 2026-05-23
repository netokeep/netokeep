package services

import (
	"fmt"
	"log"
	"netokeep/internal/local"
	"netokeep/pkg/utils"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func StartSshdService() (uint16, func(), error) {
	if sshPid, err := local.ReadPID("sshd"); err == nil {
		if process, err := os.FindProcess(sshPid); err == nil {
			if err := process.Signal(syscall.Signal(0)); err == nil {
				// If service is already running, return the existing port
				port, err := local.ReadPort("sshd")
				if err != nil {
					log.Printf("[sshd] Error in reading SSH port: %v\n", err)
				}
				portStr := "unknown"
				if err == nil && port != 0 {
					portStr = strconv.FormatUint(uint64(port), 10)
				}
				log.Printf("[sshd] SSHD service is already running (PID: %d, Port: %s)\n", sshPid, portStr)
				return port, cleanupFunc(sshPid), nil
			}
		}
	}

	// Find an available free port from a high range
	port, err := utils.FindFreePort()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to find free port for sshd: %w", err)
	}

	// Prepare user space sshd configuration directory
	if err := os.MkdirAll("/run/sshd", 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create '/run/sshd' directory: %w", err)
	}

	// Start the sshd process
	cmd := exec.Command("/usr/sbin/sshd", "-D", "-e", "-p", strconv.Itoa(int(port)))
	cmd.Stdout = logWriter{prefix: "sshd"}
	cmd.Stderr = logWriter{prefix: "sshd"}

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		return 0, nil, fmt.Errorf("failed to start sshd process: %w", err)
	}

	// Record the PID and Port
	if err := local.WritePID("sshd", cmd.Process.Pid); err != nil {
		cmd.Process.Kill()
		return 0, nil, fmt.Errorf("failed to write sshd PID: %w", err)
	}
	if err := local.WritePort("sshd", uint16(port)); err != nil {
		cmd.Process.Kill()
		return 0, nil, fmt.Errorf("failed to write sshd port: %w", err)
	}

	log.Printf("🌐 SSHD service started (PID: %d, Port: %d)\n", cmd.Process.Pid, port)

	return uint16(port), cleanupFunc(cmd.Process.Pid), nil
}

func cleanupFunc(sshPid int) func() {
	return func() {
		log.Printf("[sshd] Stopping sshd process (PID: %d)...", sshPid)
		process, err := os.FindProcess(sshPid)
		if err != nil {
			log.Printf("[sshd] Failed to find sshd process with PID %d: %v", sshPid, err)
			return
		}
		if err := process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("[sshd] Failed to send SIGTERM to sshd process with PID %d: %v", sshPid, err)
			return
		}
		if err := local.RemovePID("sshd"); err != nil {
			log.Printf("[sshd] Failed to remove sshd PID: %v", err)
		}
		if err := local.RemovePort("sshd"); err != nil {
			log.Printf("[sshd] Failed to remove sshd port: %v", err)
		}
	}
}

type logWriter struct {
	prefix string
}

func (w logWriter) Write(p []byte) (int, error) {
	log.Printf("[%s] %s", w.prefix, strings.TrimSpace(string(p)))
	return len(p), nil
}
