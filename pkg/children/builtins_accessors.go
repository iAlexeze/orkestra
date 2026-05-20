// pkg/children/builtins_accessors.go
//
// Accessor functions over the builtInRegistry defined in builtins.go.
// These are the public API for querying built-in kind metadata.
// To add a new built-in kind, edit builtins.go — not this file.
package children

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// EnrichmentResult holds the result of a built-in lookup.
type EnrichmentResult struct {
	Found        bool
	Kind         string // canonical PascalCase name (e.g. "Deployment")
	BuiltIn      BuiltInKind
	DisplayGroup string // "core" for empty group, otherwise the group string
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

// LookupBuiltInByResource looks up a built-in by singular key, shorthand, or plural
// resource name. Returns (BuiltInKind, true) when found. Used by RBAC generation
// and any caller that works with resource strings rather than Kind names.
func LookupBuiltInByResource(resource string) (BuiltInKind, bool) {
	key := strings.ToLower(strings.TrimSpace(resource))
	if b, ok := builtInRegistry[key]; ok && b.Detect != nil {
		return b, true
	}
	if expanded, ok := shorthandIndex[key]; ok {
		if b, ok := builtInRegistry[expanded]; ok && b.Detect != nil {
			return b, true
		}
	}
	for _, b := range builtInRegistry {
		if b.Plural == key && b.Detect != nil {
			return b, true
		}
	}
	return BuiltInKind{}, false
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

// AllBuiltInKindDefs returns all entries from the built-in registry.
// Entries with a nil Detect field represent aliases or internal entries without
// RBAC detection logic. Callers that need only detectable entries should filter
// on Detect != nil.
func AllBuiltInKindDefs() []BuiltInKind {
	result := make([]BuiltInKind, 0, len(builtInRegistry))
	for _, b := range builtInRegistry {
		result = append(result, b)
	}
	return result
}

// enrichmentGroups maps each canonical built-in name to its full list of valid
// enrichment identifiers (name, plural, shorthands, synthetic aliases).
// Computed once at init from the immutable builtInRegistry.
var enrichmentGroups map[string][]string

// enrichmentIndex is the reverse: any valid identifier → canonical name.
// Used for O(1) lookups in enrichmentEnabled and IsValidEnrichmentTarget.
var enrichmentIndex map[string]string

func init() {
	enrichmentGroups = buildEnrichmentGroups()
	idx := make(map[string]string)
	for canonical, aliases := range enrichmentGroups {
		for _, a := range aliases {
			idx[a] = canonical
		}
	}
	enrichmentIndex = idx
}

// buildEnrichmentGroups constructs a map where each canonical built-in name
// maps to the list of all valid enrichment identifiers for that resource.
// Reads from enrichmentMeta (in builtins.go) for target/key config, and from
// builtInRegistry for plural and shorthand aliases.
func buildEnrichmentGroups() map[string][]string {
	groups := make(map[string][]string)

	for name, em := range enrichmentMeta {
		if !em.Target {
			continue
		}

		var list []string
		list = append(list, name)

		if b, ok := builtInRegistry[name]; ok {
			if b.Plural != "" {
				list = append(list, b.Plural)
			}
			for _, s := range b.Shorthands {
				list = append(list, strings.ToLower(s))
			}
		}

		for _, a := range em.EnrichKeys {
			list = append(list, a)
		}

		sort.Strings(list)
		groups[name] = list
	}

	return groups
}

// SupportedEnrichmentGroups returns all supported enrichment targets, including
// built-in Kubernetes resources and synthetic Orkestra-only targets.
func SupportedEnrichmentGroups() map[string][]string {
	return enrichmentGroups
}

// IsValidEnrichmentTarget reports whether the given name is a supported
// context-enrichment target.
func IsValidEnrichmentTarget(name string) bool {
	name = strings.ToLower(name)
	if name == "" {
		return false
	}
	_, ok := enrichmentIndex[name]
	return ok
}

// ── Readiness / deletion-protection queries ───────────────────────────────────

func SkipObservedGenerationGVKs() []string {
	return gvksByFlag(func(b BuiltInKind) bool { return b.SkipObservedGeneration })
}

func SkipStatusSubresourceGVKs() []string {
	return gvksByFlag(func(b BuiltInKind) bool { return b.SkipStatusSubresource })
}

func StatuslessGVKs() []string {
	return gvksByFlag(func(b BuiltInKind) bool { return b.Statusless })
}

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
