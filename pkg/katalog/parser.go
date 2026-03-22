// pkg/katalog/parsek.go
package katalog

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/merger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) KomposeKatalogFromYaml(m *merger.Merger, paths ...string) ([]orktypes.CRDEntry, error) {
	k.Spec = m.ToSpec()
	k.enabledCRDs = m.Enabled() // Enabled CRDs for all operations
	k.allCRDs = m.All()         // All CRDs for documentation
	k.metadata = m.ToMeta()     // Metadata for CLI and health endpoints

	// Enrich enabled CRDs
	for i := range k.enabledCRDs {
		entry := &k.enabledCRDs[i]

		outcome, err := EnrichCRDEntry(entry)
		if err != nil {
			return nil, err
		}
		entry.EnrichmentOutcome = outcome
	}

	return k.enabledCRDs, nil
}

// Validate Config
func (k *Katalog) ValidateConfig() (*Katalog, error) {
	// Validate config
	// -------------------------------------------------------------------------
	// 1. Field-level validation (required, DNS group, workers <= 5, etc.)
	// -------------------------------------------------------------------------
	if valErr := konfig.Validate().Struct(k); valErr != nil {
		k.handleValidationErrors(valErr)
		return nil, valErr
	}

	// -------------------------------------------------------------------------
	// 2. Uniqueness validation
	// -------------------------------------------------------------------------
	if err := k.validateUniqueness(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 3. dependsOn validation (existence + cycle detection)
	// -------------------------------------------------------------------------
	if err := k.validateDependsOn(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 4. Set GroupVersionKind and Defaults
	// -------------------------------------------------------------------------
	if err := k.setGroupVersionKind(); err != nil {
		return nil, err
	}

	if err := k.setDefaults(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 5. Add Reconcilers		// ReconcilerRegistry → Constructor
	// -------------------------------------------------------------------------
	if err := k.addReconcilers(); err != nil {
		logger.Error().Err(err).Msgf("Add Reconcilers error: %v", err)
		return nil, err
	}
	// -------------------------------------------------------------------------
	// 6. Add RuntimeObjects	// ObjectRegistry + ListRegistry
	// -------------------------------------------------------------------------
	if err := k.addRuntimeObjects(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 6. Add Hooks	// HookRegistry → HookFactory
	// -------------------------------------------------------------------------
	if err := k.addHooks(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 7. Validate Reconciler modes
	// -------------------------------------------------------------------------
	if err := k.validateReconcilerMode(); err != nil {
		return nil, err
	}
	return k, nil
}

// Helpers
func (k *Katalog) empty() bool {
	return len(k.Spec.CRDs) == 0
}

func (k *Katalog) enabledEmpty() bool {
	return len(k.enabledCRDs) == 0
}

// Filter enabled CRDs
func (k *Katalog) filterEnabled() error {
	if k.empty() {
		return fmt.Errorf("Katalog is empty")
	}

	// Filter enabled CRDs
	for _, crd := range k.Spec.CRDs {
		if crd.Enabled {
			k.enabledCRDs = append(k.enabledCRDs, crd)
		} else {
			logger.Warn().Msgf("%s disabled. skipping...", crd.Name)
		}
	}

	if k.enabledEmpty() {
		return fmt.Errorf("no enabled CRDs found")
	}

	return nil
}
