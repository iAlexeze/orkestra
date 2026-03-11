package katalog

import (
	"reflect"

	"github.com/ialexeze/orkestra/initialize"
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
	aPIVersion string `yaml:"apiVersion"`
	kind       string `yaml:"kind"`
	metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`

	crds        []initialize.CRDEntry `yaml:"crds"` // raw from YAML - documentation - CLI
	enabledCRDs []initialize.CRDEntry `yaml:"-"`    // filtered
	mode        struct {
		Go   bool `yaml:"go"`
		Yaml bool `yaml:"yaml"`
	} `yaml:"mode"`
}
