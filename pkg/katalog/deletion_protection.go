// pkg/katalog/deletion_protection.go
//
// Deletion protection — computes the webhook rules from the Katalog.
//
// The deletion protection webhook intercepts DELETE operations on:
//  1. All CRDs managed by this operator (apiextensions.k8s.io/v1)
//  2. The Orkestra deployment itself (apps/v1 — selected by label)
//
// DeletionProtectionGVRs returns the GVREntry slice that
// registerDeletionProtectionWebhook passes to buildDeletionProtectionRules.
package katalog

// DeletionProtectionGVRs returns the webhook rules for deletion protection.
//
// Always includes:
//
//   - apiextensions.k8s.io/v1 customresourcedefinitions (DELETE)
//     Filtered by object name — only the CRDs this operator manages.
//     The handler checks the name at request time; the rule is broad.
//
//   - apps/v1 deployments (DELETE)
//     Filtered by label selector in the webhook config — only the Orkestra deployment.
//
// The returned entries are passed to buildDeletionProtectionRules which
// converts them to admissionv1.RuleWithOperations.
func (k *Katalog) DeletionProtectionGVRs() []GVREntry {
	if !k.IsDeletionProtectionEnabled() {
		return nil
	}

	return []GVREntry{
		{
			// Protect all CRDs managed by this operator.
			// The /deletion-protection handler checks the CRD name at request time
			// against the set of CRDs in the Katalog — unrecognised CRDs are allowed.
			Key:        "apiextensions.k8s.io/v1/customresourcedefinitions",
			Group:      "apiextensions.k8s.io",
			Version:    "v1",
			Resource:   "customresourcedefinitions",
			Operations: []string{"DELETE"},
		},
		{
			// Protect the Orkestra operator deployment.
			// The webhook config uses an ObjectSelector to narrow to the specific deployment.
			Key:        "apps/v1/deployments",
			Group:      "apps",
			Version:    "v1",
			Resource:   "deployments",
			Operations: []string{"DELETE"},
		},
	}
}

// ProtectedCRDNames returns the set of CRD full names (e.g. "pipelines.platform.io")
// that should be blocked from deletion.
// Used by the /deletion-protection handler to decide whether to deny a request.
func (k *Katalog) ProtectedCRDNames() map[string]struct{} {
	if !k.IsDeletionProtectionEnabled() {
		return nil
	}

	names := make(map[string]struct{}, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.IsBuiltIn {
			// Built-in types (ConfigMap, Deployment) are not CRDs —
			// they cannot be deleted via the CRD API and need no protection here.
			continue
		}
		// Full CRD name: "pipelines.platform.io"
		plural := crd.APITypes.Plural
		group := crd.APITypes.Group
		if plural != "" && group != "" {
			names[plural+"."+group] = struct{}{}
		}
	}
	return names
}
