package registry

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
type CRDRegistry struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`

	CRDs        []initialize.CRDEntry `yaml:"crds"` // raw from YAML
	EnabledCRDs []initialize.CRDEntry `yaml:"-"`    // filtered
	AllCRDs     []initialize.CRDEntry `yaml:"-"`    // documentation - CLI

	Mode struct {
		Go   bool `yaml:"go"`
		Yaml bool `yaml:"yaml"`
	} `yaml:"mode"`
}
