package buildx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppEntry represents one application in a multi-app project.
type AppEntry struct {
	Name       string // app name — used for image tag, CR name, and katalog dir
	Dir        string // absolute path to the app's build context directory
	Dockerfile string // path to Dockerfile (absolute or relative); empty = "Dockerfile" in Dir
}

// InitConfig is the full configuration persisted in .orkestra/bundle/.init.ork.
// It bridges `ork doctor init` settings through to `ork doctor deploy`.
type InitConfig struct {
	UseCompose  bool       // true → compose-based build
	ComposeFile string     // path to docker-compose.yaml (only when UseCompose)
	Apps        []AppEntry // one entry per buildable app; empty = legacy single-app
}

// WriteInitConfig persists a single-app or compose config to .orkestra/bundle/.init.ork.
func WriteInitConfig(dir string, useCompose bool, composeFile string) error {
	return WriteInitConfigFull(dir, InitConfig{
		UseCompose:  useCompose,
		ComposeFile: composeFile,
	})
}

// WriteInitConfigFull writes the full InitConfig to .orkestra/bundle/.init.ork.
func WriteInitConfigFull(dir string, cfg InitConfig) error {
	path := GetInitConfigPath(dir)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf(".init.ork: cannot create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(".init.ork: cannot write file: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "useCompose=%t\n", cfg.UseCompose)
	if cfg.UseCompose && cfg.ComposeFile != "" {
		fmt.Fprintf(f, "composeFile=%s\n", cfg.ComposeFile)
	}

	if len(cfg.Apps) > 0 {
		names := make([]string, len(cfg.Apps))
		for i, a := range cfg.Apps {
			names[i] = a.Name
		}
		fmt.Fprintf(f, "apps=%s\n", strings.Join(names, ","))
		for i, a := range cfg.Apps {
			fmt.Fprintf(f, "app[%d].name=%s\n", i, a.Name)
			fmt.Fprintf(f, "app[%d].dir=%s\n", i, a.Dir)
			if a.Dockerfile != "" {
				fmt.Fprintf(f, "app[%d].dockerfile=%s\n", i, a.Dockerfile)
			}
		}
	}

	return nil
}

// LoadInitConfig reads .orkestra/bundle/.init.ork and returns the full InitConfig.
// Returns an empty InitConfig (no error) when the file does not exist.
func LoadInitConfig(dir string) (InitConfig, error) {
	path := GetInitConfigPath(dir)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return InitConfig{}, nil
		}
		return InitConfig{}, fmt.Errorf(".init.ork: %w", err)
	}

	raw := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok {
			raw[k] = v
		}
	}

	cfg := InitConfig{
		UseCompose:  raw["useCompose"] == "true",
		ComposeFile: raw["composeFile"],
	}

	if appsStr, ok := raw["apps"]; ok && appsStr != "" {
		for i, name := range strings.Split(appsStr, ",") {
			entry := AppEntry{Name: strings.TrimSpace(name)}
			if d := raw[fmt.Sprintf("app[%d].dir", i)]; d != "" {
				entry.Dir = d
			}
			if df := raw[fmt.Sprintf("app[%d].dockerfile", i)]; df != "" {
				entry.Dockerfile = df
			}
			cfg.Apps = append(cfg.Apps, entry)
		}
	}

	return cfg, nil
}

// CleanupInitConfig removes .orkestra/bundle/.init.ork after a successful deployment.
func CleanupInitConfig(dir string) error {
	path := GetInitConfigPath(dir)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf(".init.ork: cleanup failed: %w", err)
	}
	return nil
}

// Helpers

// GetInitConfigPath returns the absolute path to the .init.ork file
// located within the Orkestra bundle directory.
//
// The file is stored at: <dir>/.orkestra/bundle/.init.ork
// This directory is ignored by git (via .gitignore rule for .orkestra/bundle/),
// ensuring the init configuration remains local to the environment and is not
// accidentally committed to version control.
//
// The caller is responsible for creating the parent directories and the file itself.
// Use WriteInitConfig to write content,
// LoadInitConfig to load content
// and CleanupInitConfig to remove the file
// after a successful deployment.
func GetInitConfigPath(dir string) string {
	return filepath.Join(dir, ".orkestra", "bundle", ".init.ork")
}
