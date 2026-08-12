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
	// 2. Deprecation timeline validation
	// -------------------------------------------------------------------------
	if err := k.validateDeprecation(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 3. Uniqueness validation
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

	// 26. Validate user-defined profiles (uniqueness, shadowing warnings)
	// -------------------------------------------------------------------------
	if err := k.validateUserProfiles(); err != nil {
		return nil, err
	}

	// 26c. Validate user-defined notes (uniqueness, valid template, shadowing)
	// -------------------------------------------------------------------------
	if err := k.validateUserNotes(); err != nil {
		return nil, err
	}

	// 26b. Validate HPA Behavior Profiles
	// -------------------------------------------------------------------------
	if err := k.validateHPABehaviorProfiles(); err != nil {
		return nil, err
	}

	// 27. Validate PDB Behavior Profiles
	// -------------------------------------------------------------------------
	if err := k.validatePDBBehaviorProfiles(); err != nil {
		return nil, err
	}

	// 28. Validate Rolling Update Profiles
	// -------------------------------------------------------------------------
	if err := k.validateRollingUpdateProfiles(); err != nil {
		return nil, err
	}

	// 30. Validate NetworkPolicy Profiles
	// -------------------------------------------------------------------------
	if err := k.validateNetworkPolicyProfiles(); err != nil {
		return nil, err
	}

	// 31. Validate ResourceQuota Profiles
	// -------------------------------------------------------------------------
	if err := k.validateResourceQuotaProfiles(); err != nil {
		return nil, err
	}

	if err := k.validateLimitRangeProfiles(); err != nil {
		return nil, err
	}

	// 32. Validate cross-namespace copy pairs (fromNamespace ↔ toNamespaces)
	// -------------------------------------------------------------------------
	if err := k.validateCrossNamespaceOps(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 29. Validate port protocols (Deployments, ReplicaSets, StatefulSets, Pods)
	// -------------------------------------------------------------------------
	if err := k.validatePortProtocols(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 33. Validate workload autoscale blocks on Deployment declarations
	// -------------------------------------------------------------------------
	if err := k.validateWorkloadAutoscale(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 34. Validate external call lists (unique non-empty names)
	// -------------------------------------------------------------------------
	if err := k.validateExternalCalls(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 35. Validate envFrom refs (suffix requires keys)
	// -------------------------------------------------------------------------
	if err := k.validateEnvFromRefs(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 36. Validate Serve configuration
	// -------------------------------------------------------------------------
	if err := k.ValidateServe(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 37. Validate validation.rules / mutation.rules operators are known
	// -------------------------------------------------------------------------
	if err := k.validateAdmissionOperators(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 38. Validate validation.rules link: values
	// -------------------------------------------------------------------------
	if err := k.validateValidationRuleLinks(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 41. Validate gateway tokens (no duplicates, source exclusivity, OIDC rules)
	// -------------------------------------------------------------------------
	if err := k.validateGatewayTokens(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 42. Validate gateway webhooks (intake sources)
	// -------------------------------------------------------------------------
	if err := k.validateGatewayWebhooks(); err != nil {
		return nil, err
	}

	// -------------------------------------------------------------------------
	// 34. Validate Publish config
	// -------------------------------------------------------------------------
	if err := k.validatePublish(); err != nil {
		return nil, err
	}

	return k, nil
}
