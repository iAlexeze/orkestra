// pkg/katalog/parser.go
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
func (r *Katalog) buildKatalogFromYaml(path string) ([]initialize.CRDEntry, error) {
	data, err := utils.LoadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, r); err != nil {
		return nil, err
	}

	// Filter enabled CRDs
	if err := r.filterEnabled(); err != nil {
		return nil, err
	}

	r.mode.Yaml = true
	return r.enabledCRDs, nil
}

// -----------------------------------------------------------------------------
//
//	GO Builder
//
// -----------------------------------------------------------------------------
func (r *Katalog) buildKatalogFromGo() ([]initialize.CRDEntry, error) {
	r.crds = initialize.BuildKatalogFromGo()

	// Filter
	if err := r.filterEnabled(); err != nil {
		return nil, err
	}

	r.mode.Go = true
	return r.enabledCRDs, nil
}

// Validate Config
func (r *Katalog) validateConfig() (*Katalog, error) {
	// Validate config
	// -------------------------------------------------------------------------
	// 1. Field-level validation (required, DNS group, workers <= 5, etc.)
	// -------------------------------------------------------------------------
	if valErr := konfig.Validate().Struct(r); valErr != nil {
		r.handleValidationErrors(valErr)
		return nil, valErr
	}

	// -------------------------------------------------------------------------
	// 2. GVK uniqueness validation
	// -------------------------------------------------------------------------
	if err := r.validateGVKUniqueness(); err != nil {
		logger.Error().Err(err).Msgf("GVK uniqueness error: %v", err)
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 3. dependsOn validation (existence + cycle detection)
	// -------------------------------------------------------------------------
	if err := r.validateDependsOn(); err != nil {
		logger.Error().Err(err).Msgf("dependsOn validation error: %v", err)
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 4. Set GroupVersionKind and Defaults
	// -------------------------------------------------------------------------
	if err := r.SetGroupVersionKind(); err != nil {
		logger.Error().Err(err).Msgf("Set GroupVersionKind error: %v", err)
		return nil, err
	}

	if err := r.SetDefaults(); err != nil {
		logger.Error().Err(err).Msgf("Set Defaults error: %v", err)
		return nil, err
	}

	if r.mode.Yaml {
		// -------------------------------------------------------------------------
		// 5. Add Reconcilers
		// -------------------------------------------------------------------------
		if err := r.addReconcilers(); err != nil {
			logger.Error().Err(err).Msgf("Add Reconcilers error: %v", err)
			return nil, err
		}

		// -------------------------------------------------------------------------
		// 6. Add RuntimeObjects
		// -------------------------------------------------------------------------
		if err := r.addRuntimeObjects(); err != nil {
			logger.Error().Err(err).Msgf("Add RuntimeObjects error: %v", err)
			return nil, err
		}
	}

	return r, nil
}

// Helpers
func (r *Katalog) empty() bool {
	return len(r.crds) == 0
}

func (r *Katalog) enabledEmpty() bool {
	return len(r.enabledCRDs) == 0
}

func (r *Katalog) List() []initialize.CRDEntry {
	return r.crds
}

func (r *Katalog) Enabled() []initialize.CRDEntry {
	return r.enabledCRDs
}

func (r *Katalog) filterEnabled() error {
	if r.empty() {
		return fmt.Errorf("Katalog is empty")
	}

	// Filter enabled CRDs
	for _, crd := range r.crds {
		if crd.Enabled {
			r.enabledCRDs = append(r.enabledCRDs, crd)
		} else {
			logger.Warn().Msgf("%s disabled. skipping...", crd.Name)
		}
	}

	if r.enabledEmpty() {
		return fmt.Errorf("no enabled CRDs found")
	}

	return nil
}
