// pkg/katalog/builtins.go
//
// Single authoritative registry for every Kubernetes built-in resource kind
// Orkestra knows about. Adding one entry here is the only change required to:
//
//   - Resolve GVR for children and GVR lookups
//   - Generate RBAC rules (ClusterRole) for any katalog that uses the resource
//   - Detect usage in onReconcile / onCreate / onDelete template blocks
//   - Expand kind shorthands (e.g. "hpa" → "horizontalpodautoscaler")
//   - Derive the canonical PascalCase Kind name
//   - Drive readiness and deletion-protection logic
//
// Keys are lowercase singular Kind names (e.g. "deployment", "namespace").
package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BuiltInKind holds the fully-qualified API metadata for a Kubernetes
// built-in resource kind, plus Orkestra-specific readiness and usage metadata.
type BuiltInKind struct {
	// ── Kubernetes API identity ───────────────────────────────────────────────

	Kind    string // PascalCase Kind name (e.g. "Deployment")
	Group   string // API group; empty string for core
	Version string // API version (e.g. "v1", "v2")
	Plural  string // plural resource name (e.g. "deployments")

	Namespaced bool   // true if resource is namespaced
	APIPath    string // "/api" for core, "/apis" otherwise

	// ── Discovery ─────────────────────────────────────────────────────────────

	// Shorthands are case-insensitive aliases resolved by LookupBuiltIn.
	// e.g. "hpa" → "horizontalpodautoscaler"
	Shorthands []string

	// ── Usage detection ───────────────────────────────────────────────────────

	// Detect reports whether a CRD's operatorBox uses this resource in any
	// hook template block (onCreate / onReconcile / onDelete).
	// nil for resources that cannot appear in hook templates (e.g. Node, Event).
	Detect func(crd orktypes.CRDEntry) bool

	// ── Orkestra readiness metadata ───────────────────────────────────────────

	Statusless             bool // No meaningful status; treat as ready on existence
	SkipStatusSubresource  bool // No /status subresource; never PATCH status
	SkipObservedGeneration bool // Has status but no observedGeneration; skip generation check
	IsChild                bool // Orkestra may create this as a child resource
	OrkestraInternal       bool // Part of Orkestra's own control-plane installation
}

// detectAny returns true if any hook template block in the CRD uses the
// resource selected by sel.
func detectAny[T any](crd orktypes.CRDEntry, sel func(*orktypes.HookTemplates) []T) bool {
	rc := crd.OperatorBox
	return orktypes.UsesTemplates(rc.OnCreate, sel) ||
		orktypes.UsesTemplates(rc.OnReconcile, sel) ||
		orktypes.UsesTemplates(rc.OnDelete, sel)
}

// builtInRegistry is the single source of truth for all Kubernetes built-in
// resource kinds. Keys are lowercase singular Kind names.
var builtInRegistry = map[string]BuiltInKind{

	// ── Core group (group: "", apiVersion: v1) ────────────────────────────────

	"pod": {
		Kind: "Pod", Group: "", Version: "v1", Plural: "pods",
		Namespaced: true, APIPath: "/api",
		SkipObservedGeneration: true,
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PodTemplateSource { return t.Pods })
		},
	},

	"service": {
		Kind: "Service", Group: "", Version: "v1", Plural: "services",
		Namespaced: true, APIPath: "/api",
		SkipObservedGeneration: true, IsChild: true, OrkestraInternal: true,
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.ServiceTemplateSource { return t.Services })
		},
	},

	"configmap": {
		Kind: "ConfigMap", Group: "", Version: "v1", Plural: "configmaps",
		Namespaced: true, APIPath: "/api",
		Statusless: true, SkipStatusSubresource: true, IsChild: true, OrkestraInternal: true,
		Shorthands: []string{"cm"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.ConfigMapTemplateSource { return t.ConfigMaps })
		},
	},

	"secret": {
		Kind: "Secret", Group: "", Version: "v1", Plural: "secrets",
		Namespaced: true, APIPath: "/api",
		Statusless: true, SkipStatusSubresource: true, IsChild: true,
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.SecretTemplateSource { return t.Secrets })
		},
	},

	"namespace": {
		Kind: "Namespace", Group: "", Version: "v1", Plural: "namespaces",
		Namespaced: false, APIPath: "/api",
		SkipObservedGeneration: true, OrkestraInternal: true, IsChild: true,
		Shorthands: []string{"ns"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.NamespaceTemplateSource { return t.Namespaces })
		},
	},

	"serviceaccount": {
		Kind: "ServiceAccount", Group: "", Version: "v1", Plural: "serviceaccounts",
		Namespaced: true, APIPath: "/api",
		Statusless: true, SkipStatusSubresource: true, IsChild: true, OrkestraInternal: true,
		Shorthands: []string{"sa"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.ServiceAccountTemplateSource { return t.ServiceAccounts })
		},
	},

	"persistentvolumeclaim": {
		Kind: "PersistentVolumeClaim", Group: "", Version: "v1", Plural: "persistentvolumeclaims",
		Namespaced: true, APIPath: "/api",
		SkipObservedGeneration: true,
		Shorthands:             []string{"pvc"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PVCTemplateSource { return t.PersistentVolumeClaims })
		},
	},

	"persistentvolume": {
		Kind: "PersistentVolume", Group: "", Version: "v1", Plural: "persistentvolumes",
		Namespaced: false, APIPath: "/api",
		SkipObservedGeneration: true,
		Shorthands:             []string{"pv"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PVTemplateSource { return t.PersistentVolumes })
		},
	},

	"event": {
		Kind: "Event", Group: "", Version: "v1", Plural: "events",
		Namespaced: true, APIPath: "/api",
		Statusless: true, SkipStatusSubresource: true,
	},

	"node": {
		Kind: "Node", Group: "", Version: "v1", Plural: "nodes",
		Namespaced: false, APIPath: "/api",
		SkipObservedGeneration: true,
	},

	"resourcequota": {
		Kind: "ResourceQuota", Group: "", Version: "v1", Plural: "resourcequotas",
		Namespaced: true, APIPath: "/api",
		SkipObservedGeneration: true,
	},

	"limitrange": {
		Kind: "LimitRange", Group: "", Version: "v1", Plural: "limitranges",
		Namespaced: true, APIPath: "/api",
		SkipObservedGeneration: true,
	},

	"componentstatus": {
		Kind: "ComponentStatus", Group: "", Version: "v1", Plural: "componentstatuses",
		Namespaced: false, APIPath: "/api",
		SkipStatusSubresource: true,
	},

	"podtemplate": {
		Kind: "PodTemplate", Group: "", Version: "v1", Plural: "podtemplates",
		Namespaced: true, APIPath: "/api",
		Statusless: true, SkipStatusSubresource: true,
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PlaceholderSource { return t.PodTemplates })
		},
	},

	// ── apps/v1 ───────────────────────────────────────────────────────────────

	"deployment": {
		Kind: "Deployment", Group: "apps", Version: "v1", Plural: "deployments",
		Namespaced: true, APIPath: "/apis",
		IsChild: true, OrkestraInternal: true,
		Shorthands: []string{"deploy", "dep"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.DeploymentTemplateSource { return t.Deployments })
		},
	},

	"statefulset": {
		Kind: "StatefulSet", Group: "apps", Version: "v1", Plural: "statefulsets",
		Namespaced: true, APIPath: "/apis",
		Shorthands: []string{"sts"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.StatefulSetTemplateSource { return t.StatefulSets })
		},
	},

	"daemonset": {
		Kind: "DaemonSet", Group: "apps", Version: "v1", Plural: "daemonsets",
		Namespaced: true, APIPath: "/apis",
		Shorthands: []string{"ds"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PlaceholderSource { return t.DaemonSets })
		},
	},

	"replicaset": {
		Kind: "ReplicaSet", Group: "apps", Version: "v1", Plural: "replicasets",
		Namespaced: true, APIPath: "/apis",
		Shorthands: []string{"rs"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.ReplicaSetTemplateSource { return t.ReplicaSets })
		},
	},

	// ── batch/v1 ──────────────────────────────────────────────────────────────

	"job": {
		Kind: "Job", Group: "batch", Version: "v1", Plural: "jobs",
		Namespaced: true, APIPath: "/apis",
		SkipStatusSubresource: true, IsChild: true,
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.JobTemplateSource { return t.Jobs })
		},
	},

	"cronjob": {
		Kind: "CronJob", Group: "batch", Version: "v1", Plural: "cronjobs",
		Namespaced: true, APIPath: "/apis",
		SkipStatusSubresource: true, IsChild: true,
		Shorthands: []string{"cj"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.CronJobTemplateSource { return t.CronJobs })
		},
	},

	// ── networking.k8s.io/v1 ─────────────────────────────────────────────────

	"ingress": {
		Kind: "Ingress", Group: "networking.k8s.io", Version: "v1", Plural: "ingresses",
		Namespaced: true, APIPath: "/apis",
		OrkestraInternal: true,
		Shorthands:       []string{"ing"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.IngressTemplateSource { return t.Ingresses })
		},
	},

	"networkpolicy": {
		Kind: "NetworkPolicy", Group: "networking.k8s.io", Version: "v1", Plural: "networkpolicies",
		Namespaced: true, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
		Shorthands: []string{"np"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PlaceholderSource { return t.NetworkPolicies })
		},
	},

	"ingressclass": {
		Kind: "IngressClass", Group: "networking.k8s.io", Version: "v1", Plural: "ingressclasses",
		Namespaced: false, APIPath: "/apis",
	},

	// ── autoscaling/v2 ────────────────────────────────────────────────────────

	"horizontalpodautoscaler": {
		Kind: "HorizontalPodAutoscaler", Group: "autoscaling", Version: "v2", Plural: "horizontalpodautoscalers",
		Namespaced: true, APIPath: "/apis",
		OrkestraInternal: true,
		Shorthands:       []string{"hpa"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.HPATemplateSource { return t.HorizontalPodAutoscalers })
		},
	},

	// ── rbac.authorization.k8s.io/v1 ─────────────────────────────────────────

	"role": {
		Kind: "Role", Group: "rbac.authorization.k8s.io", Version: "v1", Plural: "roles",
		Namespaced: true, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.RoleTemplateSource { return t.Roles })
		},
	},

	"rolebinding": {
		Kind: "RoleBinding", Group: "rbac.authorization.k8s.io", Version: "v1", Plural: "rolebindings",
		Namespaced: true, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
		Shorthands: []string{"rb"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.RoleBindingTemplateSource { return t.RoleBindings })
		},
	},

	"clusterrole": {
		Kind: "ClusterRole", Group: "rbac.authorization.k8s.io", Version: "v1", Plural: "clusterroles",
		Namespaced: false, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
		Shorthands: []string{"cr"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PlaceholderSource { return t.ClusterRoles })
		},
	},

	"clusterrolebinding": {
		Kind: "ClusterRoleBinding", Group: "rbac.authorization.k8s.io", Version: "v1", Plural: "clusterrolebindings",
		Namespaced: false, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
		Shorthands: []string{"crb"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PlaceholderSource { return t.ClusterRoleBindings })
		},
	},

	// ── policy/v1 ─────────────────────────────────────────────────────────────

	"poddisruptionbudget": {
		Kind: "PodDisruptionBudget", Group: "policy", Version: "v1", Plural: "poddisruptionbudgets",
		Namespaced: true, APIPath: "/apis",
		SkipObservedGeneration: true, OrkestraInternal: true,
		Shorthands: []string{"pdb"},
		Detect: func(crd orktypes.CRDEntry) bool {
			return detectAny(crd, func(t *orktypes.HookTemplates) []orktypes.PDBTemplateSource { return t.PodDisruptionBudgets })
		},
	},

	// ── storage.k8s.io/v1 ────────────────────────────────────────────────────

	"storageclass": {
		Kind: "StorageClass", Group: "storage.k8s.io", Version: "v1", Plural: "storageclasses",
		Namespaced: false, APIPath: "/apis",
		Shorthands: []string{"sc"},
	},

	"volumeattachment": {
		Kind: "VolumeAttachment", Group: "storage.k8s.io", Version: "v1", Plural: "volumeattachments",
		Namespaced: false, APIPath: "/apis",
		SkipObservedGeneration: true,
	},

	// ── apiextensions.k8s.io/v1 ──────────────────────────────────────────────

	"customresourcedefinition": {
		Kind: "CustomResourceDefinition", Group: "apiextensions.k8s.io", Version: "v1", Plural: "customresourcedefinitions",
		Namespaced: false, APIPath: "/apis",
		SkipObservedGeneration: true, OrkestraInternal: true,
		Shorthands: []string{"crd"},
	},

	// ── apiregistration.k8s.io/v1 ────────────────────────────────────────────

	"apiservice": {
		Kind: "APIService", Group: "apiregistration.k8s.io", Version: "v1", Plural: "apiservices",
		Namespaced: false, APIPath: "/apis",
		SkipObservedGeneration: true,
	},

	// ── admissionregistration.k8s.io/v1 ──────────────────────────────────────

	"mutatingwebhookconfiguration": {
		Kind: "MutatingWebhookConfiguration", Group: "admissionregistration.k8s.io", Version: "v1", Plural: "mutatingwebhookconfigurations",
		Namespaced: false, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
	},

	"validatingwebhookconfiguration": {
		Kind: "ValidatingWebhookConfiguration", Group: "admissionregistration.k8s.io", Version: "v1", Plural: "validatingwebhookconfigurations",
		Namespaced: false, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true, OrkestraInternal: true,
	},

	// ── scheduling.k8s.io/v1 ─────────────────────────────────────────────────

	"priorityclass": {
		Kind: "PriorityClass", Group: "scheduling.k8s.io", Version: "v1", Plural: "priorityclasses",
		Namespaced: false, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true,
	},

	// ── events.k8s.io/v1 ─────────────────────────────────────────────────────
	// Internal key avoids clashing with core/v1 Event.

	"event_events": {
		Kind: "Event", Group: "events.k8s.io", Version: "v1", Plural: "events",
		Namespaced: true, APIPath: "/apis",
		Statusless: true, SkipStatusSubresource: true,
	},

	// ── discovery.k8s.io/v1 ──────────────────────────────────────────────────

	"endpointslice": {
		Kind: "EndpointSlice", Group: "discovery.k8s.io", Version: "v1", Plural: "endpointslices",
		Namespaced: true, APIPath: "/apis",
		SkipObservedGeneration: true,
		Shorthands:             []string{"ep"},
	},

	// ── coordination.k8s.io/v1 ───────────────────────────────────────────────

	"lease": {
		Kind: "Lease", Group: "coordination.k8s.io", Version: "v1", Plural: "leases",
		Namespaced: true, APIPath: "/apis",
		SkipObservedGeneration: true, OrkestraInternal: true,
	},
}

// shorthandIndex maps each shorthand alias to its canonical registry key.
// Built once at init from the Shorthands field of each registry entry.
var shorthandIndex map[string]string

func init() {
	shorthandIndex = make(map[string]string, 32)
	for key, b := range builtInRegistry {
		for _, sh := range b.Shorthands {
			shorthandIndex[sh] = key
		}
	}
}

// ── Lookup API ────────────────────────────────────────────────────────────────

// EnrichmentResult holds the result of a built-in lookup.
type EnrichmentResult struct {
	Found        bool
	Kind         string
	BuiltIn      BuiltInKind
	DisplayGroup string
}

// LookupBuiltIn looks up a Kind in the built-in registry.
// Case-insensitive. Expands shorthands (e.g. "hpa" → "horizontalpodautoscaler").
func LookupBuiltIn(kind string) EnrichmentResult {
	key := strings.ToLower(strings.TrimSpace(kind))
	if key == "" {
		return EnrichmentResult{}
	}
	if expanded, ok := shorthandIndex[key]; ok {
		key = expanded
	}
	b, ok := builtInRegistry[key]
	if !ok {
		return EnrichmentResult{}
	}
	displayGroup := b.Group
	if displayGroup == "" {
		displayGroup = "core"
	}
	return EnrichmentResult{
		Found:        true,
		Kind:         b.Kind,
		BuiltIn:      b,
		DisplayGroup: displayGroup,
	}
}

// GVRForBuiltIn returns the GroupVersionResource for a built-in kind.
func GVRForBuiltIn(kind string) (schema.GroupVersionResource, bool) {
	res := LookupBuiltIn(kind)
	if !res.Found {
		return schema.GroupVersionResource{}, false
	}
	b := res.BuiltIn
	return schema.GroupVersionResource{Group: b.Group, Version: b.Version, Resource: b.Plural}, true
}

// BuiltInMeta returns metadata for a built-in kind. Zero value when unknown.
func BuiltInMeta(kind string) BuiltInKind {
	res := LookupBuiltIn(kind)
	if !res.Found {
		return BuiltInKind{}
	}
	return res.BuiltIn
}

// IsBuiltIn reports whether kind is a known Kubernetes built-in (case-insensitive).
func IsBuiltIn(kind string) bool {
	return LookupBuiltIn(kind).Found
}

// AllBuiltInKinds returns all canonical Kind names, sorted alphabetically.
func AllBuiltInKinds() []string {
	kinds := make([]string, 0, len(builtInRegistry))
	for k, b := range builtInRegistry {
		if strings.Contains(k, "_") {
			continue // skip internal alias keys like "event_events"
		}
		kinds = append(kinds, b.Kind)
	}
	for i := 0; i < len(kinds); i++ {
		for j := i + 1; j < len(kinds); j++ {
			if kinds[i] > kinds[j] {
				kinds[i], kinds[j] = kinds[j], kinds[i]
			}
		}
	}
	return kinds
}

// ── Readiness / deletion-protection queries ───────────────────────────────────

func SkipObservedGenerationGVKs() []string {
	return gvksByFlag(func(b BuiltInKind) bool { return b.SkipObservedGeneration })
}
func SkipStatusSubresourceGVKs() []string {
	return gvksByFlag(func(b BuiltInKind) bool { return b.SkipStatusSubresource })
}
func StatuslessGVKs() []string { return gvksByFlag(func(b BuiltInKind) bool { return b.Statusless }) }

func gvksByFlag(predicate func(BuiltInKind) bool) []string {
	var out []string
	for key, b := range builtInRegistry {
		if !predicate(b) {
			continue
		}
		kind := b.Kind
		if kind == "" {
			kind = strings.ToUpper(key[:1]) + key[1:]
		}
		if b.Group == "" {
			out = append(out, b.Version+"/"+kind)
		} else {
			out = append(out, b.Group+"/"+b.Version+"/"+kind)
		}
	}
	return out
}

// OrkestraInternalGVRs returns GVRs for Orkestra's own control-plane resources.
// Used by the deletion-protection webhook.
func OrkestraInternalGVRs() []GVREntry {
	var out []GVREntry
	for _, b := range builtInRegistry {
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
