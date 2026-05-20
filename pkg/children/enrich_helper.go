package children

import orktypes "github.com/orkspace/orkestra/pkg/types"

// resolveEnrichmentTarget returns the canonical built-in name for a given
// enrichment identifier (name, plural, shorthand, alias). Returns "" if unknown.
func resolveEnrichmentTarget(s string) string {
	for canonical, list := range buildEnrichmentGroups() {
		for _, v := range list {
			if v == s {
				return canonical
			}
		}
	}
	return ""
}

// enrichmentEnabled reports whether the CRD has enabled enrichment for the
// given identifier. The identifier may be a canonical name, plural,
// shorthand, or alias; it is resolved to the canonical built‑in target
// before checking CRDEntry.ShouldEnrich.
//
// This allows enrichers to call enrichmentEnabled("cronjob") while users may
// specify any equivalent identifier in spec.enrich (e.g. "cronjobs", "cj",
// or an alias). All identifiers normalize to the same canonical target.
func enrichmentEnabled(s string, crd orktypes.CRDEntry) bool {
	resolved := resolveEnrichmentTarget(s)
	if resolved == "" || !crd.ShouldEnrich(resolved) {
		return false
	}
	return true
}
