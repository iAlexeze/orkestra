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
	enabledCRDs        []orktypes.CRDEntry `yaml:"-"` // filtered
	allCRDs            []orktypes.CRDEntry `yaml:"-"` // all CRDs
	conversionRegistry *InMemoryConversionRegistry
	admissionRegistry  *InMemoryAdmissionRegistry
}

func (k *Katalog) EnabledCRDs() []orktypes.CRDEntry {
	return k.enabledCRDs
}

func (k *Katalog) AllCRDs() []orktypes.CRDEntry {
	return k.allCRDs
}

func (k *Katalog) Metadata() orktypes.KatalogMeta {
	return k.metadata
}
