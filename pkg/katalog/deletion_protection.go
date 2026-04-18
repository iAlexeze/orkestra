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

import "github.com/orkspace/orkestra/pkg/utils"

// DeletionProtectionGVRs returns the list of GVRs that the deletion‑protection
// admission webhook should intercept.
//
// This includes:
//   - CRDs managed by this Katalog (broad match; handler filters by name)
//   - Orkestra’s own admission webhooks (validating + mutating)
//   - Orkestra’s internal control‑plane resources (deployment, service,
//     serviceaccount, configmap, RBAC objects, ingress, etc.)
//
// The internal resources are derived from the built‑ins registry via
// OrkestraInternalGVRs(), ensuring the list is declarative and maintained
// in a single place.
//
// When running outside the cluster (e.g. `ork run`), the webhook cannot be
// reached, so no rules are returned.
func (k *Katalog) DeletionProtectionGVRs() []GVREntry {
	if !k.IsDeletionProtectionEnabled() {
		return nil
	}
	if !utils.IsRunningInCluster() {
		return nil
	}

	// Protect all CRDs managed by this Katalog and
	// Orkestra’s internal control‑plane resources.
	// These are defined declaratively in the built‑ins registry via
	// the OrkestraInternal flag.
	return OrkestraInternalGVRs()
}

// ProtectedCRDNames returns the set of CRD full names managed by this Katalog.
// e.g. {"cronjobs.demo.orkestra.io": {}}
// Used by the /deletion-protection handler for name-based filtering.
// A CRD not in this set is allowed through even though the webhook intercepted it.
// When running outside the cluster (e.g. `ork run`), the webhook cannot be
// reached, so no protection is guaranteed.
func (k *Katalog) ProtectedCRDNames() map[string]struct{} {
	if !k.IsDeletionProtectionEnabled() {
		return nil
	}

	if !utils.IsRunningInCluster() {
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
