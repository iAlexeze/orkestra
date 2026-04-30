// pkg/registry/pattern.go
//
// Pattern is the atomic unit of the Orkestra Registry.
// A pattern directory contains:
//
//	katalog.yaml  — operator declaration (required)
//	crd.yaml      — CRD schema (required)
//	README.md     — human documentation (optional)
//	cr.yaml       — example CR (optional)
//
// Pattern metadata is derived entirely from katalog.yaml:
//   - name, version, description, author, tags from metadata
//   - required providers from spec.providers
//
// No separate pattern.yaml is required or allowed.
package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// MediaType is the OCI artifact media type for Orkestra patterns.
	MediaType = "application/vnd.orkestra.pattern.v1+tar+gzip"

	// FileKatalog is the required operator declaration file.
	FileKatalog = "katalog.yaml"

	// FileCRD is the required CRD schema file.
	FileCRD = "crd.yaml"

	// FileReadme is the human documentation file.
	FileReadme = "README.md"

	// FileCR is the example CR file.
	FileCR = "cr.yaml"
)

// PatternMeta holds metadata derived from the Katalog.
type PatternMeta struct {
	Name        string           `yaml:"name"`
	Version     string           `yaml:"version"`
	Description string           `yaml:"description"`
	Author      string           `yaml:"author,omitempty"`
	License     string           `yaml:"license,omitempty"`
	Tags        []string         `yaml:"tags,omitempty"`
	Requires    PatternRequires  `yaml:"requires,omitempty"`
	Changelog   []ChangelogEntry `yaml:"changelog,omitempty"` // reserved for future use
}

// PatternRequires declares external dependencies.
type PatternRequires struct {
	Providers []string `yaml:"providers,omitempty"`
}

// ChangelogEntry represents one version entry in the changelog.
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

// ValidateDirectory checks that dir contains a valid pattern.
// Returns metadata (from katalog.yaml) and the list of files to include.
func ValidateDirectory(dir string) (*PatternMeta, []string, error) {
	required := []string{FileKatalog, FileCRD}
	for _, f := range required {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return nil, nil, fmt.Errorf("required file missing: %s", f)
		}
	}

	meta, err := deriveMetadataFromKatalog(filepath.Join(dir, FileKatalog))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing katalog.yaml: %w", err)
	}

	files := []string{FileKatalog, FileCRD}
	optional := []string{FileReadme, FileCR}
	for _, f := range optional {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			files = append(files, f)
		}
	}
	return meta, files, nil
}

// deriveMetadataFromKatalog reads katalog.yaml and builds PatternMeta.
func deriveMetadataFromKatalog(path string) (*PatternMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var katalog struct {
		Metadata struct {
			Name        string   `yaml:"name"`
			Version     string   `yaml:"version"`
			Author      string   `yaml:"author"`
			Description string   `yaml:"description"`
			Tags        []string `yaml:"tags"`
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
	if katalog.Metadata.Name == "" {
		return nil, fmt.Errorf("metadata.name is required")
	}
	meta := &PatternMeta{
		Name:        katalog.Metadata.Name,
		Version:     katalog.Metadata.Version,
		Description: katalog.Metadata.Description,
		Author:      katalog.Metadata.Author,
		Tags:        katalog.Metadata.Tags,
	}
	// Collect required providers
	var providers []string
	for _, p := range katalog.Spec.Providers {
		if p.Required {
			providers = append(providers, p.Name)
		}
	}
	if len(providers) > 0 {
		meta.Requires.Providers = providers
	}
	// Apply defaults
	if meta.Version == "" {
		meta.Version = "latest"
	}
	if meta.Description == "" {
		meta.Description = fmt.Sprintf("Pattern for %s", meta.Name)
	}
	return meta, nil
}

// LoadPatternMeta returns pattern metadata for a directory (convenience wrapper).
func LoadPatternMeta(dir string) (*PatternMeta, error) {
	meta, _, err := ValidateDirectory(dir)
	return meta, err
}
