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

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/children"
	"github.com/orkspace/orkestra/pkg/utils"
)

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

	// Add child resources of CRDs created
	// Builtin or not
	gvr := append(k.customResourceGVRs(), orkestraInternalGVRs()...)
	return gvr
}

// orkestraInternalGVRs builds the GVREntry list for Orkestra's own control-plane
// resources by iterating over the children built-in registry and filtering on
// the OrkestraInternal flag.
func orkestraInternalGVRs() []GVREntry {
	var out []GVREntry
	for _, b := range children.AllBuiltInKindDefs() {
		if !b.OrkestraInternal {
			continue
		}
		out = append(out, GVREntry{
			Key:        fmt.Sprintf("%s/%s/%s", b.Group, b.Version, b.Plural),
			Group:      b.Group,
			Version:    b.Version,
			Resource:   b.Plural,
			Operations: []string{"DELETE"},
		})
	}
	return out
}

// customResourceGVRs returns the list of GVRs for all custom resource instances
// managed by this Katalog. These GVRs are used to extend deletion protection
// from the CRD definitions (the types) to individual custom resource instances.
//
// The returned slice is intended to be added to the second webhook in
// registerDeletionProtectionWebhook (the "protect.resources" webhook).
// That webhook intercepts DELETE requests on these GVRs and the handler checks
// whether the specific instance has the label `orkestra.io/deletion-protection=true`.
//
// Built-in resource types (e.g., ConfigMap, Deployment, Namespace) are excluded
// from this list — they are handled separately via the built‑in registry
// (OrkestraInternal flag) and appear in orkestraInternalGVRs().
//
// When running outside the cluster (e.g. `ork run`), the webhook is not reachable
// and this function returns nil (via the caller's guard).
//
// Example:
//
//	For a Katalog enabling "websites.demo.orkestra.io", this returns:
//	[
//	  {
//	    Key: "demo.orkestra.io/v1, Resource=websites",
//	    Group: "demo.orkestra.io",
//	    Version: "v1",
//	    Resource: "websites",
//	    Operations: ["DELETE"]
//	  }
//	]
//
// Important: This function does NOT protect the CRD definitions themselves.
// Those are protected separately by the first webhook (CRD protection) using
// DeletionProtectedCRDNames().
func (k *Katalog) customResourceGVRs() []GVREntry {
	var gvrList []GVREntry
	for _, crd := range k.enabledCRDs {
		if crd.APITypes.Plural == "" || crd.APITypes.Group == "" {
			continue
		}
		gvr := crd.GVR()
		gvrList = append(gvrList, GVREntry{
			Key:        gvr.String(),
			Group:      gvr.Group,
			Version:    gvr.Version,
			Resource:   gvr.Resource,
			Operations: []string{"DELETE"},
		})
	}
	return gvrList
}

// DeletionProtectedCRDNames returns the set of CRD full names managed by this Katalog.
// e.g. {"cronjobs.demo.orkestra.io": {}}
// Used by the /deletion-protection handler for name-based filtering.
// A CRD not in this set is allowed through even though the webhook intercepted it.
// When running outside the cluster (e.g. `ork run`), the webhook cannot be
// reached, so no protection is guaranteed.
func (k *Katalog) DeletionProtectedCRDNames() map[string]struct{} {
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
