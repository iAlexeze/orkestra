// pkg/katalog/builtins.go
package katalog

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BuiltInKind holds the fully-qualified API metadata for a Kubernetes
// built-in resource kind, plus Orkestra-specific readiness metadata.
//
// Lookup is always by Kind (case-insensitive) via LookupBuiltIn / BuiltInMeta.
type BuiltInKind struct {
	// Kubernetes API metadata
	Group      string // API group; empty for core
	Version    string // API version
	Plural     string // plural resource name
	Namespaced bool   // true if namespaced
	APIPath    string // "/api" for core, "/apis" otherwise

	// Orkestra readiness metadata
	Statusless             bool // No meaningful status; treat as ready on existence
	SkipStatusSubresource  bool // No /status subresource; never PATCH status
	SkipObservedGeneration bool // Has status but no observedGeneration; skip generation-based checks
	IsChild                bool // Orkestra may create this as a child resource
	OrkestraInternal       bool // To protect Orkestra’s own control‑plane resources when security.deletionProtection=true
}

// builtInRegistry is the canonical map of Kubernetes built-in resource kinds
// to their fully-qualified API metadata and Orkestra readiness hints.
//
// Keys are lowercase Kind names (e.g. "pod", "deployment").
var builtInRegistry = map[string]BuiltInKind{
	// ── Core group (group: "", apiVersion: v1) ────────────────────────────────
	"pod": {
		Group:      "",
		Version:    "v1",
		Plural:     "pods",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/Pod
	},
	"service": {
		Group:      "",
		Version:    "v1",
		Plural:     "services",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/Service
		IsChild:                true,
		OrkestraInternal:       true,
	},
	"configmap": {
		Group:      "",
		Version:    "v1",
		Plural:     "configmaps",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		Statusless:            true, // v1/ConfigMap
		SkipStatusSubresource: true,
		IsChild:               true,
		OrkestraInternal:      true,
	},
	"secret": {
		Group:      "",
		Version:    "v1",
		Plural:     "secrets",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		Statusless:            true, // v1/Secret
		SkipStatusSubresource: true,
		IsChild:               true,
	},
	"namespace": {
		Group:      "",
		Version:    "v1",
		Plural:     "namespaces",
		Namespaced: false,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/Namespace
		OrkestraInternal:       true,
		IsChild:                true,
	},
	"serviceaccount": {
		Group:      "",
		Version:    "v1",
		Plural:     "serviceaccounts",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		Statusless:            true, // v1/ServiceAccount
		SkipStatusSubresource: true,
		IsChild:               true,
		OrkestraInternal:      true,
	},
	"persistentvolumeclaim": {
		Group:      "",
		Version:    "v1",
		Plural:     "persistentvolumeclaims",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/PersistentVolumeClaim
	},
	"persistentvolume": {
		Group:      "",
		Version:    "v1",
		Plural:     "persistentvolumes",
		Namespaced: false,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/PersistentVolume
	},
	"event": {
		Group:      "",
		Version:    "v1",
		Plural:     "events",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		Statusless:            true, // v1/Event
		SkipStatusSubresource: true,
	},
	"node": {
		Group:      "",
		Version:    "v1",
		Plural:     "nodes",
		Namespaced: false,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/Node
	},
	"resourcequota": {
		Group:      "",
		Version:    "v1",
		Plural:     "resourcequotas",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/ResourceQuota
	},
	"limitrange": {
		Group:      "",
		Version:    "v1",
		Plural:     "limitranges",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		SkipObservedGeneration: true, // v1/LimitRange
	},
	"componentstatus": {
		Group:      "",
		Version:    "v1",
		Plural:     "componentstatuses",
		Namespaced: false,
		APIPath:    "/api",
		// Orkestra
		SkipStatusSubresource: true, // v1/ComponentStatus
	},
	"podtemplate": {
		Group:      "",
		Version:    "v1",
		Plural:     "podtemplates",
		Namespaced: true,
		APIPath:    "/api",
		// Orkestra
		Statusless:            true, // v1/PodTemplate
		SkipStatusSubresource: true,
	},

	// ── apps/v1 ───────────────────────────────────────────────────────────────
	"deployment": {
		Group:      "apps",
		Version:    "v1",
		Plural:     "deployments",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		IsChild:          true,
		OrkestraInternal: true,
	},
	"statefulset": {
		Group:      "apps",
		Version:    "v1",
		Plural:     "statefulsets",
		Namespaced: true,
		APIPath:    "/apis",
	},
	"daemonset": {
		Group:      "apps",
		Version:    "v1",
		Plural:     "daemonsets",
		Namespaced: true,
		APIPath:    "/apis",
	},
	"replicaset": {
		Group:      "apps",
		Version:    "v1",
		Plural:     "replicasets",
		Namespaced: true,
		APIPath:    "/apis",
	},

	// ── batch/v1 ──────────────────────────────────────────────────────────────
	"job": {
		Group:      "batch",
		Version:    "v1",
		Plural:     "jobs",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		SkipStatusSubresource: true, // batch/v1/Job
		IsChild:               true,
	},

	"cronjob": {
		Group:      "batch",
		Version:    "v1",
		Plural:     "cronjobs",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		SkipStatusSubresource: true, // batch/v1/CronJob
		IsChild:               true,
	},

	// ── networking.k8s.io/v1 ─────────────────────────────────────────────────
	"ingress": {
		Group:            "networking.k8s.io",
		Version:          "v1",
		Plural:           "ingresses",
		Namespaced:       true,
		APIPath:          "/apis",
		OrkestraInternal: true,
	},
	"networkpolicy": {
		Group:      "networking.k8s.io",
		Version:    "v1",
		Plural:     "networkpolicies",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // networking.k8s.io/v1/NetworkPolicy
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},
	"ingressclass": {
		Group:      "networking.k8s.io",
		Version:    "v1",
		Plural:     "ingressclasses",
		Namespaced: false,
		APIPath:    "/apis",
	},

	// ── autoscaling/v2 ────────────────────────────────────────────────────────
	"horizontalpodautoscaler": {
		Group:            "autoscaling",
		Version:          "v2",
		Plural:           "horizontalpodautoscalers",
		Namespaced:       true,
		APIPath:          "/apis",
		OrkestraInternal: true,
	},

	// ── rbac.authorization.k8s.io/v1 ─────────────────────────────────────────
	"clusterrole": {
		Group:      "rbac.authorization.k8s.io",
		Version:    "v1",
		Plural:     "clusterroles",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // rbac.authorization.k8s.io/v1/ClusterRole
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},
	"clusterrolebinding": {
		Group:      "rbac.authorization.k8s.io",
		Version:    "v1",
		Plural:     "clusterrolebindings",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // rbac.authorization.k8s.io/v1/ClusterRoleBinding
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},
	"role": {
		Group:      "rbac.authorization.k8s.io",
		Version:    "v1",
		Plural:     "roles",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // rbac.authorization.k8s.io/v1/Role
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},

	"rolebinding": {
		Group:      "rbac.authorization.k8s.io",
		Version:    "v1",
		Plural:     "rolebindings",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // rbac.authorization.k8s.io/v1/RoleBinding
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},

	// ── storage.k8s.io/v1 ─────────────────────────────────────────────────────
	"storageclass": {
		Group:      "storage.k8s.io",
		Version:    "v1",
		Plural:     "storageclasses",
		Namespaced: false,
		APIPath:    "/apis",
	},
	"volumeattachment": {
		Group:      "storage.k8s.io",
		Version:    "v1",
		Plural:     "volumeattachments",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		SkipObservedGeneration: true, // storage.k8s.io/v1/VolumeAttachment
	},

	// ── policy/v1 ─────────────────────────────────────────────────────────────
	"poddisruptionbudget": {
		Group:      "policy",
		Version:    "v1",
		Plural:     "poddisruptionbudgets",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		SkipObservedGeneration: true, // policy/v1/PodDisruptionBudget
		OrkestraInternal:       true,
	},

	// ── apiextensions.k8s.io/v1 ──────────────────────────────────────────────
	"customresourcedefinition": {
		Group:      "apiextensions.k8s.io",
		Version:    "v1",
		Plural:     "customresourcedefinitions",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		SkipObservedGeneration: true, // apiextensions.k8s.io/v1/CustomResourceDefinition
		OrkestraInternal:       true,
	},

	// ── apiregistration.k8s.io/v1 ────────────────────────────────────────────
	"apiservice": {
		Group:      "apiregistration.k8s.io",
		Version:    "v1",
		Plural:     "apiservices",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		SkipObservedGeneration: true, // apiregistration.k8s.io/v1/APIService
	},

	// ── admissionregistration.k8s.io/v1 ──────────────────────────────────────
	"mutatingwebhookconfiguration": {
		Group:      "admissionregistration.k8s.io",
		Version:    "v1",
		Plural:     "mutatingwebhookconfigurations",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // admissionregistration.k8s.io/v1/MutatingWebhookConfiguration
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},

	"validatingwebhookconfiguration": {
		Group:      "admissionregistration.k8s.io",
		Version:    "v1",
		Plural:     "validatingwebhookconfigurations",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // admissionregistration.k8s.io/v1/ValidatingWebhookConfiguration
		SkipStatusSubresource: true,
		OrkestraInternal:      true,
	},

	// ── scheduling.k8s.io/v1 ─────────────────────────────────────────────────
	"priorityclass": {
		Group:      "scheduling.k8s.io",
		Version:    "v1",
		Plural:     "priorityclasses",
		Namespaced: false,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // scheduling.k8s.io/v1/PriorityClass
		SkipStatusSubresource: true,
	},

	// ── events.k8s.io/v1 ─────────────────────────────────────────────────────
	"event_events": { // internal key to avoid clashing with core/v1 Event
		Group:      "events.k8s.io",
		Version:    "v1",
		Plural:     "events",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		Statusless:            true, // events.k8s.io/v1/Event
		SkipStatusSubresource: true,
	},

	// ── discovery.k8s.io/v1 ──────────────────────────────────────────────────
	"endpointslice": {
		Group:      "discovery.k8s.io",
		Version:    "v1",
		Plural:     "endpointslices",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		SkipObservedGeneration: true, // discovery.k8s.io/v1/EndpointSlice
	},

	// ── coordination.k8s.io/v1 ──────────────────────────────────────────────────
	"lease": {
		Group:      "v1",
		Version:    "coordination.k8s.io",
		Plural:     "leases",
		Namespaced: true,
		APIPath:    "/apis",
		// Orkestra
		SkipObservedGeneration: true,
		OrkestraInternal:       true,
	},
}

// EnrichmentResult holds the result of a built-in lookup.
type EnrichmentResult struct {
	Found        bool
	Kind         string
	BuiltIn      BuiltInKind
	DisplayGroup string
}

// kindShorthands maps common abbreviations to their canonical registry keys.
// Applied before built-in lookup so users can write e.g. "hpa" or "pdb".
var kindShorthands = map[string]string{
	"dep": "deployment",
	"sts": "statefulset",
	"ds":  "daemonset",
	"rs":  "replicaset",
	"cj":  "cronjob",
	"ing": "ingress",
	"np":  "networkpolicy",
	"hpa": "horizontalpodautoscaler",
	"pdb": "poddisruptionbudget",
	"cm":  "configmap",
	"sa":  "serviceaccount",
	"pvc": "persistentvolumeclaim",
	"pv":  "persistentvolume",
	"ns":  "namespace",
	"crd": "customresourcedefinition",
	"rb":  "rolebinding",
	"crb": "clusterrolebinding",
	"cr":  "clusterrole",
	"sc":  "storageclass",
	"ep":  "endpointslice",
}

// LookupBuiltIn looks up a Kind in the built-in registry.
// Case-insensitive. Expands common shorthands (e.g. "hpa" → "horizontalpodautoscaler").
// Returns EnrichmentResult; check .Found before use.
func LookupBuiltIn(kind string) EnrichmentResult {
	key := strings.ToLower(strings.TrimSpace(kind))
	if key == "" {
		return EnrichmentResult{}
	}

	if expanded, ok := kindShorthands[key]; ok {
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

	canonicalKind := canonicalKindName(key)

	return EnrichmentResult{
		Found:        true,
		Kind:         canonicalKind,
		BuiltIn:      b,
		DisplayGroup: displayGroup,
	}
}

// GVRForBuiltIn returns the GroupVersionResource for a built-in kind.
// Returns the zero-value GVR and false when the kind is unknown.
func GVRForBuiltIn(kind string) (schema.GroupVersionResource, bool) {
	res := LookupBuiltIn(kind)
	if !res.Found {
		return schema.GroupVersionResource{}, false
	}

	b := res.BuiltIn
	return schema.GroupVersionResource{
		Group:    b.Group,
		Version:  b.Version,
		Resource: b.Plural,
	}, true
}

// BuiltInMeta returns metadata for a built-in kind.
// Zero value is returned when the kind is unknown.
func BuiltInMeta(kind string) BuiltInKind {
	res := LookupBuiltIn(kind)
	if !res.Found {
		return BuiltInKind{}
	}
	return res.BuiltIn
}

// IsBuiltIn reports whether a kind string refers to a known Kubernetes built-in.
// Case-insensitive. Does not require the fully-qualified group/version.
func IsBuiltIn(kind string) bool {
	return LookupBuiltIn(kind).Found
}

// AllBuiltInKinds returns the canonical Kind names of all registered built-ins.
// Sorted alphabetically (simple O(n^2) sort to avoid extra imports).
func AllBuiltInKinds() []string {
	kinds := make([]string, 0, len(builtInRegistry))
	for k := range builtInRegistry {
		if strings.Contains(k, "_") {
			// skip internal alias keys like "event_events"
			continue
		}
		kinds = append(kinds, canonicalKindName(k))
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

// SkipObservedGenerationGVKs returns the GVKs that should skip generation-based readiness checks.
func SkipObservedGenerationGVKs() []string {
	var out []string
	for key, b := range builtInRegistry {
		if !b.SkipObservedGeneration {
			continue
		}
		kind := canonicalKindName(key)
		if b.Group == "" {
			out = append(out, b.Version+"/"+kind)
		} else {
			out = append(out, b.Group+"/"+b.Version+"/"+kind)
		}
	}
	return out
}

// SkipStatusSubresourceGVKs returns the GVKs that do not have a /status subresource.
func SkipStatusSubresourceGVKs() []string {
	var out []string
	for key, b := range builtInRegistry {
		if !b.SkipStatusSubresource {
			continue
		}
		kind := canonicalKindName(key)
		if b.Group == "" {
			out = append(out, b.Version+"/"+kind)
		} else {
			out = append(out, b.Group+"/"+b.Version+"/"+kind)
		}
	}
	return out
}

// StatuslessGVKs returns the GVKs that should be treated as "ready on existence".
func StatuslessGVKs() []string {
	var out []string
	for key, b := range builtInRegistry {
		if !b.Statusless {
			continue
		}
		kind := canonicalKindName(key)
		if b.Group == "" {
			out = append(out, b.Version+"/"+kind)
		} else {
			out = append(out, b.Group+"/"+b.Version+"/"+kind)
		}
	}
	return out
}

// OrkestraInternalGVRs returns the list of Kubernetes resources that belong to
// Orkestra’s own control‑plane installation (runtime Deployment, Service,
// ServiceAccount, RBAC objects, NetworkPolicy, PDB, etc.).
//
// These resources are marked in the built‑ins registry with
// BuiltInKind.OrkestraInternal = true.
//
// The deletion‑protection webhook uses this list to prevent accidental deletion
// of Orkestra’s control‑plane components. User‑created resources (including
// operator‑managed children) are *not* protected, because they are not marked
// as OrkestraInternal.
//
// This keeps the protection surface minimal, declarative, and fully aligned
// with the built‑ins registry.
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

// canonicalKindNames maps lowercase registry keys to canonical Kind names.
var canonicalKindNames = map[string]string{
	"pod":                            "Pod",
	"service":                        "Service",
	"configmap":                      "ConfigMap",
	"secret":                         "Secret",
	"namespace":                      "Namespace",
	"serviceaccount":                 "ServiceAccount",
	"persistentvolumeclaim":          "PersistentVolumeClaim",
	"persistentvolume":               "PersistentVolume",
	"endpointslice":                  "EndpointSlice",
	"event":                          "Event",
	"node":                           "Node",
	"resourcequota":                  "ResourceQuota",
	"limitrange":                     "LimitRange",
	"componentstatus":                "ComponentStatus",
	"podtemplate":                    "PodTemplate",
	"deployment":                     "Deployment",
	"statefulset":                    "StatefulSet",
	"daemonset":                      "DaemonSet",
	"replicaset":                     "ReplicaSet",
	"job":                            "Job",
	"cronjob":                        "CronJob",
	"ingress":                        "Ingress",
	"networkpolicy":                  "NetworkPolicy",
	"ingressclass":                   "IngressClass",
	"horizontalpodautoscaler":        "HorizontalPodAutoscaler",
	"clusterrole":                    "ClusterRole",
	"clusterrolebinding":             "ClusterRoleBinding",
	"role":                           "Role",
	"rolebinding":                    "RoleBinding",
	"storageclass":                   "StorageClass",
	"volumeattachment":               "VolumeAttachment",
	"poddisruptionbudget":            "PodDisruptionBudget",
	"customresourcedefinition":       "CustomResourceDefinition",
	"apiservice":                     "APIService",
	"mutatingwebhookconfiguration":   "MutatingWebhookConfiguration",
	"validatingwebhookconfiguration": "ValidatingWebhookConfiguration",
	"priorityclass":                  "PriorityClass",
	"event_events":                   "Event", // events.k8s.io/v1/Event
}

// canonicalKindName returns the conventional PascalCase Kind name from a
// lowercase registry key. Falls back to capitalising the first letter.
func canonicalKindName(key string) string {
	if name, ok := canonicalKindNames[key]; ok {
		return name
	}
	if len(key) == 0 {
		return key
	}
	return strings.ToUpper(key[:1]) + key[1:]
}
