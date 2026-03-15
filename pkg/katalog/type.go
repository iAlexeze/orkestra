package katalog

import (
	"reflect"

	"github.com/ialexeze/orkestra/pkg/runtime"
)

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------
const (
	GoMode   = "go"
	YamlMode = "yaml"
)

// -----------------------------------------------------------------------------
// Variables
// -----------------------------------------------------------------------------
var (
	// For updating the CRD instance - needed for lookups
	resourceTypeMap = map[reflect.Type]string{}
)

// -----------------------------------------------------------------------------
// Structs
// -----------------------------------------------------------------------------
type Katalog struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Version     string `yaml:"version"`
		Author      string `yaml:"author"`
		Website     string `yaml:"website"`
		Email       string `yaml:"email"`
	} `yaml:"metadata"`
	Spec struct {
		Finalizers []string           `yaml:"finalizers"`
		CRDs       []runtime.CRDEntry `yaml:"crds"` // raw from YAML - documentation - CLI
	} `yaml:"spec"`

	// Internal
	enabledCRDs []runtime.CRDEntry `yaml:"-"` // filtered
	mode        struct {
		Go   bool `yaml:"go"`
		Yaml bool `yaml:"yaml"`
	} `yaml:"mode"`
}
