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
	APIVersion       string                   `yaml:"apiVersion"`
	Kind             string                   `yaml:"kind"`
	Spec             orktypes.KatalogSpec     `yaml:"spec"`
	Security         orktypes.KatalogSecurity `yaml:"security"`
	KomposerMetadata orktypes.KatalogMeta     `yaml:"metadata"`

	// Internal — enabledCRDs is enriched and validated; Spec.CRDs holds all (including disabled)
	metadata           orktypes.KatalogMeta         `yaml:"-" json:"-"`
	enabledCRDs        map[string]orktypes.CRDEntry `yaml:"-" json:"-"`
	conversionRegistry *InMemoryConversionRegistry
	admissionRegistry  *InMemoryAdmissionRegistry

	// konfig for managing katalog-related user inputs from env
	konfig *konfig.Konfig
}

// EnabledCRDs returns a map of enabled CRDs.
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
