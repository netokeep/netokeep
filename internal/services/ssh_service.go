package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	// Route sshd output through log line by line, so it gets a timestamp and prefix.
	// Closed after cmd.Wait returns (defer), which ends the scanner goroutine.
	pr, pw := io.Pipe()
	logDone := make(chan struct{})
	// Close the pipe, then wait until the last lines are logged
	defer func() { pw.Close(); <-logDone }()
	go func() {
		defer close(logDone)
		s := bufio.NewScanner(pr)
		for s.Scan() {
			log.Printf("[sshd] %s", s.Text())
		}
		if err := s.Err(); err != nil {
			log.Printf("[sshd] stopped reading output: %v", err)
			// Keep draining, otherwise sshd blocks writing to a full pipe
			io.Copy(io.Discard, pr)
		}
	}()
	cmd.Stdout = pw
	cmd.Stderr = pw

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
			// sshd will log "Connection closed by <this addr>" for this probe
			log.Printf("[sshd] port %d is up, closing probe connection %s", port, conn.LocalAddr())
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
