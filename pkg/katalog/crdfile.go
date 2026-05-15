package katalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

// populateAPITypesFromCRDFile reads the CRD YAML declared in entry.CRDFile and
// overwrites entry.APITypes with the values parsed from the file.
//
// crdFile is the source of truth — any apiTypes already declared on the entry
// are replaced. This is called in the enrichment loop in KomposeRuntimeKatalog,
// before EnrichCRDEntry, so the entry is fully specified by the time
// isFullySpecified runs.
//
// Remote URLs (http/https) are left untouched — kubectl handles those at runtime.
func populateAPITypesFromCRDFile(entry *orktypes.CRDEntry, katalogDir string) error {
	path := entry.CRDFile

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return nil
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(katalogDir, path)
	}

	apiTypes, err := readAPITypesFromCRDFile(path)
	if err != nil {
		return fmt.Errorf("reading crdFile %q: %w", entry.CRDFile, err)
	}

	// Preserve typed-mode fields (Object, List, Alias, Location) — those come
	// from the user's apiTypes declaration and are not present in the CRD YAML.
	apiTypes.Object = entry.APITypes.Object
	apiTypes.List = entry.APITypes.List
	apiTypes.Alias = entry.APITypes.Alias
	apiTypes.Location = entry.APITypes.Location
	apiTypes.APIPath = entry.APITypes.APIPath

	entry.APITypes = *apiTypes
	return nil
}

// crdFileHeader is the minimal structure needed to extract apiTypes from a CRD YAML.
type crdFileHeader struct {
	Kind string `yaml:"kind"`
	Spec struct {
		Group string `yaml:"group"`
		Names struct {
			Kind   string `yaml:"kind"`
			Plural string `yaml:"plural"`
		} `yaml:"names"`
		Versions []struct {
			Name    string `yaml:"name"`
			Served  bool   `yaml:"served"`
			Storage bool   `yaml:"storage"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

// readAPITypesFromCRDFile reads a CRD YAML file and extracts APITypes.
func readAPITypesFromCRDFile(path string) (*orktypes.APITypes, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	var doc crdFileHeader
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if doc.Kind != "CustomResourceDefinition" {
		return nil, fmt.Errorf("expected kind: CustomResourceDefinition, got %q", doc.Kind)
	}
	if doc.Spec.Group == "" {
		return nil, fmt.Errorf("spec.group is missing")
	}
	if doc.Spec.Names.Kind == "" {
		return nil, fmt.Errorf("spec.names.kind is missing")
	}
	if doc.Spec.Names.Plural == "" {
		return nil, fmt.Errorf("spec.names.plural is missing")
	}
	if len(doc.Spec.Versions) == 0 {
		return nil, fmt.Errorf("spec.versions is empty")
	}

	version := selectCRDVersion(doc.Spec.Versions)
	if version == "" {
		return nil, fmt.Errorf("no served or storage version found")
	}

	return &orktypes.APITypes{
		Group:   doc.Spec.Group,
		Version: version,
		Kind:    doc.Spec.Names.Kind,
		Plural:  doc.Spec.Names.Plural,
	}, nil
}

// selectCRDVersion picks the version name from the CRD's version list.
// Priority: storage: true → served: true → first in list.
func selectCRDVersion(versions []struct {
	Name    string `yaml:"name"`
	Served  bool   `yaml:"served"`
	Storage bool   `yaml:"storage"`
}) string {
	for _, v := range versions {
		if v.Storage {
			return v.Name
		}
	}
	for _, v := range versions {
		if v.Served {
			return v.Name
		}
	}
	if len(versions) > 0 {
		return versions[0].Name
	}
	return ""
}
