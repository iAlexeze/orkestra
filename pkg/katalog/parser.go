// pkg/katalog/parsek.go
package katalog

import (
	"fmt"

	"github.com/ialexeze/orkestra/crdkatalog"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) KomposeKatalogFromYaml(path string) ([]orktypes.CRDEntry, error) {
	data, err := utils.LoadFile(path)
	if err != nil {
		return nil, err
	}

	if err := utils.StrictUnmarshal(data, k); err != nil {
		return nil, err
	}

	// Filter enabled CRDs
	if err := k.filterEnabled(); err != nil {
		return nil, err
	}

	k.mode.Dynamic = true
	return k.enabledCRDs, nil
}

// -----------------------------------------------------------------------------
//
//	GO Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) KomposeKatalogFromGo() ([]orktypes.CRDEntry, error) {
	k.Spec.CRDs = crdkatalog.KomposeKatalogFromGo()

	// Filter
	if err := k.filterEnabled(); err != nil {
		return nil, err
	}

	k.mode.Typed = true
	return k.enabledCRDs, nil
}

// Validate Config
func (k *Katalog) validateConfig() (*Katalog, error) {
	// Validate config
	// -------------------------------------------------------------------------
	// 1. Field-level validation (required, DNS group, workers <= 5, etc.)
	// -------------------------------------------------------------------------
	if valErr := konfig.Validate().Struct(k); valErr != nil {
		k.handleValidationErrors(valErr)
		return nil, valErr
	}

	// -------------------------------------------------------------------------
	// 2. GVK uniqueness validation
	// -------------------------------------------------------------------------
	if err := k.validateGVKUniqueness(); err != nil {
		logger.Error().Err(err).Msgf("GVK uniqueness error: %v", err)
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 3. dependsOn validation (existence + cycle detection)
	// -------------------------------------------------------------------------
	if err := k.validateDependsOn(); err != nil {
		logger.Error().Err(err).Msgf("dependsOn validation error: %v", err)
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 4. Set GroupVersionKind and Defaults
	// -------------------------------------------------------------------------
	if err := k.setGroupVersionKind(); err != nil {
		logger.Error().Err(err).Msgf("Set GroupVersionKind error: %v", err)
		return nil, err
	}

	if err := k.setDefaults(); err != nil {
		logger.Error().Err(err).Msgf("Set Defaults error: %v", err)
		return nil, err
	}

	if k.mode.Dynamic {
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
			logger.Error().Err(err).Msgf("Add RuntimeObjects error: %v", err)
			return nil, err
		}

		// -------------------------------------------------------------------------
		// 6. Add Hooks	// HookRegistry → HookFactory
		// -------------------------------------------------------------------------
		if err := k.addHooks(); err != nil {
			logger.Error().Err(err).Msgf("Add Hooks error: %v", err)
			return nil, err
		}

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
