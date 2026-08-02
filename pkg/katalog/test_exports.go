package katalog

import orktypes "github.com/orkspace/orkestra/pkg/types"

// NewKatalogForTest creates a Katalog with pre-set enabledCRDs for testing.
// Bypasses YAML parsing and ValidateConfig so tests can construct controlled graphs.
func NewKatalogForTest(crds map[string]orktypes.CRDEntry) *Katalog {
	if crds == nil {
		crds = map[string]orktypes.CRDEntry{}
	}
	return &Katalog{enabledCRDs: crds}
}

// NewFromEntryPointers creates a Katalog from a map of CRD entry pointers.
// Useful when you have pointers and want to avoid copying.
func NewFromEntryPointers(entries map[string]*orktypes.CRDEntry) *Katalog {
	if entries == nil {
		entries = map[string]*orktypes.CRDEntry{}
	}
	kat := &Katalog{
		enabledCRDs: make(map[string]orktypes.CRDEntry, len(entries)),
	}
	for k, v := range entries {
		if v != nil {
			kat.enabledCRDs[k] = *v
		}
	}
	return kat
}

// DetectCyclesForTest exposes detectDependencyCycles for integration tests.
func DetectCyclesForTest(k *Katalog) error {
	return k.detectDependencyCycles()
}
