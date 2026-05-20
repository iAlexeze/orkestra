package katalog

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/children"
)

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

		supportedGroups := children.SupportedEnrichmentGroups()

		for _, target := range crd.Enrich {
			if !children.IsValidEnrichmentTarget(target) {
				return fmt.Errorf(
					"crd %q: unknown enrich target %q — supported targets:%s",
					name,
					target,
					formatEnrichmentGroups(supportedGroups),
				)
			}
		}

	}
	return nil
}

// helper
func formatEnrichmentGroups(groups map[string][]string) string {
	var b strings.Builder

	for kind, list := range groups {
		b.WriteString("\n  ")
		b.WriteString(kind)
		b.WriteString(":\n    ")
		b.WriteString(strings.Join(list, ", "))
	}

	return b.String()
}
