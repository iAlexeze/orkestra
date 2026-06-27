package katalog

import (
	"strings"

	"github.com/orkspace/orkestra/pkg/children"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Standard verbs for managed resources.
var defaultVerbs = []string{
	"get", "list", "watch", "create", "update", "patch", "delete",
}

// rbacVerbsFor returns the appropriate verb set for a built-in resource.
//
// Roles and ClusterRoles require two extra verbs beyond standard CRUD:
//   - "escalate": allows creating/updating a Role or ClusterRole that grants
//     permissions the Orkestra SA does not already hold. Without it Kubernetes
//     blocks any attempt to provision a role with a broader permission set.
//   - "bind": allows creating a RoleBinding or ClusterRoleBinding that
//     references a Role/ClusterRole whose permissions the SA doesn't hold.
//     Without it the binding creation is blocked even after the role exists.
//
// Both verbs are required whenever the operator provisions RBAC on behalf of
// tenant service accounts (e.g. via clusterRoles:/roles: in onCreate).
// They are absent from the generated bundle when no Roles or ClusterRoles are
// detected in the Katalog, preserving least-privilege for all other operators.
func rbacVerbsFor(group, plural string) []string {
	if group == "rbac.authorization.k8s.io" && (plural == "roles" || plural == "clusterroles") {
		return append(defaultVerbs, "escalate", "bind")
	}
	return defaultVerbs
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

		// Custom CRs registered in the deletion-protection webhook need GET so
		// the gateway can read the instance and check its protection label/annotation.
		rules = append(rules, k.customResourceDeletionProtectionRBACRules()...)
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

		// CRD patching with CA bundle and watching for MODIFIED event by housekeeper
		if crd.Conversion != nil && crd.UpdateCRDCaBundle() {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups:     []string{"apiextensions.k8s.io"},
				Resources:     []string{"customresourcedefinitions"},
				Verbs:         []string{"patch", "watch"},
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
	for _, b := range children.AllBuiltInKindDefs() {
		if b.Detect == nil {
			continue
		}
		if k.anyDetects(b.Detect) {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: []string{b.Group},
				Resources: []string{b.Plural},
				Verbs:     rbacVerbsFor(b.Group, b.Plural),
			})
		}
	}

	// ───────────────────────────────────────────────
	// Custom resource RBAC — derived from onCreate/onReconcile custom: entries.
	// Built-in kinds are covered above; third-party CRDs (cert-manager, ArgoCD,
	// Crossplane, etc.) are not in the built-in registry and must be emitted here.
	// ───────────────────────────────────────────────
	rules = append(rules, k.customResourceRBACRules()...)

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

// GenerateRuntimeRBACRules returns the RBAC rules required by the runtime reconciler process.
// This is GenerateRBACRules() minus the NeedsCertificates() block (webhook/secrets),
// minus the IsDeletionProtectionEnabled() namespace block, and minus CRD CA-bundle patch rules.
func (k *Katalog) GenerateRuntimeRBACRules() []rbacv1.PolicyRule {
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
	// CRD RBAC (main + status, no CA bundle patch)
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
	// Built-in resource RBAC
	// ───────────────────────────────────────────────
	for _, b := range children.AllBuiltInKindDefs() {
		if b.Detect == nil {
			continue
		}
		if k.anyDetects(b.Detect) {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: []string{b.Group},
				Resources: []string{b.Plural},
				Verbs:     rbacVerbsFor(b.Group, b.Plural),
			})
		}
	}

	// ───────────────────────────────────────────────
	// Custom resource RBAC
	// ───────────────────────────────────────────────
	rules = append(rules, k.customResourceRBACRules()...)

	return rules
}

// GenerateGatewayRBACRules returns the RBAC rules required by the gateway process
// (webhook server, certificate management, namespace labeling).
func (k *Katalog) GenerateGatewayRBACRules() []rbacv1.PolicyRule {
	if !k.IsGatewayEnabled() {
		return nil
	}

	var rules []rbacv1.PolicyRule

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

		// Needs permission to create and manage secret
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     defaultVerbs,
		})
	}

	// ───────────────────────────────────────────────
	// Deletion protection — namespace labeling
	// ───────────────────────────────────────────────
	if k.IsDeletionProtectionEnabled() {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get", "patch"},
		})
	}

	// ───────────────────────────────────────────────
	// CRD CA bundle patching (conversion webhooks)
	// ───────────────────────────────────────────────
	for _, crd := range k.Enabled() {
		if crd.Conversion != nil && crd.UpdateCRDCaBundle() {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups:     []string{"apiextensions.k8s.io"},
				Resources:     []string{"customresourcedefinitions"},
				Verbs:         []string{"patch", "watch"},
				ResourceNames: []string{crd.APITypes.Plural + "." + crd.APITypes.Group},
			})
		}
	}

	return rules
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
//     → use children.GVRForBuiltIn(kind).
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
	if gvr, ok := children.GVRForBuiltIn(r.Kind); ok {
		return gvr, true
	}

	// ───────────────────────────────────────────────
	// 6. Could not resolve
	// ───────────────────────────────────────────────
	return schema.GroupVersionResource{}, false
}

// GeneratePerCRDRBACRules returns the RBAC rules attributed to each enabled CRD.
// The map key is the CRD name. Excludes system-level rules (leases, events,
// secrets, namespaces, webhook configurations) — use GenerateRuntimeRBACRules /
// GenerateGatewayRBACRules for those.
func (k *Katalog) GeneratePerCRDRBACRules() map[string][]rbacv1.PolicyRule {
	result := make(map[string][]rbacv1.PolicyRule, len(k.enabledCRDs))

	for name, crd := range k.Enabled() {
		var rules []rbacv1.PolicyRule

		if crd.APITypes.Group != "" || crd.IsBuiltInType() {
			if crd.APITypes.Plural != "" {
				rules = append(rules,
					rbacv1.PolicyRule{
						APIGroups: []string{crd.APITypes.Group},
						Resources: []string{crd.APITypes.Plural},
						Verbs:     defaultVerbs,
					},
					rbacv1.PolicyRule{
						APIGroups: []string{crd.APITypes.Group},
						Resources: []string{crd.APITypes.Plural + "/status"},
						Verbs:     []string{"get", "update", "patch"},
					},
				)
			}
			if crd.Conversion != nil && crd.UpdateCRDCaBundle() {
				rules = append(rules, rbacv1.PolicyRule{
					APIGroups:     []string{"apiextensions.k8s.io"},
					Resources:     []string{"customresourcedefinitions"},
					Verbs:         []string{"patch"},
					ResourceNames: []string{crd.APITypes.Plural + "." + crd.APITypes.Group},
				})
			}
		}

		if crd.WithHookManagedResources() {
			for _, r := range crd.HookManagedResources() {
				if gvr, ok := k.ResolveGVR(r); ok {
					rules = append(rules, rbacv1.PolicyRule{
						APIGroups: []string{gvr.Group},
						Resources: []string{gvr.Resource},
						Verbs:     defaultVerbs,
					})
				}
			}
		}

		if crd.WithConstructorManagedResources() {
			for _, r := range crd.ConstructorManagedResources() {
				if gvr, ok := k.ResolveGVR(r); ok {
					rules = append(rules, rbacv1.PolicyRule{
						APIGroups: []string{gvr.Group},
						Resources: []string{gvr.Resource},
						Verbs:     defaultVerbs,
					})
				}
			}
		}

		for _, b := range children.AllBuiltInKindDefs() {
			if b.Detect != nil && b.Detect(crd) {
				rules = append(rules, rbacv1.PolicyRule{
					APIGroups: []string{b.Group},
					Resources: []string{b.Plural},
					Verbs:     defaultVerbs,
				})
			}
		}

		for _, rule := range customRBACRulesForCRD(crd) {
			rules = append(rules, rule)
		}

		result[name] = rules
	}

	return result
}

// customResourceRBACRules collects RBAC rules for all third-party CRDs declared
// in onCreate/onReconcile custom: blocks across every enabled CRD.
// Built-in Kubernetes kinds are already covered by the builtInRegistry loop;
// this handles everything else (cert-manager, ArgoCD, Crossplane, etc.).
func (k *Katalog) customResourceRBACRules() []rbacv1.PolicyRule {
	var rules []rbacv1.PolicyRule
	seen := make(map[string]bool)
	for _, crd := range k.Enabled() {
		for _, rule := range customRBACRulesForCRD(crd) {
			key := strings.Join(rule.APIGroups, ",") + "/" + strings.Join(rule.Resources, ",")
			if !seen[key] {
				seen[key] = true
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

// customRBACRulesForCRD derives RBAC rules from the custom: entries of one CRD's
// onCreate and onReconcile blocks. Each entry's apiVersion + kind is resolved into
// a group + plural via ParseGroupVersion and lowercase+s inference.
func customRBACRulesForCRD(crd orktypes.CRDEntry) []rbacv1.PolicyRule {
	var entries []orktypes.CustomResourceTemplateSource
	if crd.OperatorBox.OnCreate != nil {
		entries = append(entries, crd.OperatorBox.OnCreate.CustomResource...)
	}
	if crd.OperatorBox.OnReconcile != nil {
		entries = append(entries, crd.OperatorBox.OnReconcile.CustomResource...)
	}

	seen := make(map[string]bool)
	var rules []rbacv1.PolicyRule
	for _, entry := range entries {
		if entry.APIVersion == "" || entry.Kind == "" {
			continue
		}
		gv, err := schema.ParseGroupVersion(entry.APIVersion)
		if err != nil {
			continue
		}
		plural := strings.ToLower(entry.Kind) + "s"
		key := gv.Group + "/" + plural
		if seen[key] {
			continue
		}
		seen[key] = true
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{gv.Group},
			Resources: []string{plural},
			Verbs:     defaultVerbs,
		})
	}
	return rules
}

// customResourceDeletionProtectionRBACRules returns gateway RBAC rules granting
// GET on every custom child resource that the deletion-protection webhook
// intercepts. Mirrors customRBACRulesForCRD but uses get-only verbs — the
// gateway reads each object to check its protection label before allowing the DELETE.
func (k *Katalog) customResourceDeletionProtectionRBACRules() []rbacv1.PolicyRule {
	seen := make(map[string]bool)
	var rules []rbacv1.PolicyRule
	for _, crd := range k.Enabled() {
		for _, rule := range customRBACRulesForCRD(crd) {
			key := strings.Join(rule.APIGroups, ",") + "/" + strings.Join(rule.Resources, ",")
			if seen[key] {
				continue
			}
			seen[key] = true
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: rule.APIGroups,
				Resources: rule.Resources,
				Verbs:     []string{"get"},
			})
		}
	}
	return rules
}
