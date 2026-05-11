package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DeployState is the contents of ~/.orkestra/doctor/deploy/state.json.
type DeployState struct {
	ClusterContext string                   `json:"clusterContext"`
	Projects       map[string]*ProjectState `json:"projects"`
	// KatalogHash is the SHA-256 of the last central developer katalog written.
	// Used to detect changes without relying on git, since the katalog lives
	// outside the project repo and is never committed.
	KatalogHash string `json:"katalogHash,omitempty"`
	// DirApps maps an absolute project directory to the ordered list of app names
	// it manages. Used by multi-app compose projects so ork doctor deploy can reconstruct
	// the full app list without reading .init.ork.
	DirApps map[string][]string `json:"dirApps,omitempty"`
}

// ProjectState tracks one deployed project.
type ProjectState struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	CurrentImage  string    `json:"currentImage"`
	PreviousImage string    `json:"previousImage,omitempty"`
	KatalogPath   string    `json:"katalogPath"`
	DeployedAt    time.Time `json:"deployedAt"`

	// Developer path — persisted so the central katalog can be rebuilt on re-deploy.
	AppData       map[string]string `json:"appData,omitempty"`
	Port          string            `json:"port,omitempty"`
	Language      string            `json:"language,omitempty"`
	GitCommit     string            `json:"gitCommit,omitempty"`
	HasDockerfile bool              `json:"hasDockerfile,omitempty"`
	SecretCount   int               `json:"secretCount,omitempty"`
	ConfigCount   int               `json:"configCount,omitempty"`
	HasSecrets    bool              `json:"hasSecrets,omitempty"`
	HasConfig     bool              `json:"hasConfig,omitempty"`

	// Init settings — replaces .orkestra/bundle/.init.ork.
	// Written by ork doctor init, read by ork doctor deploy.
	Dir         string `json:"dir,omitempty"`         // absolute project directory
	UseCompose  bool   `json:"useCompose,omitempty"`  // true when initialised from a compose file
	ComposeFile string `json:"composeFile,omitempty"` // path to docker-compose.yaml
}

// StateDir returns ~/.orkestra/doctor/deploy/
func StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".orkestra", "doctor", "deploy"), nil
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

// DeployedAppNames returns a sorted list of app names recorded in state.
func (s *DeployState) DeployedAppNames() []string {
	names := make([]string, 0, len(s.Projects))
	for name := range s.Projects {
		names = append(names, name)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

// MotifDir returns ~/.orkestra/apps/ — where per-app motif templates are stored.
// Motifs live here rather than in the project directory so developers never see them.
func MotifDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".orkestra", "apps"), nil
}

// MotifPath returns the path to the motif template for a given app name.
// ~/.orkestra/apps/<appname>/motif.yaml
func MotifPath(appName string) (string, error) {
	base, err := MotifDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName, "motif.yaml"), nil
}

// CurrentContext returns the active kubectl context name.
func CurrentContext() string {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// CentralKatalogChanged reads ~/.orkestra/doctor/deploy/katalog.yaml, hashes it, and
// compares with the hash stored in state. Returns true when the katalog is new
// or has changed since the last deploy. Persists the new hash to state so the
// next call returns false unless the content changes again.
//
// This replaces the git-diff-based KatalogChanged for the developer path, since
// the central katalog lives outside the project repo and is never committed.
func CentralKatalogChanged(state *DeployState, deployDir string) bool {
	data, err := os.ReadFile(filepath.Join(deployDir, "katalog.yaml"))
	if err != nil {
		return true // assume changed if we can't read it
	}
	h := sha256.Sum256(data)
	newHash := hex.EncodeToString(h[:])
	if state.KatalogHash == newHash {
		return false
	}
	state.KatalogHash = newHash
	return true
}
