package doktor

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DeployState is the contents of ~/.orkestra/deploy/state.json.
type DeployState struct {
	ClusterContext string                   `json:"clusterContext"`
	Projects       map[string]*ProjectState `json:"projects"`
}

// ProjectState tracks one deployed project.
type ProjectState struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	CurrentImage  string    `json:"currentImage"`
	PreviousImage string    `json:"previousImage,omitempty"`
	KatalogPath   string    `json:"katalogPath"`
	DeployedAt    time.Time `json:"deployedAt"`
}

// StateDir returns ~/.orkestra/deploy/
func StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".orkestra", "deploy"), nil
}

// LoadState reads the state file. Returns an empty state if not found.
func LoadState() (*DeployState, error) {
	dir, err := StateDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if os.IsNotExist(err) {
		return &DeployState{Projects: make(map[string]*ProjectState)}, nil
	}
	if err != nil {
		return nil, err
	}

	var state DeployState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.Projects == nil {
		state.Projects = make(map[string]*ProjectState)
	}
	return &state, nil
}

// Save writes the state file atomically.
func (s *DeployState) Save() error {
	dir, err := StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "state.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "state.json"))
}

// RecordDeploy captures the current image as previous before recording the new one.
// Call this BEFORE patching the cluster so previousImage is always the live image.
func (s *DeployState) RecordDeploy(appName, namespace, katalogPath, newImage string) {
	existing, ok := s.Projects[appName]
	if ok && existing.CurrentImage != "" && existing.CurrentImage != newImage {
		existing.PreviousImage = existing.CurrentImage
	}
	if !ok {
		existing = &ProjectState{Name: appName, Namespace: namespace}
	}
	existing.CurrentImage = newImage
	existing.KatalogPath = katalogPath
	existing.DeployedAt = time.Now()
	s.Projects[appName] = existing
}

// PreviousImage returns the image deployed before the last deploy, or "" if none.
func (s *DeployState) PreviousImage(appName string) string {
	if p, ok := s.Projects[appName]; ok {
		return p.PreviousImage
	}
	return ""
}

// CurrentContext returns the active kubectl context name.
func CurrentContext() string {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
