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
	serveEnabledCRDs   []*orktypes.CRDEntry         `yaml:"-" json:"-"`
	withCRDFiles       []string                     `yaml:"-" json:"-"` // CRD names that declared crdFile, captured before the field is cleared
	katalogDir         string                       `yaml:"-" json:"-"`
	conversionRegistry *InMemoryConversionRegistry
	admissionRegistry  *InMemoryAdmissionRegistry

	// konfig for managing katalog-related user inputs from env
	konfig *konfig.Konfig

	// Warnings collects non‑fatal validation messages for this CRD.
	Warnings orktypes.Warnings `json:"-"` // not serialized

	// Indexes for O(1) lookups
	kindIndex       map[string]string `yaml:"-" json:"-"` // kind -> crd name
	nameIndex       map[string]string `yaml:"-" json:"-"` // lowercase(map key) -> crd name
	pluralIndex     map[string]string `yaml:"-" json:"-"` // plural resource name -> crd name
	apiVersionIndex map[string]string `yaml:"-" json:"-"` // apiVersion -> crd name
	gvkIndex        map[string]string `yaml:"-" json:"-"` // gvk.String() -> crd name
	gvrIndex        map[string]string `yaml:"-" json:"-"` // gvr.String() -> crd name
	targetIndex     map[string]string `yaml:"-" json:"-"` // target -> crd name

	webhookNameIndex map[string]string `yaml:"-" json:"-"` // lowercase(webhook entry name) -> source ("github"/"gitlab"/"slack"/"generic")
}

// EnabledCRDs returns a map of enabled CRDs keyed by their name.
func (k *Katalog) EnabledCRDs() map[string]orktypes.CRDEntry {
	return k.enabledCRDs
}

// ListEnabledCRDs returns a slice of all enabled CRD entries.
// Use this for iteration when the map key is not needed.
func (k *Katalog) ListEnabledCRDs() []orktypes.CRDEntry {
	entries := make([]orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		entries = append(entries, crd)
	}
	return entries
}

// ListEnabledCRDPointers returns a slice of pointers to all enabled CRD entries.
// Use this when you need to modify CRD entries or avoid copying.
func (k *Katalog) ListEnabledCRDPointers() []*orktypes.CRDEntry {
	entries := make([]*orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		crdCopy := crd
		entries = append(entries, &crdCopy)
	}
	return entries
}

// EnabledCRDsList returns a slice of all enabled CRD entries.
func (k *Katalog) EnabledCRDsList() []orktypes.CRDEntry {
	entries := make([]orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		entries = append(entries, crd)
	}
	return entries
}

// EnabledCRDMap returns the raw map of enabled CRDs.
func (k *Katalog) EnabledCRDMap() map[string]orktypes.CRDEntry {
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

// Deprecation returns the raw deprecation block, or nil if absent.
func (k *Katalog) Deprecation() *orktypes.KatalogDeprecation {
	return k.metadata.Deprecation
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
