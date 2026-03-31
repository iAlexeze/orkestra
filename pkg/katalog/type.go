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
	metadata orktypes.KatalogMeta `yaml:"metadata"`
	Spec     orktypes.KatalogSpec `yaml:"spec"`
	// Internal
	enabledCRDs        []orktypes.CRDEntry `yaml:"-"` // filtered
	allCRDs            []orktypes.CRDEntry `yaml:"-"` // all CRDs
	conversionRegistry *InMemoryConversionRegistry
	admissionRegistry  *InMemoryAdmissionRegistry

	// konfig for managing katalog-related user inputs from env
	konfig *konfig.Konfig
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

// Methods to maintain the zero footprint promise of orkestra
//
// HasConversionPaths returns true only if:
//  1. Conversion is enabled in konfig, AND
//  2. At least one CRD declares conversion paths.
//
// This protects the zero‑footprint promise:
// Orkestra exposes /convert ONLY when the user explicitly declares conversion.
func (k *Katalog) HasConversionPaths() bool {
	// Katalog or konfig may be nil in edge cases (e.g., NewEmptyKatalog)
	if k == nil || k.konfig == nil {
		return false
	}

	if !k.konfig.ConversionEnabled() {
		return false
	}

	for _, crd := range k.Spec.CRDs {
		if crd.Conversion != nil && len(crd.Conversion.Paths) > 0 {
			return true
		}
	}
	return false
}

// HasValidationRules returns true only if:
//  1. Admission is enabled in konfig, AND
//  2. At least one CRD declares validation rules.
//
// This ensures /validate is created ONLY when the user declares rules.
func (k *Katalog) HasValidationRules() bool {
	if k == nil || k.konfig == nil {
		return false
	}

	if !k.konfig.AdmissionEnabled() {
		return false
	}

	for _, crd := range k.Spec.CRDs {
		if crd.Validation != nil && len(crd.Validation.Rules) > 0 {
			return true
		}
	}
	return false
}

// HasMutationRules returns true only if:
//  1. Admission is enabled in konfig, AND
//  2. At least one CRD declares mutation rules.
//
// This ensures /mutate is created ONLY when the user declares rules.
func (k *Katalog) HasMutationRules() bool {
	if k == nil || k.konfig == nil {
		return false
	}

	if !k.konfig.AdmissionEnabled() {
		return false
	}

	for _, crd := range k.Spec.CRDs {
		// Protect against nil Mutation or nil Rules
		if crd.Mutation != nil && len(crd.Mutation.Rules) > 0 {
			return true
		}
	}
	return false
}

// Empty katalog for testing
func NewEmptyKatalog() *Katalog {
	return &Katalog{}
}
