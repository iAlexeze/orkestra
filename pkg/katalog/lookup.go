package katalog

import (
	"sort"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// -----------------------------------------------------------------------------
// Index Building
// -----------------------------------------------------------------------------

// BuildLookupIndexes builds O(1) lookup maps indexes after CRDs are loaded.
// All indexes are stored in lowercase for case-insensitive lookups.
// After this runs, the Katalog supports:
//   - LookupByKind
//   - LookupByTarget
//   - LookupByGVKString
//   - LookupByGVRString
//
// Called last in setGroupVersionKind.
func (k *Katalog) BuildLookupIndexes() {
	k.apiVersionIndex = make(map[string]string)
	k.kindIndex = make(map[string]string)
	k.gvkIndex = make(map[string]string)
	k.gvrIndex = make(map[string]string)
	k.targetIndex = make(map[string]string)

	for name, crd := range k.enabledCRDs {
		// Store all keys in lowercase for case-insensitive lookups.
		// apiVersionIndex is keyed by apiVersion+kind concatenated, matching
		// LookupByAPIVersionAndKind's query — apiVersion alone isn't unique
		// when multiple Kinds share a group/version.
		k.apiVersionIndex[strings.ToLower(crd.APIVersion()+crd.Kind())] = name
		k.kindIndex[strings.ToLower(crd.Kind())] = name
		k.gvkIndex[strings.ToLower(crd.GVKString())] = name
		k.gvrIndex[strings.ToLower(crd.GVRString())] = name
		if crd.HasIDPTarget() {
			k.targetIndex[strings.ToLower(crd.IDPTarget())] = name
		}
	}
}

// BuildIDPEnabledCRDs returns a slice of all IDP-enabled CRDs.
// Use for iteration when the map key is not needed.
func (k *Katalog) BuildIDPEnabledCRDs() []*orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	idpEnabled := make([]*orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.IDPEnabled() {
			entry := crd
			idpEnabled = append(idpEnabled, &entry)
		}
	}
	return idpEnabled
}

// -----------------------------------------------------------------------------
// Lookup Methods (O(1) index-based)
// -----------------------------------------------------------------------------

// LookupByKind finds the CRD entry whose Kind matches the given kind string.
// O(1) lookup using the kind index. Case-insensitive. Nil-safe.
func (k *Katalog) LookupByKind(kind string) *orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(kind))
	if name, ok := k.kindIndex[key]; ok {
		if entry, ok := k.enabledCRDs[name]; ok {
			return &entry
		}
	}
	return nil
}

// LookupByName finds the CRD entry whose name (the Katalog map key) matches
// the given name. O(1) — enabledCRDs is already keyed by name. Case-insensitive. Nil-safe.
func (k *Katalog) LookupByName(name string) *orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if entry, ok := k.enabledCRDs[key]; ok {
		return &entry
	}
	return nil
}

// LookupByAPIVersionAndKind finds the CRD whose APIVersion and Kind matches the given strings.
// O(1) lookup using the apiVersionIndex.
func (k *Katalog) LookupByAPIVersionAndKind(apiVersion, kind string) *orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(apiVersion) + strings.TrimSpace(kind))
	if name, ok := k.apiVersionIndex[key]; ok {
		if entry, ok := k.enabledCRDs[name]; ok {
			return &entry
		}
	}
	return nil
}

// LookupByGVKString finds the CRD entry whose GroupVersionKind matches the given GVK string.
// O(1) lookup using the GVK index. Case-insensitive.
func (k *Katalog) LookupByGVKString(gvkString string) *orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(gvkString))
	if name, ok := k.gvkIndex[key]; ok {
		if entry, ok := k.enabledCRDs[name]; ok {
			return &entry
		}
	}
	return nil
}

// LookupByGVRString finds the CRD entry whose GroupVersionResource matches the given GVR string.
// O(1) lookup using the GVR index. Case-insensitive.
func (k *Katalog) LookupByGVRString(gvrString string) *orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(gvrString))
	if name, ok := k.gvrIndex[key]; ok {
		if entry, ok := k.enabledCRDs[name]; ok {
			return &entry
		}
	}
	return nil
}

// LookupByTarget finds the CRD entry whose resolved target matches t.
// O(1) lookup using the target index. Case-insensitive.
func (k *Katalog) LookupByTarget(target string) *orktypes.CRDEntry {
	if k == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(target))
	if name, ok := k.targetIndex[key]; ok {
		if entry, ok := k.enabledCRDs[name]; ok {
			return &entry
		}
	}
	return nil
}

// LookupByTargetOrKind attempts to find a CRD by target first, then by kind.
// Useful for handlers that accept either a target or a kind. Case-insensitive.
func (k *Katalog) LookupByTargetOrKind(identifier string) *orktypes.CRDEntry {
	identifier = strings.TrimSpace(identifier)
	if crd := k.LookupByTarget(identifier); crd != nil {
		return crd
	}
	return k.LookupByKind(identifier)
}

// MustLookupByTarget finds the CRD entry whose resolved target matches t.
// Panics if not found. Use when you know the target exists. Case-insensitive.
func (k *Katalog) MustLookupByTarget(target string) *orktypes.CRDEntry {
	crd := k.LookupByTarget(target)
	if crd == nil {
		panic("CRD not found for target: " + target)
	}
	return crd
}

// MustLookupByKind finds the CRD entry whose Kind matches the given kind string.
// Panics if not found. Use when you know the kind exists. Case-insensitive.
func (k *Katalog) MustLookupByKind(kind string) *orktypes.CRDEntry {
	crd := k.LookupByKind(kind)
	if crd == nil {
		panic("CRD not found for kind: " + kind)
	}
	return crd
}

// -----------------------------------------------------------------------------
// Existence Checks
// -----------------------------------------------------------------------------

// IsKindRegistered returns true if the kind exists in the index.
// Case-insensitive.
func (k *Katalog) IsKindRegistered(kind string) bool {
	if k == nil {
		return false
	}
	_, ok := k.kindIndex[strings.ToLower(strings.TrimSpace(kind))]
	return ok
}

// IsTargetRegistered returns true if the target exists in the index.
// Case-insensitive.
func (k *Katalog) IsTargetRegistered(target string) bool {
	if k == nil {
		return false
	}
	_, ok := k.targetIndex[strings.ToLower(strings.TrimSpace(target))]
	return ok
}

// IsGVKRegistered returns true if the GVK exists in the index.
// Case-insensitive.
func (k *Katalog) IsGVKRegistered(gvkString string) bool {
	if k == nil {
		return false
	}
	_, ok := k.gvkIndex[strings.ToLower(strings.TrimSpace(gvkString))]
	return ok
}

// IsGVRRegistered returns true if the GVR exists in the index.
// Case-insensitive.
func (k *Katalog) IsGVRRegistered(gvrString string) bool {
	if k == nil {
		return false
	}
	_, ok := k.gvrIndex[strings.ToLower(strings.TrimSpace(gvrString))]
	return ok
}

// -----------------------------------------------------------------------------
// List Methods
// -----------------------------------------------------------------------------

// ListTargets returns all registered targets (lowercase), sorted.
func (k *Katalog) ListTargets() []string {
	if k == nil {
		return nil
	}
	targets := make([]string, 0, len(k.targetIndex))
	for target := range k.targetIndex {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

// ListKinds returns all registered kinds (lowercase), sorted.
func (k *Katalog) ListKinds() []string {
	if k == nil {
		return nil
	}
	kinds := make([]string, 0, len(k.kindIndex))
	for kind := range k.kindIndex {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// ListGVKs returns all registered GVK strings (lowercase), sorted.
func (k *Katalog) ListGVKs() []string {
	if k == nil {
		return nil
	}
	gvks := make([]string, 0, len(k.gvkIndex))
	for gvk := range k.gvkIndex {
		gvks = append(gvks, gvk)
	}
	sort.Strings(gvks)
	return gvks
}

// ListGVRs returns all registered GVR strings (lowercase), sorted.
func (k *Katalog) ListGVRs() []string {
	if k == nil {
		return nil
	}
	gvrs := make([]string, 0, len(k.gvrIndex))
	for gvr := range k.gvrIndex {
		gvrs = append(gvrs, gvr)
	}
	sort.Strings(gvrs)
	return gvrs
}
