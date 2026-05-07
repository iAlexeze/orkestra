package doctor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/konfig"
	"gopkg.in/yaml.v3"
)

const (
	RuntimeKatalogPath = "__runtime_katalog_do_not_edit.yml"
)

// GlobalKomposer is the structure of ~/.orkestra/deploy/komposer.yaml.
// It aggregates Katalog paths from all deployed projects on this machine.
type GlobalKomposer struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   KomposerMeta    `yaml:"metadata"`
	Imports    KomposerSources `yaml:"imports"`
}

type KomposerMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version,omitempty"`
	Author      string `yaml:"author,omitempty"`
	License     string `yaml:"license,omitempty"`
}

type KomposerSources struct {
	Files []string `yaml:"files,omitempty"`
}

// GlobalKomposerPath returns ~/.orkestra/deploy/komposer.yaml.
func GlobalKomposerPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "komposer.yaml"), nil
}

// LoadGlobalKomposer reads the global Komposer, returning a fresh one if absent.
func LoadGlobalKomposer() (*GlobalKomposer, error) {
	clusterName := CurrentContext()
	if clusterName == "unknown" {
		clusterName = "orkestra-managed-cluster"
	}

	author, err := LastCommitAuthor()
	if err != nil {
		author = &GitAuthor{}
	}

	path, err := GlobalKomposerPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &GlobalKomposer{
			APIVersion: "orkestra.orkspace.io/v1",
			Kind:       konfig.KomposerKind(),
			Metadata: KomposerMeta{
				Name:   clusterName,
				Author: author.Raw,
			},
			Imports: KomposerSources{},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var k GlobalKomposer
	return &k, yaml.Unmarshal(data, &k)
}

// RegisterKatalog adds katalogPath to the Komposer sources if not already present.
// Returns true when the path was newly added.
func (k *GlobalKomposer) RegisterKatalog(katalogPath string) bool {
	for _, f := range k.Imports.Files {
		if f == katalogPath {
			return false
		}
	}
	k.Imports.Files = append(k.Imports.Files, katalogPath)
	return true
}

// Save writes the global Komposer file.
func (k *GlobalKomposer) Save() error {
	path, err := GlobalKomposerPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// DeployedProjects returns app names inferred from the Komposer source paths.
// Paths are expected to contain a .orkestra/ directory segment.
func (k *GlobalKomposer) DeployedProjects() []string {
	var names []string
	for _, f := range k.Imports.Files {
		parts := strings.Split(filepath.ToSlash(f), "/")
		for i, p := range parts {
			if p == ".orkestra" && i > 0 {
				names = append(names, parts[i-1])
				break
			}
		}
	}
	return names
}

type GitAuthor struct {
	Name     string
	Email    string
	Raw      string // "Name <email>"
	Notfound bool
}

// LastCommitAuthor returns "Name <email>" from the most recent Git commit.
func LastCommitAuthor() (*GitAuthor, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%an|%ae")

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(out.String()), "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected git output: %q", out.String())
	}

	name := strings.TrimSpace(parts[0])
	email := strings.TrimSpace(parts[1])

	author := GitAuthor{}
	if name == "" && email == "" {
		author.Notfound = true
	}

	author.Email = email
	author.Name = name
	author.Raw = fmt.Sprintf("%s <%s>", name, email)

	return &author, nil
}
