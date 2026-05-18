package katalog

import (
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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
	// Typed‑mode RBAC (hooks or constructor)
	// ───────────────────────────────────────────────
	for _, crd := range k.Enabled() {

		// Hooks-managed resources
		if crd.WithHookManagedResources() {
			for _, r := range crd.HookManagedResources() {
				gvr, ok := k.ResolveGVR(r)
				if !ok {
					// optional: skip or log
					continue
				}
				rules = append(rules, rbacv1.PolicyRule{
					APIGroups: []string{gvr.Group},
					Resources: []string{gvr.Resource},
					Verbs:     defaultVerbs,
				})
			}
		}

		// Constructor-managed resources
		if crd.WithConstructorManagedResources() {
			for _, r := range crd.ConstructorManagedResources() {
				gvr, ok := k.ResolveGVR(r)
				if !ok {
					// optional: skip or log
					continue
				}
				rules = append(rules, rbacv1.PolicyRule{
					APIGroups: []string{gvr.Group},
					Resources: []string{gvr.Resource},
					Verbs:     defaultVerbs,
				})
			}
		}
	}

	// ───────────────────────────────────────────────
	// Built-in resource RBAC — driven by builtInRegistry.
	// Any entry with a Detect function is a candidate; emit a rule when
	// at least one enabled CRD actually uses that resource.
	// ───────────────────────────────────────────────
	for _, b := range builtInRegistry {
		if b.Detect == nil {
			continue
		}
		if k.anyDetects(b.Detect) {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: []string{b.Group},
				Resources: []string{b.Plural},
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

// ResolveGVR resolves a ManagedResource into a concrete GroupVersionResource.
//
// Resolution priority (explicit always wins):
//
//  1. Full explicit GVR:
//     - group + version + plural are all provided
//     → use them directly.
//
//  2. Explicit group + version (plural omitted)
//     → infer plural as strings.ToLower(kind) + "s".
//
//  3. APIVersion + plural:
//     - apiVersion: "group/version"
//     - plural provided
//     → parse apiVersion and use provided plural.
//
//  4. APIVersion only:
//     - apiVersion: "group/version"
//     - plural omitted
//     → parse apiVersion and infer plural as strings.ToLower(kind) + "s".
//
//  5. Built‑in Kubernetes resource:
//     - kind matches Orkestra's built‑in registry
//     → use GVRForBuiltIn(kind).
//
//  6. Otherwise:
//     → resolution fails and (GVR{}, false) is returned.
//
// This ensures:
//   - Explicit declarations always override inference.
//   - Custom resources can be fully specified without guessing.
//   - Built‑ins remain simple (kind‑only).
//   - RBAC generation remains deterministic and zero‑footprint safe.
func (k *Katalog) ResolveGVR(r orktypes.ManagedResource) (schema.GroupVersionResource, bool) {
	// ───────────────────────────────────────────────
	// 1. Full explicit GVR: group + version + plural
	// ───────────────────────────────────────────────
	if r.Group != "" && r.Version != "" && r.Plural != "" {
		return schema.GroupVersionResource{
			Group:    r.Group,
			Version:  r.Version,
			Resource: r.Plural,
		}, true
	}

	// ───────────────────────────────────────────────
	// 2. Explicit group + version, infer plural
	// ───────────────────────────────────────────────
	if r.Group != "" && r.Version != "" {
		return schema.GroupVersionResource{
			Group:    r.Group,
			Version:  r.Version,
			Resource: strings.ToLower(r.Kind) + "s",
		}, true
	}

	// ───────────────────────────────────────────────
	// 3. APIVersion + plural
	// ───────────────────────────────────────────────
	if r.APIVersion != "" && r.Plural != "" {
		gv, err := schema.ParseGroupVersion(r.APIVersion)
		if err != nil {
			return schema.GroupVersionResource{}, false
		}
		return schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: r.Plural,
		}, true
	}

	// ───────────────────────────────────────────────
	// 4. APIVersion only, infer plural
	// ───────────────────────────────────────────────
	if r.APIVersion != "" {
		gv, err := schema.ParseGroupVersion(r.APIVersion)
		if err != nil {
			return schema.GroupVersionResource{}, false
		}
		return schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: strings.ToLower(r.Kind) + "s",
		}, true
	}

	// ───────────────────────────────────────────────
	// 5. Built‑in resource
	// ───────────────────────────────────────────────
	if gvr, ok := GVRForBuiltIn(r.Kind); ok {
		return gvr, true
	}

	// ───────────────────────────────────────────────
	// 6. Could not resolve
	// ───────────────────────────────────────────────
	return schema.GroupVersionResource{}, false
}
