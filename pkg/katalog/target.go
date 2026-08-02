package katalog

import (
	"sort"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// AvailableTargets returns the targets of all IDP-enabled CRDs, sorted
// alphabetically. Used in error messages ("available targets: app, database")
// and in the catalog endpoint.
func (k *Katalog) AvailableTargets() []string {
	targets := make([]string, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.HasIDPTarget() {
			targets = append(targets, crd.IDPTarget())
		}
	}
	sort.Strings(targets)
	return targets
}

// IDPCatalog returns all IDP-enabled CRD entries, sorted by target.
// Used by the catalog endpoint to list available services.
func (k *Katalog) IDPCatalog() []*orktypes.CRDEntry {
	catalog := make([]*orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.HasIDPTarget() {
			catalog = append(catalog, &crd)
		}
	}
	sort.Slice(catalog, func(i, j int) bool {
		return catalog[i].IDPTarget() < catalog[j].IDPTarget()
	})
	return catalog
}
