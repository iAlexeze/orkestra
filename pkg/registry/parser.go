// pkg/registry/parser.go
package registry

import (
	"os"

	"github.com/ialexeze/multi-crd-controller/pkg/config/initialize"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/config"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"gopkg.in/yaml.v3"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------
func (r *CRDRegistry) buildCRDRegistryFromYaml(path string) ([]initialize.CRDEntry, error) {
	// Check if the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	// Read the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the YAML data
	if err := yaml.Unmarshal(data, r); err != nil {
		return nil, err
	}

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
	if valErr := config.Validate().Struct(r); valErr != nil {
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
	// 4. Set GroupVersionKind
	// -------------------------------------------------------------------------
	r.SetGroupVersionKind()

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
