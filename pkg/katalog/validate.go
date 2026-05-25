package katalog

import "github.com/orkspace/orkestra/pkg/konfig"

// Validate Config
func (k *Katalog) ValidateConfig(kfg *konfig.Konfig) (*Katalog, error) {

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
	// 12. Validate Probe Profiles
	// -------------------------------------------------------------------------
	if err := k.validateProbeProfiles(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 13. Validate Autoscale Metrics Type
	// -------------------------------------------------------------------------
	if err := k.validateAutoscalerMetrics(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 14. Validate Namespace protection
	// -------------------------------------------------------------------------
	if err := k.validateNamespaceProtection(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 15. Validate Time Duration
	// -------------------------------------------------------------------------
	if err := k.validateTimeDuration(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 16. Validate HPA Reference
	// -------------------------------------------------------------------------
	if err := k.validateHPAReference(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 17. Validate Notify Teams
	// -------------------------------------------------------------------------
	if err := k.validateTeams(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 18. Validate Status Types
	// -------------------------------------------------------------------------
	if err := k.validateStatusTypes(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 19. Validate Services
	// -------------------------------------------------------------------------
	if err := k.validateService(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 20. Validate CustomResources
	// -------------------------------------------------------------------------
	if err := k.validateCustomResources(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 21. Validate Gateway requirement
	// Security features and notifications require a companion gateway process.
	// Fail fast so users are never surprised by silently inactive features.
	// -------------------------------------------------------------------------
	if err := k.validateGateway(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 22. Validate Enrich config
	// -------------------------------------------------------------------------
	if err := k.validateEnrich(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 23. Validate Deletion Protection
	// -------------------------------------------------------------------------
	if err := k.validateDeletionProtection(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 24. Validate Security Profiles
	// -------------------------------------------------------------------------
	if err := k.validateSecurityProfiles(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 25. Validate Capability Names
	// -------------------------------------------------------------------------
	if err := k.validateSecurityCapabilities(); err != nil {
		return nil, err
	}

	return k, nil
}
