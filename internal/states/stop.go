package states

import (
	"errors"
	"fmt"
	"netokeep/internal/utils"
	"time"
)

var ErrNotRunning = errors.New("not running")

// Stop asks the core process recorded under name to shut down and waits until it exits.
// On Unix the core gets SIGINT and shuts down gracefully; on Windows it is killed outright,
// so the state is marked stopped here because the core cannot do it itself.
func Stop(name string, timeout time.Duration) (*StateData, error) {
	data, err := LoadState(name)
	if err != nil {
		return nil, err
	}
	if !CheckRunning(data) {
		return data, ErrNotRunning
	}

	if err := utils.TerminateProcess(data.Pid); err != nil {
		return data, fmt.Errorf("failed to signal process %d: %w", data.Pid, err)
	}
	deadline := time.Now().Add(timeout)
	for utils.IsProcessRunning(data.Pid) {
		if time.Now().After(deadline) {
			return data, fmt.Errorf("process %d did not exit within %v", data.Pid, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}

	if cur, err := LoadState(name); err == nil && cur.State == StateRunning && cur.Pid == data.Pid {
		cur.State = StateStopped
		if err := WriteState(name, cur); err != nil {
			return data, fmt.Errorf("stopped, but failed to update state: %w", err)
		}
	}
	return data, nil
}
