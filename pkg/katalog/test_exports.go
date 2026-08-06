package katalog

import (
	"github.com/orkspace/orkestra/pkg/konfig"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// wireForTest runs the same setGroupVersionKind → setDefaults sequence
// ValidateConfig runs in production (see validate.go), so a Katalog built by
// NewKatalogForTest/NewFromEntryPointers is immediately usable by
// LookupByKind/LookupByName/GVR/etc. without every test remembering to call
// BuildLookupIndexes (or hitting Kind()/GVR() silently returning zero values
// because GroupVersionKind/GroupVersionResource were never computed).
// Panics on error — these are hand-authored test fixtures, not untrusted
// input, so a malformed one should fail loudly and immediately.
func (k *Katalog) wireForTest() *Katalog {
	// setGroupVersionKind is best-effort: graph-only fixtures (e.g. cycle
	// detection tests) legitimately omit apiTypes, so we skip rather than panic.
	_ = k.setGroupVersionKind()
	if err := k.setDefaults(konfig.NewDefaultKonfig()); err != nil {
		panic("katalog test fixture: " + err.Error())
	}
	return k
}

// NewKatalogForTest creates a Katalog with pre-set enabledCRDs for testing.
// Bypasses YAML parsing and ValidateConfig's other steps (uniqueness,
// dependsOn, reconciler mode, …) but still wires GVK/GVR/lookup indexes and
// defaults exactly as ValidateConfig would, so lookups and field defaults
// behave the same as a fully loaded Katalog.
func NewKatalogForTest(crds map[string]orktypes.CRDEntry) *Katalog {
	if crds == nil {
		crds = map[string]orktypes.CRDEntry{}
	}
	for key, entry := range crds {
		if entry.Name == "" {
			entry.Name = key
			crds[key] = entry
		}
	}
	return (&Katalog{enabledCRDs: crds}).wireForTest()
}

// NewFromEntryPointers creates a Katalog from a map of CRD entry pointers.
// Useful when you have pointers and want to avoid copying. See
// NewKatalogForTest for what gets wired.
func NewFromEntryPointers(entries map[string]*orktypes.CRDEntry) *Katalog {
	if entries == nil {
		entries = map[string]*orktypes.CRDEntry{}
	}
	kat := &Katalog{
		enabledCRDs: make(map[string]orktypes.CRDEntry, len(entries)),
	}
	for k, v := range entries {
		if v != nil {
			entry := *v
			if entry.Name == "" {
				entry.Name = k
			}
			kat.enabledCRDs[k] = entry
		}
	}
	return kat.wireForTest()
}

// DetectCyclesForTest exposes detectDependencyCycles for integration tests.
func DetectCyclesForTest(k *Katalog) error {
	return k.detectDependencyCycles()
}
