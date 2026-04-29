// pkg/registry/pattern.go
//
// Pattern is the atomic unit of the Orkestra Registry.
// A pattern directory contains five files:
//
//	katalog.yaml  — the operator declaration (required)
//	pattern.yaml  — registry metadata (required)
//	README.md     — human documentation
//	cr.yaml       — example CR for ork init and testing
//	crd.yaml      — CRD schema
//
// This file defines the Pattern type, the pattern.yaml schema,
// and validation logic run before push.
package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// MediaType is the OCI artifact media type for Orkestra patterns.
	// Used as the config media type in the OCI manifest.
	MediaType = "application/vnd.orkestra.pattern.v1+json"

	// FileKatalog is the required operator declaration file.
	FileKatalog = "katalog.yaml"

	// FilePattern is the required registry metadata file.
	FilePattern = "pattern.yaml"

	// FileReadme is the human documentation file.
	FileReadme = "README.md"

	// FileCR is the example CR file for ork init and e2e tests.
	FileCR = "cr.yaml"

	// FileCRD is the CRD schema file.
	FileCRD = "crd.yaml"
)

// PatternMeta is the schema for pattern.yaml.
// Read from the pattern directory and embedded as OCI manifest annotations.
type PatternMeta struct {
	Name        string           `yaml:"name"`
	Version     string           `yaml:"version"`
	Description string           `yaml:"description"`
	Author      string           `yaml:"author,omitempty"`
	License     string           `yaml:"license,omitempty"`
	Tags        []string         `yaml:"tags,omitempty"`
	Requires    PatternRequires  `yaml:"requires,omitempty"`
	Changelog   []ChangelogEntry `yaml:"changelog,omitempty"`
}

// PatternRequires declares external dependencies for a pattern.
type PatternRequires struct {
	Providers []string `yaml:"providers,omitempty"`
}

// ChangelogEntry is one version entry in the changelog.
type ChangelogEntry struct {
	Version string `yaml:"version"`
	Notes   string `yaml:"notes"`
}

// PatternIndex is the top-level index stored at registry/index:latest.
// Lists all available patterns for ork registry list.
type PatternIndex struct {
	UpdatedAt string         `json:"updatedAt"`
	Patterns  []PatternEntry `json:"patterns"`
}

// PatternEntry is one row in the pattern index.
type PatternEntry struct {
	Name          string   `json:"name"`
	LatestVersion string   `json:"latestVersion"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	Author        string   `json:"author,omitempty"`
}

// ValidateDirectory validates that dir contains a valid pattern.
// Returns a list of validated files and an error if required files are missing.
func ValidateDirectory(dir string) (*PatternMeta, []string, error) {
	// Required files
	required := []string{FileKatalog, FilePattern}
	for _, f := range required {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("required file missing: %s", f)
		}
	}

	// Parse pattern.yaml
	meta, err := loadPatternMeta(filepath.Join(dir, FilePattern))
	if err != nil {
		return nil, nil, fmt.Errorf("pattern.yaml: %w", err)
	}
	if err := meta.Validate(); err != nil {
		return nil, nil, fmt.Errorf("pattern.yaml invalid: %w", err)
	}

	// Collect present files in canonical order
	optional := []string{FileReadme, FileCR, FileCRD}
	files := append([]string{}, required...)
	for _, f := range optional {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			files = append(files, f)
		}
	}

	return meta, files, nil
}

// Validate checks that required fields are present.
func (m *PatternMeta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}

func loadPatternMeta(path string) (*PatternMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta PatternMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// LoadPatternMeta reads pattern.yaml from the given directory.
func LoadPatternMeta(dir string) (*PatternMeta, error) {
	return loadPatternMeta(filepath.Join(dir, FilePattern))
}
