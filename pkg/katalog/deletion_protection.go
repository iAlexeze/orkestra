// pkg/katalog/deletion_protection.go
//
// Deletion protection — webhook rules and protected CRD name resolution.
//
// Architecture: two-level filtering.
//
//	Level 1 — Webhook rules (DeletionProtectionGVRs):
//	  Intercepts ALL DELETE on customresourcedefinitions and Orkestra deployments.
//	  Must be broad — Kubernetes webhook rules filter by GVR, not by object name.
//
//	Level 2 — Handler (isProtectedCRD):
//	  Narrows to only the CRDs managed by THIS Katalog.
//	  "websites.demo.orkestra.io" from a different operator → allowed.
//	  "cronjobs.demo.orkestra.io" from this Katalog → denied.
//
// When to register the webhook:
//
//	Requires a reachable Kubernetes Service. With ork run the operator runs
//	locally — there is no Service. failurePolicy: Fail would block ALL CRD
//	deletions when unreachable. Only register when running inside the cluster.
package katalog

import "os"

// DeletionProtectionGVRs returns the webhook intercept rules.
// Returns nil when running outside the cluster (ork run) — in that case
// the webhook cannot be reached and must not be registered.
func (k *Katalog) DeletionProtectionGVRs() []GVREntry {
	if !k.IsDeletionProtectionEnabled() {
		return nil
	}
	if !isRunningInCluster() {
		return nil
	}

	return []GVREntry{
		{
			// Broad rule — handler filters to managed CRDs via ProtectedCRDNames()
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

// ProtectedCRDNames returns the set of CRD full names managed by this Katalog.
// e.g. {"cronjobs.demo.orkestra.io": {}}
// Used by the /deletion-protection handler for name-based filtering.
// A CRD not in this set is allowed through even though the webhook intercepted it.
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
		if crd.APITypes.Plural != "" && crd.APITypes.Group != "" {
			names[crd.APITypes.Plural+"."+crd.APITypes.Group] = struct{}{}
		}
	}
	return names
}

// isRunningInCluster returns true when running inside a Kubernetes pod.
// The service account token is always present inside a pod.
func isRunningInCluster() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return err == nil
}
