// pkg/katalog/parsek.go
package katalog

import (
	// "fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) KomposeKatalogFromYaml(m *merger.Merger, paths ...string) (map[string]orktypes.CRDEntry, error) {
	k.Spec = m.ToSpec()
	k.Security = m.ToSecurity()
	k.Providers = m.ToProviders()
	k.enabledCRDs = m.Enabled()           // Enabled CRDs for all operations
	k.metadata = m.APIMetadata().Metadata // Metadata for CLI and health endpoints
	k.APIVersion = m.APIMetadata().APIVersion
	k.Kind = m.APIMetadata().Kind

	// Enrich enabled CRDs — must copy back since map values are not addressable
	for name, entry := range k.enabledCRDs {
		outcome, err := EnrichCRDEntry(&entry)
		if err != nil {
			return nil, err
		}
		entry.EnrichmentOutcome = outcome
		k.enabledCRDs[name] = entry
	}

	// initialize conversion registry and admission registry
	k.conversionRegistry = NewInMemoryConversionRegistry()
	k.admissionRegistry = NewInMemoryAdmissionRegistry()

	// now safe to register rules
	for _, entry := range k.Spec.CRDs {
		k.admissionRegistry.RegisterAdmissionRulesFromEntry(entry)
		k.conversionRegistry.registerConversionRulesFromSpec(entry)
	}

	return k.enabledCRDs, nil
}

// Validate Config
func (k *Katalog) ValidateConfig(kfg *konfig.Konfig) (*Katalog, error) {
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

	if err := k.setDefaults(kfg); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 5. Add Reconcilers		// ReconcilerRegistry → Constructor
	// -------------------------------------------------------------------------
	if err := k.addReconcilers(); err != nil {
		logger.Error().Err(err).Msg("failed to add reconcilers")
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

	// -------------------------------------------------------------------------
	// 8. Validate Status
	// -------------------------------------------------------------------------
	k.validateStatus()

	// -------------------------------------------------------------------------
	// 9. Validate Autoscale Profile
	// -------------------------------------------------------------------------
	if err := k.validateAutoscaleProfile(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 10. Validate Autoscale Metrics Type
	// -------------------------------------------------------------------------
	if err := k.validateAutoscalerMetrics(); err != nil {
		return nil, err
	}

	return k, nil
}
