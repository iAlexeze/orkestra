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
	if k.HasValidationRules() || k.HasMutationRules() || k.HasConversionPaths() {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{"admissionregistration.k8s.io"},
			Resources: []string{
				"validatingwebhookconfigurations",
				"mutatingwebhookconfigurations",
			},
			Verbs: []string{"get", "create", "update", "patch"},
		})
	}

	// ───────────────────────────────────────────────
	// CRD RBAC (main + status)
	// ───────────────────────────────────────────────
	for _, crd := range k.Spec.CRDs {
		if crd.APITypes.Group == "" || crd.APITypes.Plural == "" {
			continue
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
