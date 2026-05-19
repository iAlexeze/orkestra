package katalog

import "fmt"

// validEnrichTargets is the set of currently supported enrichment targets.
var validEnrichTargets = map[string]bool{
	"pods":      true,
	"endpoints": true,
	"events":    true,
	"pvc":       true,
}

// validateEnrich fails fast when:
//   - both enrichAll and enrich are set on the same CRD (mutually exclusive)
//   - an unrecognised target appears in enrich
func (k *Katalog) validateEnrich() error {
	for name, crd := range k.Enabled() {
		if crd.EnrichAll && len(crd.Enrich) > 0 {
			return fmt.Errorf(
				"crd %q: enrichAll and enrich are mutually exclusive — use enrichAll: true OR enrich: [...]",
				name,
			)
		}
		for _, target := range crd.Enrich {
			if !validEnrichTargets[target] {
				return fmt.Errorf(
					"crd %q: unknown enrich target %q — supported targets: pods, endpoints, events, pvc",
					name, target,
				)
			}
		}
	}
	return nil
}
