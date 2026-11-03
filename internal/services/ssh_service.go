package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func StartSSHService(
	ctx context.Context,
	stop context.CancelFunc,
	ready chan struct{},
	port uint16,
) error {
	// Stop the whole program once sshd exits
	defer stop()

	// Prepare user space sshd configuration directory
	if err := os.MkdirAll("/run/sshd", 0755); err != nil {
		return fmt.Errorf("failed to create '/run/sshd' directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "/usr/sbin/sshd", "-D", "-e",
		"-p", strconv.Itoa(int(port)),
		"-o", "ListenAddress=127.0.0.1",
	)
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start sshd: %w", err)
	}

	if err := waitForPort(ctx, port, 3*time.Second); err != nil {
		cmd.Cancel()
		cmd.Wait()
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("failed to wait for SSH service to be ready: %w", err)
	}

	close(ready)
	log.Println("SSH service is ready")

	err := cmd.Wait()
	if ctx.Err() != nil {
		return nil
	}
	return fmt.Errorf("sshd exited with error: %w", err)
}

func waitForPort(ctx context.Context, port uint16, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port)))
	var d net.Dialer
	for {
		attemptCtx, attemptCancel := context.WithTimeout(ctx, 100*time.Millisecond)
		conn, err := d.DialContext(attemptCtx, "tcp", addr)
		attemptCancel()
		if err == nil {
			conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("port %d not ready: %w", port, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
