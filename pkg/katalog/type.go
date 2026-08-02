package katalog

import (
	"reflect"

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
	Notes        orktypes.NoteRegistry                 `yaml:"notes,omitempty"`
	Profiles     orktypes.ProfileRegistry              `yaml:"profiles,omitempty"`
	Gateway      *orktypes.GatewayConfig               `yaml:"gateway,omitempty"`
	Notification *orktypes.KatalogNotification         `yaml:"notification,omitempty"`
	Providers    []orktypes.KatalogProviderRequirement `yaml:"providers,omitempty"`
	projectInfo  interface{}                           `yaml:"projectInfo,omitempty"`

	KomposerMetadata orktypes.KatalogMeta `yaml:"metadata"`

	// Internal — enabledCRDs is enriched and validated; Spec.CRDs holds all (including disabled)
	metadata           orktypes.KatalogMeta         `yaml:"-" json:"-"`
	enabledCRDs        map[string]orktypes.CRDEntry `yaml:"-" json:"-"`
	withCRDFiles       []string                     `yaml:"-" json:"-"` // CRD names that declared crdFile, captured before the field is cleared
	katalogDir         string                       `yaml:"-" json:"-"`
	conversionRegistry *InMemoryConversionRegistry
	admissionRegistry  *InMemoryAdmissionRegistry

	// konfig for managing katalog-related user inputs from env
	konfig *konfig.Konfig

	// Warnings collects non‑fatal validation messages for this CRD.
	Warnings orktypes.Warnings `json:"-"` // not serialized

}

// EnabledCRDs returns a map of enabled CRDs.
func (k *Katalog) EnabledCRDs() map[string]orktypes.CRDEntry {
	return k.enabledCRDs
}

// AllCRDs returns all CRDs including disabled ones (from Spec).
func (k *Katalog) AllCRDs() map[string]orktypes.CRDEntry {
	return k.Spec.CRDs
}

// WithCRDFiles returns the names of CRDs that declared a crdFile.
// Populated during KomposeRuntimeKatalog before the field is cleared,
// so callers can inspect which CRDs used the local-file shortcut even
// after apiTypes have been resolved and CRDFile wiped.
func (k *Katalog) WithCRDFiles() []string {
	return k.withCRDFiles
}

// Metadata returns the Katalog metadata.
func (k *Katalog) Metadata() orktypes.KatalogMeta {
	return k.metadata
}

// IsDeprecated returns true if the Katalog is deprecated.
func (k *Katalog) IsDeprecated() bool {
	if k.metadata.Deprecation == nil {
		return false
	}
	return k.metadata.Deprecation.IsDeprecated()
}

// IsMigrated returns true if the Katalog is deprecated and has a migration target.
func (k *Katalog) IsMigrated() bool {
	if k.metadata.Deprecation == nil {
		return false
	}
	return k.metadata.Deprecation.MigrationTarget() != ""
}

// MigrationTarget returns the value of the MigratedTo field.
// If the deprecation block is nil or empty, it returns an empty string.
func (k *Katalog) MigrationTarget() string {
	if k.metadata.Deprecation == nil {
		return ""
	}
	return k.metadata.Deprecation.MigrationTarget()
}

// MigrationMessage returns the deprecation message.
func (k *Katalog) MigrationMessage() string {
	if k.metadata.Deprecation == nil {
		return ""
	}
	return k.metadata.Deprecation.MigrationMessage()
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
