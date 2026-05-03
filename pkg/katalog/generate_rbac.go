package katalog

import rbacv1 "k8s.io/api/rbac/v1"

// Standard verbs for managed resources.
var defaultVerbs = []string{
	"get", "list", "watch", "create", "update", "patch", "delete",
}

func (k *Katalog) GenerateRBACRules() []rbacv1.PolicyRule {
	var rules []rbacv1.PolicyRule

	// ───────────────────────────────────────────────
	// Base RBAC (always required)
	// ───────────────────────────────────────────────
	rules = append(rules,
		rbacv1.PolicyRule{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "create", "update"},
		},
		rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create", "patch"},
		},
	)

	// ───────────────────────────────────────────────
	// Admission webhook RBAC (conditional)
	// ───────────────────────────────────────────────
	if k.NeedsCertificates() {
		webhookResources := k.WebhookResources()

		if len(webhookResources) > 0 {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: webhookResources,
				Verbs:     defaultVerbs,
			})
		}

		// ───────────────────────────────────────────────
		// Needs permission to create and manage secret
		// ───────────────────────────────────────────────
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     defaultVerbs,
		})
	}

	// ───────────────────────────────────────────────
	// Deletion protection — namespace labeling
	// ───────────────────────────────────────────────
	// ensureNamespaceLabeled patches the Orkestra namespace with deletion-protection
	// labels at startup so the admission webhook's ObjectSelector matches it.
	// Requires get (to confirm the namespace exists) and patch (to apply labels).
	if k.IsDeletionProtectionEnabled() {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get", "patch"},
		})
	}

	// ───────────────────────────────────────────────
	// CRD RBAC (main + status)
	// ───────────────────────────────────────────────
	for _, crd := range k.Enabled() {
		if crd.APITypes.Group == "" || crd.APITypes.Plural == "" {
			if !crd.IsBuiltInType() {
				continue
			}
		}

		// Main resource
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{crd.APITypes.Group},
			Resources: []string{crd.APITypes.Plural},
			Verbs:     defaultVerbs,
		})

		// Status subresource
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{crd.APITypes.Group},
			Resources: []string{crd.APITypes.Plural + "/status"},
			Verbs:     []string{"get", "update", "patch"},
		})

		// CRD patching with CA bundle
		if crd.Conversion != nil && crd.UpdateCRDCaBundle() {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups:     []string{"apiextensions.k8s.io"},
				Resources:     []string{"customresourcedefinitions"},
				Verbs:         []string{"patch"},
				ResourceNames: []string{crd.APITypes.Plural + "." + crd.APITypes.Group},
			})
		}
	}

	// ───────────────────────────────────────────────
	// Table‑driven Kubernetes resource RBAC
	// ───────────────────────────────────────────────
	for key, rule := range rbacRules {
		if k.Uses(key) {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: []string{rule.APIGroup},
				Resources: []string{rule.Resource},
				Verbs:     defaultVerbs,
			})
		}
	}

	return rules
}

// WebhookResources returns the list of admission webhook resources that Orkestra
// needs to manage when webhooks/certificates are required.
//
// Rules:
//   - validatingwebhookconfigurations is required for deletion protection,
//     namespace protection, or any validation rules.
//   - mutatingwebhookconfigurations is required only when mutation rules exist.
//   - conversion webhooks are handled separately and do not require these resources.
func (k *Katalog) WebhookResources() []string {
	var resources []string

	// validatingwebhookconfigurations is needed for:
	// - deletion protection
	// - namespace protection
	// - validation rules (HasValidationRules)
	if k.IsDeletionProtectionEnabled() || k.IsNamespaceProtectionEnabled() || k.HasValidationRules() {
		resources = append(resources, "validatingwebhookconfigurations")
	}

	// mutatingwebhookconfigurations is only needed when mutation rules exist
	if k.HasMutationRules() {
		resources = append(resources, "mutatingwebhookconfigurations")
	}

	return resources
}
