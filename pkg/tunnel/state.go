// pkg/tunnel/state.go
//
// Multi-tunnel daemon state — persisted to ~/.orkestra/doctor/tunnel/tunnel-state.json
// as a map[name]State so multiple tunnels (per app, controlcenter) coexist.
package tunnel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// State records one running tunnel daemon.
type State struct {
	Name           string    `json:"name"`
	Provider       string    `json:"provider"`
	PID            int       `json:"pid"`
	PortForwardPID int       `json:"portForwardPid,omitempty"`
	URL            string    `json:"url"`
	LocalPort      int       `json:"localPort"`
	StartedAt      time.Time `json:"startedAt"`
}

func stateFile() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".orkestra", "doctor", "tunnel")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "tunnel-state.json")
}

// SaveTunnelState writes or updates the named tunnel entry.
func SaveTunnelState(name string, s State) error {
	states, _ := loadRawStates()
	if states == nil {
		states = make(map[string]State)
	}
	s.Name = name
	states[name] = s
	return writeStates(states)
}

// LoadAllStates reads all persisted tunnel entries.
func LoadAllStates() (map[string]State, error) {
	return loadRawStates()
}

// LoadTunnelState returns the state for name, or nil when not found.
func LoadTunnelState(name string) (*State, error) {
	states, err := loadRawStates()
	if err != nil {
		return nil, err
	}
	if s, ok := states[name]; ok {
		s.Name = name
		return &s, nil
	}
	return nil, nil
}

// RemoveTunnelState removes one entry from the state map.
// The file is deleted entirely when the map becomes empty.
func RemoveTunnelState(name string) error {
	states, _ := loadRawStates()
	if states == nil {
		return nil
	}
	delete(states, name)
	if len(states) == 0 {
		err := os.Remove(stateFile())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return writeStates(states)
}

// RemoveAllStates deletes the tunnel state file.
func RemoveAllStates() error {
	err := os.Remove(stateFile())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func loadRawStates() (map[string]State, error) {
	data, err := os.ReadFile(stateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var states map[string]State
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, fmt.Errorf("corrupt tunnel state: %w", err)
	}
	return states, nil
}

func writeStates(states map[string]State) error {
	path := stateFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// IsAlive reports whether the tunnel process is still running.
func (s *State) IsAlive() bool {
	if s.PID <= 0 {
		return false
	}
	proc, err := os.FindProcess(s.PID)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// Stop kills the tunnel daemon (and any port-forward) and removes the state entry.
func (s *State) Stop() error {
	if s.PortForwardPID > 0 {
		if proc, err := os.FindProcess(s.PortForwardPID); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}
	if s.PID > 0 {
		if proc, err := os.FindProcess(s.PID); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}
	return RemoveTunnelState(s.Name)
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
