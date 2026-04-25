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
func (k *Katalog) KomposeKatalogFromYaml(kfg *konfig.Konfig, m *merger.Merger, paths ...string) (map[string]orktypes.CRDEntry, error) {
	k.Spec = m.ToSpec()
	k.Security = m.ToSecurity()
	k.Notification = m.ToNotification()
	k.Providers = m.ToProviders()
	k.enabledCRDs = m.Enabled()           // Enabled CRDs for all operations
	k.metadata = m.APIMetadata().Metadata // Metadata for CLI and health endpoints
	k.APIVersion = m.APIMetadata().APIVersion
	k.Kind = m.APIMetadata().Kind
	k.konfig = kfg

	// Debug
	// comment after use
	// k.DebugKatalogInformation()

	// Enrich enabled CRDs
	// Switching from slice to map — must copy back since map values are not addressable
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
		k.admissionRegistry.registerAdmissionRulesFromEntry(entry)
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

	// -------------------------------------------------------------------------
	// 11. Validate Namespace protection
	// -------------------------------------------------------------------------
	if err := k.validateNamespaceProtection(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 12. Validate Time Duration
	// -------------------------------------------------------------------------
	if err := k.validateTimeDuration(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 13. Validate HPA Reference
	// -------------------------------------------------------------------------
	if err := k.validateHPAReference(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 14. Validate Notify Teams
	// -------------------------------------------------------------------------
	// if err := k.validateNotifyTeams(); err != nil {
	// 	return nil, err
	// }

	// -------------------------------------------------------------------------
	// 15 Validate Status Types
	// -------------------------------------------------------------------------
	if err := k.validateStatusTypes(); err != nil {
		return nil, err
	}

	return k, nil
}

// Debug katalog information from merger
func (k *Katalog) DebugKatalogInformation() {
	// [DEBUG] Contents of k.Security
	logger.Debug().Interface("katalog security", k.Security).Msg("katalog security")

	// [DEBUG] Contents of k.Providers
	logger.Debug().Interface("katalog providers", k.Providers).Msg("katalog providers")

	// [DEBUG] Contents of k.Spec
	logger.Debug().Interface("katalog spec", k.Spec).Msg("katalog spec")

	// [DEBUG] Contents of k.enabledCRDs
	logger.Debug().Interface("katalog enabledCRDs", k.enabledCRDs).Msg("katalog enabledCRDs")

	// [DEBUG] Contents of k.metadata
	logger.Debug().Interface("katalog metadata", k.metadata).Msg("katalog metadata")
}
