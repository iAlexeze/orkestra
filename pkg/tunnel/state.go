// pkg/tunnel/state.go
//
// Tunnel daemon state — persisted to ~/.orkestra/tunnel-state.json.
// Written on Start, read by status/stop commands.
package tunnel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// State records a running tunnel daemon.
type State struct {
	Provider  string    `json:"provider"`
	PID       int       `json:"pid"`
	URL       string    `json:"url"`
	LocalPort int       `json:"localPort"`
	StartedAt time.Time `json:"startedAt"`
}

// stateFile returns the path to the tunnel state file.
func stateFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orkestra", "tunnel-state.json")
}

// SaveState writes the tunnel state to disk.
func SaveState(s State) error {
	path := stateFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadState reads the current tunnel state. Returns nil when no state file exists.
func LoadState() (*State, error) {
	data, err := os.ReadFile(stateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("corrupt tunnel state: %w", err)
	}
	return &s, nil
}

// RemoveState deletes the state file.
func RemoveState() error {
	err := os.Remove(stateFile())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsAlive reports whether the PID in state is still a running process.
func (s *State) IsAlive() bool {
	if s.PID <= 0 {
		return false
	}
	proc, err := os.FindProcess(s.PID)
	if err != nil {
		return false
	}
	// Signal 0 checks process existence without sending a signal.
	return proc.Signal(syscall.Signal(0)) == nil
}

// Stop kills the daemon and removes the state file.
func (s *State) Stop() error {
	if s.PID > 0 {
		proc, err := os.FindProcess(s.PID)
		if err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}
	return RemoveState()
}

// Uptime returns a human-readable duration since the tunnel started.
func (s *State) Uptime() string {
	d := time.Since(s.StartedAt).Round(time.Second)
	if d.Hours() >= 1 {
		return fmt.Sprintf("%.0fh%dm", d.Hours(), int(d.Minutes())%60)
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm%ds", d.Minutes(), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}
