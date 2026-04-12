// pkg/katalog/builtins.go
package katalog

import "strings"

// BuiltInKind holds the fully-qualified API metadata for a Kubernetes
// built-in resource kind. This information is stable — it does not change
// between Kubernetes minor versions for GA resources.
//
// When a Katalog entry declares only apiTypes.kind and it matches a built-in,
// Orkestra enriches the entry with the correct group, version, plural, and
// scope automatically. The user writes:
//
//	apiTypes:
//	  kind: Pod
//
// Orkestra resolves it to group:"", version:"v1", plural:"pods", namespaced:true.
type BuiltInKind struct {
	// Group — API group. Empty string for core group resources (Pod, Service, etc.)
	Group string

	// Version — preferred stable version for this resource.
	Version string

	// Plural — plural resource name used in API paths.
	Plural string

	// Namespaced — true when this resource is namespace-scoped.
	Namespaced bool

	// APIPath — always "/apis" except for core group which uses "/api"
	APIPath string

	// Orkestra readiness metadata
    Statusless             bool // No meaningful status; ready on creation
    SkipStatusSubresource  bool // No /status subresource; do not PATCH status
    SkipObservedGeneration bool // Has status but no observedGeneration
    IsChild                bool // Orkestra may create this as a child resource
}

// builtInRegistry is the canonical map of Kubernetes built-in resource kinds
// to their fully-qualified API metadata.
//
// Only GA (stable) versions are included. Alpha and beta versions are not
// registered here — they are not stable across Kubernetes versions and
// users who need them should declare the full apiTypes block.
//
// Lookup is case-insensitive — "pod", "Pod", and "POD" all resolve correctly.

// builtInRegistry is the canonical map of Kubernetes built-in resource kinds
// to their fully-qualified API metadata and readiness semantics.
var builtInRegistry = map[string]BuiltInKind{

    // ────────────────────────────────
    // Core v1
    // ────────────────────────────────

    "pod": {
        Group:      "",
        Version:    "v1",
        Plural:     "pods",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,  // no observedGeneration
        IsChild:                true,
    },

    "service": {
        Group:      "",
        Version:    "v1",
        Plural:     "services",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             true,  // no Ready condition
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                true,
    },

    "configmap": {
        Group:      "",
        Version:    "v1",
        Plural:     "configmaps",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                true,
    },

    "secret": {
        Group:      "",
        Version:    "v1",
        Plural:     "secrets",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                true,
    },

    "serviceaccount": {
        Group:      "",
        Version:    "v1",
        Plural:     "serviceaccounts",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                true,
    },

    "namespace": {
        Group:      "",
        Version:    "v1",
        Plural:     "namespaces",
        Namespaced: false,
        APIPath:    "/api",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                false,
    },

    "event": {
        Group:      "",
        Version:    "v1",
        Plural:     "events",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "podtemplate": {
        Group:      "",
        Version:    "v1",
        Plural:     "podtemplates",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "componentstatus": {
        Group:      "",
        Version:    "v1",
        Plural:     "componentstatuses",
        Namespaced: false,
        APIPath:    "/api",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "resourcequota": {
        Group:      "",
        Version:    "v1",
        Plural:     "resourcequotas",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                false,
    },

    "limitrange": {
        Group:      "",
        Version:    "v1",
        Plural:     "limitranges",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                false,
    },

    "persistentvolume": {
        Group:      "",
        Version:    "v1",
        Plural:     "persistentvolumes",
        Namespaced: false,
        APIPath:    "/api",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                false,
    },

    "persistentvolumeclaim": {
        Group:      "",
        Version:    "v1",
        Plural:     "persistentvolumeclaims",
        Namespaced: true,
        APIPath:    "/api",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                false,
    },

    // ────────────────────────────────
    // apps/v1
    // ────────────────────────────────

    "deployment": {
        Group:      "apps",
        Version:    "v1",
        Plural:     "deployments",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: false,
        IsChild:                true,
    },

    "statefulset": {
        Group:      "apps",
        Version:    "v1",
        Plural:     "statefulsets",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "daemonset": {
        Group:      "apps",
        Version:    "v1",
        Plural:     "daemonsets",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "replicaset": {
        Group:      "apps",
        Version:    "v1",
        Plural:     "replicasets",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    // ────────────────────────────────
    // batch/v1
    // ────────────────────────────────

    "job": {
        Group:      "batch",
        Version:    "v1",
        Plural:     "jobs",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  true,  // no /status
        SkipObservedGeneration: false,
        IsChild:                true,
    },

    "cronjob": {
        Group:      "batch",
        Version:    "v1",
        Plural:     "cronjobs",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                true,
    },

    // ────────────────────────────────
    // networking.k8s.io/v1
    // ────────────────────────────────

    "networkpolicy": {
        Group:      "networking.k8s.io",
        Version:    "v1",
        Plural:     "networkpolicies",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    // ────────────────────────────────
    // rbac.authorization.k8s.io/v1
    // ────────────────────────────────

    "role": {
        Group:      "rbac.authorization.k8s.io",
        Version:    "v1",
        Plural:     "roles",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "rolebinding": {
        Group:      "rbac.authorization.k8s.io",
        Version:    "v1",
        Plural:     "rolebindings",
        Namespaced: true,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "clusterrole": {
        Group:      "rbac.authorization.k8s.io",
        Version:    "v1",
        Plural:     "clusterroles",
        Namespaced: false,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "clusterrolebinding": {
        Group:      "rbac.authorization.k8s.io",
        Version:    "v1",
        Plural:     "clusterrolebindings",
        Namespaced: false,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    // ────────────────────────────────
    // admissionregistration.k8s.io/v1
    // ────────────────────────────────

    "mutatingwebhookconfiguration": {
        Group:      "admissionregistration.k8s.io",
        Version:    "v1",
        Plural:     "mutatingwebhookconfigurations",
        Namespaced: false,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    "validatingwebhookconfiguration": {
        Group:      "admissionregistration.k8s.io",
        Version:    "v1",
        Plural:     "validatingwebhookconfigurations",
        Namespaced: false,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    // ────────────────────────────────
    // scheduling.k8s.io/v1
    // ────────────────────────────────

    "priorityclass": {
        Group:      "scheduling.k8s.io",
        Version:    "v1",
        Plural:     "priorityclasses",
        Namespaced: false,
        APIPath:    "/apis",

        Statusless:             true,
        SkipStatusSubresource:  true,
        SkipObservedGeneration: false,
        IsChild:                false,
    },

    // ────────────────────────────────
    // apiextensions.k8s.io/v1
    // ────────────────────────────────

    "customresourcedefinition": {
        Group:      "apiextensions.k8s.io",
        Version:    "v1",
        Plural:     "customresourcedefinitions",
        Namespaced: false,
        APIPath:    "/apis",

        Statusless:             false,
        SkipStatusSubresource:  false,
        SkipObservedGeneration: true,
        IsChild:                false,
    },
}

// BuiltInMeta returns metadata for a built-in kind.
func BuiltInMeta(kind string) BuiltInKind {
    key := strings.ToLower(strings.TrimSpace(kind))
    return builtInRegistry[key]
}

// EnrichmentResult holds the result of a built-in lookup.
type EnrichmentResult struct {
	// Found — true when the kind matched a built-in
	Found bool

	// Kind — the canonical Kind name (correct casing from the registry)
	Kind string

	// BuiltIn — the resolved API metadata
	BuiltIn BuiltInKind

	// DisplayGroup — human-readable group string for logs and output.
	// Shows "core" for the core group (empty string) to avoid confusion.
	DisplayGroup string
}

// LookupBuiltIn looks up a Kind in the built-in registry.
// Lookup is case-insensitive — "Pod", "pod", and "POD" all resolve.
//
// Returns an EnrichmentResult. Check result.Found before using the values.
func LookupBuiltIn(kind string) EnrichmentResult {
	key := strings.ToLower(strings.TrimSpace(kind))
	if key == "" {
		return EnrichmentResult{}
	}

	b, ok := builtInRegistry[key]
	if !ok {
		return EnrichmentResult{}
	}

	displayGroup := b.Group
	if displayGroup == "" {
		displayGroup = "core"
	}

	// Canonical Kind name — first letter uppercase, rest as registered
	canonicalKind := canonicalKindName(key)

	return EnrichmentResult{
		Found:        true,
		Kind:         canonicalKind,
		BuiltIn:      b,
		DisplayGroup: displayGroup,
	}
}

// IsBuiltIn reports whether a kind string refers to a known Kubernetes built-in.
// Case-insensitive. Does not require the fully-qualified group/version.
func IsBuiltIn(kind string) bool {
	return LookupBuiltIn(kind).Found
}

// AllBuiltInKinds returns the canonical Kind names of all registered built-ins.
// Sorted alphabetically. Used by `ork validate` to suggest alternatives.
func AllBuiltInKinds() []string {
	kinds := make([]string, 0, len(builtInRegistry))
	for k := range builtInRegistry {
		if !strings.Contains(k, "_") { // skip internal aliases
			kinds = append(kinds, canonicalKindName(k))
		}
	}
	// sort manually without importing sort — keep package lean
	for i := 0; i < len(kinds); i++ {
		for j := i + 1; j < len(kinds); j++ {
			if kinds[i] > kinds[j] {
				kinds[i], kinds[j] = kinds[j], kinds[i]
			}
		}
	}
	return kinds
}

// canonicalKindName returns the conventional PascalCase Kind name from a
// lowercase registry key. e.g. "horizontalpodautoscaler" → "HorizontalPodAutoscaler"
//
// These mappings are hardcoded because the convention is not programmatically
// derivable — "horizontalpodautoscaler" → "HorizontalPodAutoscaler" requires
// knowing where the word boundaries are.
var canonicalKindNames = map[string]string{
	"pod":                      "Pod",
	"service":                  "Service",
	"configmap":                "ConfigMap",
	"secret":                   "Secret",
	"namespace":                "Namespace",
	"serviceaccount":           "ServiceAccount",
	"persistentvolumeclaim":    "PersistentVolumeClaim",
	"persistentvolume":         "PersistentVolume",
	"endpointSlice":            "EndpointSlice",
	"event":                    "Event",
	"node":                     "Node",
	"resourcequota":            "ResourceQuota",
	"limitrange":               "LimitRange",
	"deployment":               "Deployment",
	"statefulset":              "StatefulSet",
	"daemonset":                "DaemonSet",
	"replicaset":               "ReplicaSet",
	"job":                      "Job",
	"cronjob":                  "CronJob",
	"ingress":                  "Ingress",
	"networkpolicy":            "NetworkPolicy",
	"ingressclass":             "IngressClass",
	"horizontalpodautoscaler":  "HorizontalPodAutoscaler",
	"clusterrole":              "ClusterRole",
	"clusterrolebinding":       "ClusterRoleBinding",
	"role":                     "Role",
	"rolebinding":              "RoleBinding",
	"storageclass":             "StorageClass",
	"poddisruptionbudget":      "PodDisruptionBudget",
	"customresourcedefinition": "CustomResourceDefinition",
}

func canonicalKindName(key string) string {
	if name, ok := canonicalKindNames[key]; ok {
		return name
	}
	// Fallback: capitalise first letter only
	if len(key) == 0 {
		return key
	}
	return strings.ToUpper(key[:1]) + key[1:]
}
