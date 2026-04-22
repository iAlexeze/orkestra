package katalog

import (
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Methods to maintain the zero footprint promise of orkestra
//
// HasConversionPaths returns true only if:
//  1. Conversion is enabled in konfig, AND
//  2. At least one CRD declares conversion paths.
//
// This protects the zero‑footprint promise:
// Orkestra exposes /convert ONLY when the user explicitly declares conversion.
func (k *Katalog) HasConversionPaths() bool {
	// Katalog or konfig may be nil in edge cases (e.g., NewEmptyKatalog)
	if k == nil || k.konfig == nil {
		logger.Debug().Msg("katalog or konfig is nil")
		return false
	}

	if !k.IsConversionEnabled() {
		logger.Debug().Msg("conversion is disabled")
		return false
	}

	for _, crd := range k.Enabled() {
		// Protect against nil Conversion or nil Paths
		if crd.Conversion == nil {
			logger.Debug().Str("crd", crd.Name).Msg("conversion is nil")
			continue
		}
		// Protect against empty Paths
		if len(crd.Conversion.Paths) > 0 {
			logger.Debug().Str("crd", crd.Name).Msg("conversion has paths")
			return true
		}
	}
	return false
}

// HasValidationRules returns true only if:
//  1. Admission is enabled in konfig, AND
//  2. At least one CRD declares validation rules.
//
// This ensures /validate is created ONLY when the user declares rules.
func (k *Katalog) HasValidationRules() bool {
	if k == nil || k.konfig == nil {
		logger.Debug().Msg("katalog or konfig is nil")
		return false
	}

	if !k.IsAdmissionEnabled() {
		logger.Debug().Msg("admission is disabled")
		return false
	}

	for _, crd := range k.Enabled() {
		if crd.Validation == nil {
			logger.Debug().Str("crd", crd.Name).Msg("validation is nil")
			continue
		}

		if len(crd.Validation.Rules) > 0 {
			logger.Debug().Str("crd", crd.Name).Msg("validation has rules")
			return true
		}
	}
	return false
}

// HasMutationRules returns true only if:
//  1. Admission is enabled in konfig, AND
//  2. At least one CRD declares mutation rules.
//
// This ensures /mutate is created ONLY when the user declares rules.
func (k *Katalog) HasMutationRules() bool {
	if k == nil || k.konfig == nil {
		logger.Debug().Msg("katalog or konfig is nil")
		return false
	}

	if !k.IsAdmissionEnabled() {
		logger.Debug().Msg("admission is disabled")
		return false
	}

	for _, crd := range k.Enabled() {
		// Protect against nil Mutation or nil Rules
		if crd.Mutation == nil {
			logger.Debug().Str("crd", crd.Name).Msg("mutation is nil")
			continue
		}

		if len(crd.Mutation.Rules) > 0 {
			logger.Debug().Str("crd", crd.Name).Msg("mutation has rules")
			return true
		}
	}
	return false
}

// HasValidationOrMutationRules returns true only if:
// - There is at least one CRD that declares validation or mutation rules.
// - Admission is enabled in konfig.
func (k *Katalog) HasValidationOrMutationRules() bool {
	if k == nil || k.konfig == nil {
		logger.Debug().Msg("katalog or konfig is nil")
		return false
	}

	if k.HasValidationRules() || k.HasMutationRules() {
		return true
	}
	return false
}

// Uses returns true if ANY CRD in the katalog uses the given resource type.
func (k *Katalog) Uses(resource string) bool {
	check, ok := orktypes.ResourceChecks[resource]
	if !ok {
		return false
	}
	for _, crd := range k.Enabled() {
		if check.Get(crd) {
			return true
		}
	}
	return false
}

// RBACRule describes how to generate RBAC for a Kubernetes resource.
type RBACRule struct {
	APIGroup string
	Resource string
	Verbs    []string
}

// Table‑driven RBAC rules for all supported Kubernetes resources.
var rbacRules = map[string]RBACRule{
	"services":                 {APIGroup: "", Resource: "services"},
	"configmaps":               {APIGroup: "", Resource: "configmaps"},
	"secrets":                  {APIGroup: "", Resource: "secrets"},
	"persistentvolumeclaims":   {APIGroup: "", Resource: "persistentvolumeclaims"},
	"serviceaccounts":          {APIGroup: "", Resource: "serviceaccounts"},
	"podtemplates":             {APIGroup: "", Resource: "podtemplates"},
	"volumes":                  {APIGroup: "", Resource: "volumes"},
	"volumemounts":             {APIGroup: "", Resource: "volumemounts"},
	"deployments":              {APIGroup: "apps", Resource: "deployments"},
	"statefulsets":             {APIGroup: "apps", Resource: "statefulsets"},
	"daemonsets":               {APIGroup: "apps", Resource: "daemonsets"},
	"jobs":                     {APIGroup: "batch", Resource: "jobs"},
	"cronjobs":                 {APIGroup: "batch", Resource: "cronjobs"},
	"ingresses":                {APIGroup: "networking.k8s.io", Resource: "ingresses"},
	"roles":                    {APIGroup: "rbac.authorization.k8s.io", Resource: "roles"},
	"rolebindings":             {APIGroup: "rbac.authorization.k8s.io", Resource: "rolebindings"},
	"horizontalpodautoscalers": {APIGroup: "autoscaling", Resource: "horizontalpodautoscalers"},
	"poddisruptionbudgets":     {APIGroup: "policy", Resource: "poddisruptionbudgets"},
	"networkpolicies":          {APIGroup: "networking.k8s.io", Resource: "networkpolicies"},
}
