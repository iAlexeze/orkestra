package katalog

import orktypes "github.com/orkspace/orkestra/pkg/types"

// NewKatalogForTest creates a Katalog with pre-set enabledCRDs for testing.
// Bypasses YAML parsing and ValidateConfig so tests can construct controlled graphs.
func NewKatalogForTest(crds map[string]orktypes.CRDEntry) *Katalog {
	return &Katalog{enabledCRDs: crds}
}

// DetectCyclesForTest exposes detectDependencyCycles for integration tests.
func DetectCyclesForTest(k *Katalog) error {
	return k.detectDependencyCycles()
}
