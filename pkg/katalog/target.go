package katalog

import (
	"sort"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// AvailableTargets returns the targets of all serve-enabled CRDs, sorted
// alphabetically. Used in error messages ("available targets: app, database")
// and in the catalog endpoint.
// AvailableTargets returns the targets of all serve-enabled CRDs, sorted
// alphabetically. Used in error messages ("available targets: app, database")
// and in the catalog endpoint.
func (k *Katalog) AvailableTargets() []string {
	if k == nil {
		return nil
	}
	targets := make([]string, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.HasServeTarget() {
			targets = append(targets, crd.ServeTarget())
		}
	}
	sort.Strings(targets)
	return targets
}

// ServeCatalog returns all serve-enabled CRD entries, sorted by target.
// Used by the catalog endpoint to list available services.
func (k *Katalog) ServeCatalog() []*orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	catalog := make([]*orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.HasServeTarget() {
			catalog = append(catalog, &crd)
		}
	}
	sort.Slice(catalog, func(i, j int) bool {
		return catalog[i].ServeTarget() < catalog[j].ServeTarget()
	})
	return catalog
}

// ServeEnabledCRDs returns all serve-enabled CRD entries as a slice.
// Uses cached serveEnabledCRDs if available, otherwise builds it.
func (k *Katalog) ServeEnabledCRDs() []*orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	if k.serveEnabledCRDs == nil {
		return k.BuildServeEnabledCRDs()
	}
	return k.serveEnabledCRDs
}
