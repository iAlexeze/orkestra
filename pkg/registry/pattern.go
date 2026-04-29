// pkg/registry/pattern.go
//
// Pattern is the atomic unit of the Orkestra Registry.
// A pattern directory contains:
//
//	katalog.yaml  — operator declaration (required)
//	crd.yaml      — CRD schema (required)
//	README.md     — human documentation (optional)
//	cr.yaml       — example CR (optional)
//	pattern.yaml  — registry metadata (optional, overrides derived fields)
//
// The pattern metadata is primarily derived from katalog.yaml (metadata.name,
// metadata.version, metadata.author, metadata.description, spec.providers).
// If pattern.yaml is present, its fields override the derived values.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// MediaType is the OCI artifact media type for Orkestra patterns.
	MediaType = "application/vnd.orkestra.pattern.v1+tar+gzip"

	// FileKatalog is the required operator declaration file.
	FileKatalog = "katalog.yaml"

	// FileCRD is the required CRD schema file.
	FileCRD = "crd.yaml"

	// FilePattern is the optional registry metadata file.
	FilePattern = "pattern.yaml"

	// FileReadme is the human documentation file.
	FileReadme = "README.md"

	// FileCR is the example CR file.
	FileCR = "cr.yaml"
)

// PatternMeta is the schema for pattern metadata.
// Derived from katalog.yaml (and optionally pattern.yaml/CRD).
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

// PatternRequires declares external dependencies.
type PatternRequires struct {
	Providers []string `yaml:"providers,omitempty"`
}

// ChangelogEntry is one version entry in the changelog.
type ChangelogEntry struct {
	Version string `yaml:"version"`
	Notes   string `yaml:"notes"`
}

// PatternIndex is the top-level index stored at registry/index:latest.
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
// Returns the pattern metadata (derived from katalog.yaml, overridden by pattern.yaml if present),
// and the list of files to include in the OCI artifact.
func ValidateDirectory(dir string) (*PatternMeta, []string, error) {
	// Required files
	required := []string{FileKatalog, FileCRD}
	for _, f := range required {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("required file missing: %s", f)
		}
	}

	// Derive metadata from katalog.yaml
	meta, err := deriveMetadataFromKatalog(filepath.Join(dir, FileKatalog))
	if err != nil {
		return nil, nil, fmt.Errorf("reading katalog.yaml: %w", err)
	}

	// Optionally override with pattern.yaml
	patternPath := filepath.Join(dir, FilePattern)
	if _, err := os.Stat(patternPath); err == nil {
		override, err := loadPatternMeta(patternPath)
		if err != nil {
			return nil, nil, fmt.Errorf("pattern.yaml: %w", err)
		}
		mergeMeta(meta, override)
	}

	// Validate derived metadata
	if err := meta.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid pattern metadata: %w", err)
	}

	// Collect present files in canonical order (katalog, crd, then optional)
	files := []string{FileKatalog, FileCRD}
	optional := []string{FileReadme, FileCR, FilePattern}
	for _, f := range optional {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			files = append(files, f)
		}
	}

	return meta, files, nil
}

// deriveMetadataFromKatalog reads a katalog.yaml file and extracts pattern metadata.
// It also reads the CRD to infer the pattern name if metadata.name is missing.
func deriveMetadataFromKatalog(path string) (*PatternMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var katalog struct {
		Metadata struct {
			Name        string `yaml:"name"`
			Version     string `yaml:"version"`
			Author      string `yaml:"author"`
			Description string `yaml:"description"`
		} `yaml:"metadata"`
		Spec struct {
			Providers []struct {
				Name     string `yaml:"name"`
				Required bool   `yaml:"required"`
			} `yaml:"providers"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &katalog); err != nil {
		return nil, err
	}

	meta := &PatternMeta{
		Name:        katalog.Metadata.Name,
		Version:     katalog.Metadata.Version,
		Description: katalog.Metadata.Description,
		Author:      katalog.Metadata.Author,
		Tags:        []string{},
	}

	// Extract providers from spec.providers
	var providers []string
	for _, p := range katalog.Spec.Providers {
		if p.Required {
			providers = append(providers, p.Name)
		}
	}
	if len(providers) > 0 {
		meta.Requires.Providers = providers
	}

	// If name is missing, try to infer from CRD (optional)
	if meta.Name == "" {
		if crdPath := filepath.Join(filepath.Dir(path), FileCRD); pathExists(crdPath) {
			if crdName, err := inferNameFromCRD(crdPath); err == nil && crdName != "" {
				meta.Name = crdName
			}
		}
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("failed to determine pattern name (set katalog.metadata.name or ensure crd.yaml exists)")
	}
	if meta.Version == "" {
		meta.Version = "0.1.0"
	}
	if meta.Description == "" {
		meta.Description = fmt.Sprintf("Pattern for %s", meta.Name)
	}
	return meta, nil
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func inferNameFromCRD(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var crd struct {
		Spec struct {
			Names struct {
				Kind string `yaml:"kind"`
			} `yaml:"names"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return "", err
	}
	return strings.ToLower(crd.Spec.Names.Kind), nil
}

// loadPatternMeta reads pattern.yaml.
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

// mergeMeta overlays override fields onto base (non‑empty values from override win).
func mergeMeta(base, override *PatternMeta) {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.Version != "" {
		base.Version = override.Version
	}
	if override.Description != "" {
		base.Description = override.Description
	}
	if override.Author != "" {
		base.Author = override.Author
	}
	if override.License != "" {
		base.License = override.License
	}
	if len(override.Tags) > 0 {
		base.Tags = override.Tags
	}
	if len(override.Requires.Providers) > 0 {
		base.Requires.Providers = override.Requires.Providers
	}
	if len(override.Changelog) > 0 {
		base.Changelog = override.Changelog
	}
}

// Validate checks required fields.
func (m *PatternMeta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("pattern name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("pattern version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("pattern description is required")
	}
	return nil
}

// LoadPatternMeta reads pattern metadata from the given directory,
// following the same derivation rules as ValidateDirectory.
func LoadPatternMeta(dir string) (*PatternMeta, error) {
	meta, _, err := ValidateDirectory(dir)
	return meta, err
}
