package katalog

import (
	"reflect"
	// "sync/atomic"

	"github.com/orkspace/orkestra/pkg/konfig"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
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
	APIVersion   string                                `yaml:"apiVersion"`
	Kind         string                                `yaml:"kind"`
	Spec         orktypes.KatalogSpec                  `yaml:"spec"`
	Security     orktypes.KatalogSecurity              `yaml:"security"`
	Gateway      *orktypes.GatewayConfig               `yaml:"gateway,omitempty"`
	Notification *orktypes.KatalogNotification         `yaml:"notification,omitempty"`
	Providers    []orktypes.KatalogProviderRequirement `yaml:"providers,omitempty"`
	projectInfo  *orktypes.ProjectInfo                 `yaml:"projectInfo,omitempty"`

	KomposerMetadata orktypes.KatalogMeta `yaml:"metadata"`

	// Internal — enabledCRDs is enriched and validated; Spec.CRDs holds all (including disabled)
	metadata           orktypes.KatalogMeta         `yaml:"-" json:"-"`
	enabledCRDs        map[string]orktypes.CRDEntry `yaml:"-" json:"-"`
	katalogDir         string                       `yaml:"-" json:"-"`
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

// CRDEntry returns the enabled CRD entry for the given name.
func (k *Katalog) CRDEntry(name string) (orktypes.CRDEntry, bool) {
	entry, ok := k.enabledCRDs[name]
	return entry, ok
}

// Scheme builds and returns a runtime.Scheme with all Katalog types registered.
func (k *Katalog) Scheme() (*runtime.Scheme, error) {
	return NewSchemeRegistry(k)
}

// Empty katalog for testing
func NewEmptyKatalog() *Katalog {
	return &Katalog{}
}
