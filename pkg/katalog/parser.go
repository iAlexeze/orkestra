// pkg/katalog/parsek.go
package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// -----------------------------------------------------------------------------
//
//	YAML Builder
//
// -----------------------------------------------------------------------------
func (k *Katalog) KomposeRuntimeKatalog(kfg *konfig.Konfig, m *merger.Merger, paths ...string) (map[string]orktypes.CRDEntry, error) {
	k.Spec = m.ToSpec()
	k.Security = m.ToSecurity()
	k.Notification = m.ToNotification()
	k.Providers = m.ToProviders()
	k.projectInfo = m.ToProjectInfo()
	k.enabledCRDs = m.Enabled()           // Enabled CRDs for all operations
	k.metadata = m.APIMetadata().Metadata // Metadata for CLI and health endpoints
	k.APIVersion = m.APIMetadata().APIVersion
	k.Kind = m.APIMetadata().Kind
	k.konfig = kfg
	k.katalogDir = m.FirstEntryDir()

	// Debug
	// comment after use
	// k.DebugKatalogInformation()

	for name, entry := range k.enabledCRDs {
		// Populate APITypes from crdFile before enrichment so isFullySpecified sees
		// the correct values. crdFile is the source of truth — overwrites any apiTypes.
		if entry.CRDFile != "" {
			if err := populateAPITypesFromCRDFile(&entry, k.katalogDir); err != nil {
				return nil, fmt.Errorf("CRD %q: %w", name, err)
			}
		}

		// Enrich enabled CRDs
		outcome, err := EnrichCRDEntry(&entry)
		if err != nil {
			return nil, err
		}
		entry.EnrichmentOutcome = outcome
		k.enabledCRDs[name] = entry
	}

	// Expand Motif imports declared in each operatorBox
	if err := k.expandMotifImports(); err != nil {
		return nil, err
	}

	// initialize conversion registry and admission registry
	k.conversionRegistry = NewInMemoryConversionRegistry()
	k.admissionRegistry = NewInMemoryAdmissionRegistry()

	// now safe to register rules
	for _, entry := range k.enabledCRDs {
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
	// 5. Validate Reconciler modes
	// -------------------------------------------------------------------------
	if err := k.validateReconcilerMode(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 6. Add Reconcilers		// ReconcilerRegistry → Constructor
	// -------------------------------------------------------------------------
	if err := k.addReconcilers(); err != nil {
		return nil, err
	}
	// -------------------------------------------------------------------------
	// 7. Add RuntimeObjects	// ObjectRegistry + ListRegistry
	// -------------------------------------------------------------------------
	if err := k.addRuntimeObjects(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 8. Add Hooks	// HookRegistry → HookFactory
	// -------------------------------------------------------------------------
	if err := k.addHooks(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 9. Validate Status
	// -------------------------------------------------------------------------
	k.validateStatus()

	// -------------------------------------------------------------------------
	// 10. Validate Autoscale Profile
	// -------------------------------------------------------------------------
	if err := k.validateAutoscaleProfile(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 11. Validate Resource Profile
	// -------------------------------------------------------------------------
	if err := k.validateResourceProfile(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 11a. Validate Probe Profiles
	// -------------------------------------------------------------------------
	if err := k.validateProbeProfiles(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 12. Validate Autoscale Metrics Type
	// -------------------------------------------------------------------------
	if err := k.validateAutoscalerMetrics(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 13. Validate Namespace protection
	// -------------------------------------------------------------------------
	if err := k.validateNamespaceProtection(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 14. Validate Time Duration
	// -------------------------------------------------------------------------
	if err := k.validateTimeDuration(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 15. Validate HPA Reference
	// -------------------------------------------------------------------------
	if err := k.validateHPAReference(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 16. Validate Notify Teams
	// -------------------------------------------------------------------------
	if err := k.validateTeams(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 17. Validate Status Types
	// -------------------------------------------------------------------------
	if err := k.validateStatusTypes(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 18. Validate Services
	// -------------------------------------------------------------------------
	if err := k.validateService(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 19. Validate CustomResources
	// -------------------------------------------------------------------------
	if err := k.validateCustomResources(); err != nil {
		return nil, err
	}

	return k, nil
}
