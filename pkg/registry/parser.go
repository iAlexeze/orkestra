// pkg/registry/parser.go
package registry

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
func (r *CRDRegistry) buildCRDRegistryFromYaml(path string) ([]initialize.CRDEntry, error) {
	data, err := utils.LoadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, r); err != nil {
		return nil, err
	}

	if r.Empty() {
		return nil, fmt.Errorf("CRDRegistry is empty")
	}

	// Populate All crds
	r.AllCRDs = r.CRDs

	// Filter enabled CRDs
	for _, crd := range r.CRDs {
		if crd.Enabled {
			r.EnabledCRDs = append(r.EnabledCRDs, crd)
		} else {
			logger.Warn().Msgf("%s disabled. skipping...", crd.Name)
		}
	}

	if r.EnabledEmpty() {
		return nil, fmt.Errorf("no enabled CRDs found")
	}

	r.CRDs = r.EnabledCRDs
	r.Mode.Yaml = true
	return r.CRDs, nil
}

// -----------------------------------------------------------------------------
//
//	GO Builder
//
// -----------------------------------------------------------------------------
func (r *CRDRegistry) buildCRDRegistryFromGo() []initialize.CRDEntry {
	r.Mode.Go = true
	return initialize.BuildCRDRegistryFromGo()
}

// Validate Config
func (r *CRDRegistry) validateConfig(crds []initialize.CRDEntry) (*CRDRegistry, error) {
	r.CRDs = crds

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
	r.SetGroupVersionKind()
	r.SetDefaults()

	if r.Mode.Yaml {
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
func (r *CRDRegistry) Empty() bool {
	return len(r.CRDs) == 0
}

func (r *CRDRegistry) EnabledEmpty() bool {
	return len(r.EnabledCRDs) == 0
}

func (r *CRDRegistry) List() []initialize.CRDEntry {
	return r.AllCRDs
}

func (r *CRDRegistry) EnabledList() []initialize.CRDEntry {
	return r.EnabledCRDs
}
