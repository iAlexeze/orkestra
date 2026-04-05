package katalog

import (
	"reflect"

	"github.com/ialexeze/orkestra/pkg/konfig"
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
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	metadata   orktypes.KatalogMeta `yaml:"-"`

	KomposerMetadata orktypes.KatalogMeta `yaml:"metadata"`

	Spec orktypes.KatalogSpec `yaml:"spec"`
	// Internal — enabledCRDs is enriched and validated; Spec.CRDs holds all (including disabled)
	enabledCRDs        map[string]orktypes.CRDEntry `yaml:"-"`
	conversionRegistry *InMemoryConversionRegistry
	admissionRegistry  *InMemoryAdmissionRegistry

	// konfig for managing katalog-related user inputs from env
	konfig *konfig.Konfig
}

func (k *Katalog) EnabledCRDs() map[string]orktypes.CRDEntry {
	return k.enabledCRDs
}

// AllCRDs returns all CRDs including disabled ones (from Spec).
func (k *Katalog) AllCRDs() map[string]orktypes.CRDEntry {
	return k.Spec.CRDs
}

func (k *Katalog) Metadata() orktypes.KatalogMeta {
	return k.metadata
}

// Empty katalog for testing
func NewEmptyKatalog() *Katalog {
	return &Katalog{}
}
