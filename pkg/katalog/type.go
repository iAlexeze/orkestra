package katalog

import (
	"reflect"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
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
	metadata orktypes.KatalogMeta `yaml:"metadata"`
	Spec     orktypes.KatalogSpec `yaml:"spec"`
	// Internal
	enabledCRDs []orktypes.CRDEntry `yaml:"-"` // filtered
}
