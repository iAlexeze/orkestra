// pkg/katalog/parsek.go
package katalog

import (
	"fmt"

	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
	"gopkg.in/yaml.v3"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) buildKatalogFromYaml(path string) ([]initialize.CRDEntry, error) {
	data, err := utils.LoadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, k); err != nil {
		return nil, err
	}

	// Filter enabled CRDs
	if err := k.filterEnabled(); err != nil {
		return nil, err
	}

	k.mode.Yaml = true
	return k.enabledCRDs, nil
}

// -----------------------------------------------------------------------------
//
//	GO Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) buildKatalogFromGo() ([]initialize.CRDEntry, error) {
	k.CRDs = initialize.BuildKatalogFromGo()

	// Filter
	if err := k.filterEnabled(); err != nil {
		return nil, err
	}

	k.mode.Go = true
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
	if err := k.SetGroupVersionKind(); err != nil {
		logger.Error().Err(err).Msgf("Set GroupVersionKind error: %v", err)
		return nil, err
	}

	if err := k.SetDefaults(); err != nil {
		logger.Error().Err(err).Msgf("Set Defaults error: %v", err)
		return nil, err
	}

	if k.mode.Yaml {
		// -------------------------------------------------------------------------
		// 5. Add Reconcilers
		// -------------------------------------------------------------------------
		if err := k.addReconcilers(); err != nil {
			logger.Error().Err(err).Msgf("Add Reconcilers error: %v", err)
			return nil, err
		}

		// -------------------------------------------------------------------------
		// 6. Add RuntimeObjects
		// -------------------------------------------------------------------------
		if err := k.addRuntimeObjects(); err != nil {
			logger.Error().Err(err).Msgf("Add RuntimeObjects error: %v", err)
			return nil, err
		}
	}

	return k, nil
}

// Helpers
func (k *Katalog) empty() bool {
	return len(k.CRDs) == 0
}

func (k *Katalog) enabledEmpty() bool {
	return len(k.enabledCRDs) == 0
}

func (k *Katalog) List() []initialize.CRDEntry {
	return k.CRDs
}

func (k *Katalog) Enabled() []initialize.CRDEntry {
	return k.enabledCRDs
}

// Get tries to get an enabled crd
func (k *Katalog) Get(name string) (*initialize.CRDEntry, error) {
	for _, crd := range k.enabledCRDs {
		if crd.Name == name {
			return &crd, nil
		}
	}
	return nil, fmt.Errorf("crd not found in katalog")
}

// Filter enabled CRDs
func (k *Katalog) filterEnabled() error {
	if k.empty() {
		return fmt.Errorf("Katalog is empty")
	}

	// Filter enabled CRDs
	for _, crd := range k.CRDs {
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
