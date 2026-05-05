// pkg/katalog/motif_imports.go
//
// Expands operatorBox.imports: blocks at Katalog load time.
// Each import loads the referenced Motif, binds its inputs from with:,
// and merges the expanded resources into OnReconcile.
package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/motif"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// expandMotifImports resolves all operatorBox.imports entries across enabled
// CRDs, loads each Motif, expands it with the declared bindings, and merges
// the resulting resources into the CRD's OnReconcile block.
//
// Called once during KomposeRuntimeKatalog, after the merger produces the
// enabled CRD map and before validation. Static (non-template) with: values
// are expanded here; dynamic with: values (template expressions referencing
// .spec.*) are carried through as-is and resolved per reconcile.
func (k *Katalog) expandMotifImports() error {
	for name, entry := range k.enabledCRDs {
		if len(entry.OperatorBox.Imports) == 0 {
			continue
		}

		for i, imp := range entry.OperatorBox.Imports {
			m, err := motif.LoadImport(&imp)
			if err != nil {
				return fmt.Errorf(
					"CRD %q: operatorBox.imports[%d]: loading motif %q: %w",
					name, i, imp.Motif, err,
				)
			}

			expanded, err := motif.Expand(m, imp.With)
			if err != nil {
				return fmt.Errorf(
					"CRD %q: operatorBox.imports[%d] (motif %q): %w",
					name, i, imp.Motif, err,
				)
			}

			// Ensure OnReconcile exists to merge into
			if entry.OperatorBox.OnReconcile == nil {
				entry.OperatorBox.OnReconcile = &orktypes.HookTemplates{}
			}

			motif.MergeHookTemplates(entry.OperatorBox.OnReconcile, expanded)
		}

		// Clear imports after expansion — they have been inlined
		entry.OperatorBox.Imports = nil
		k.enabledCRDs[name] = entry
	}
	return nil
}
