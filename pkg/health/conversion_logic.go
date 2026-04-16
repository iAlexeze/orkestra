// health/conversion_logic.go
package health

import (
	"fmt"
	"strings"

	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// applyConversion converts obj from its current version to targetAPIVersion.
//
// targetAPIVersion is the full apiVersion string sent by Kubernetes in the
// ConversionReview request — e.g. "demo.orkestra.io/v1alpha1" or "v1".
// The function extracts the bare version before path lookup.
//
// The output object has its apiVersion updated to targetAPIVersion and its
// spec replaced by the converted spec from the matching path.
func applyConversion(
	obj map[string]interface{},
	rules *orktypes.ConversionRules,
	targetAPIVersion string, // full apiVersion from ConversionReview.Request.DesiredAPIVersion
) (map[string]interface{}, error) {
	sourceAPIVersion, ok := obj["apiVersion"].(string)
	if !ok || sourceAPIVersion == "" {
		return nil, fmt.Errorf("object missing apiVersion")
	}

	// Extract bare version strings from full apiVersion strings.
	// Kubernetes sends "demo.orkestra.io/v1alpha1" — we need "v1alpha1".
	// Core group resources send "v1" — splitAPIVersion handles both forms.
	_, sourceVersion := splitAPIVersion(sourceAPIVersion)
	_, targetVersion := splitAPIVersion(targetAPIVersion)

	// No-op — already at the requested version
	if sourceVersion == targetVersion {
		out := copyMap(obj)
		out["apiVersion"] = targetAPIVersion
		return out, nil
	}

	// Find the explicit (from, to) path.
	// Both endpoints are now bare version strings — no ambiguity.
	path := rules.FindPath(sourceVersion, targetVersion)
	if path == nil {
		return nil, fmt.Errorf(
			"no conversion path declared for %s → %s in kind %q.\n"+
				"Add a path to the Katalog:\n"+
				"  conversion:\n"+
				"    paths:\n"+
				"      - from: %s\n"+
				"        to: %s\n"+
				"        spec:\n"+
				"          # fields in %s format",
			sourceVersion, targetVersion, rules.Kind,
			sourceVersion, targetVersion, targetVersion,
		)
	}

	// Resolve template expressions in the path spec against the source object.
	// The full object map is available: {{ .spec.image }}, {{ .metadata.name }}, etc.
	resolver := orktmpl.NewResolverFromMap(obj)

	convertedSpec, err := resolveMap(resolver, path.Spec)
	if err != nil {
		return nil, fmt.Errorf(
			"converting spec from %s to %s: %w",
			sourceVersion, targetVersion, err,
		)
	}

	out := copyMap(obj)
	out["apiVersion"] = targetAPIVersion // preserve the exact string Kubernetes sent
	out["spec"] = convertedSpec

	return out, nil
}

// resolveMap recursively walks a map and resolves string values as Go templates.
// Non-string values (numbers, booleans, nested maps, arrays) are handled correctly.
// Template expressions like {{ .spec.image }} are resolved against the source object.
func resolveMap(r *orktmpl.Resolver, src map[string]interface{}) (map[string]interface{}, error) {
	dst := make(map[string]interface{}, len(src))

	for k, v := range src {
		resolved, err := resolveValue(r, v)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", k, err)
		}
		dst[k] = resolved
	}

	return dst, nil
}

// resolveValue resolves a single value — dispatches by type.
func resolveValue(r *orktmpl.Resolver, v interface{}) (interface{}, error) {
	switch tv := v.(type) {
	case string:
		// Template expression or plain string
		return r.Resolve(tv)

	case map[string]interface{}:
		// Nested object — recurse
		return resolveMap(r, tv)

	case []interface{}:
		// Array — resolve each element
		arr := make([]interface{}, len(tv))
		for i, item := range tv {
			resolved, err := resolveValue(r, item)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			arr[i] = resolved
		}
		return arr, nil

	default:
		// bool, int64, float64, nil — pass through unchanged
		return v, nil
	}
}

// copyMap performs a shallow copy of a map.
// Deep copy is not needed here — we replace "apiVersion" and "spec"
// entirely rather than mutating nested values.
func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// splitAPIVersion splits a Kubernetes apiVersion string into group and version.
//
// "demo.orkestra.io/v1alpha1" → ("demo.orkestra.io", "v1alpha1")
// "apps/v1"                  → ("apps", "v1")
// "v1"                       → ("", "v1")       ← core group
// "v1alpha1"                 → ("", "v1alpha1")  ← bare version
func splitAPIVersion(apiVersion string) (group, version string) {
	i := strings.LastIndex(apiVersion, "/")
	if i < 0 {
		return "", apiVersion
	}
	return apiVersion[:i], apiVersion[i+1:]
}

// joinAPIVersion reconstructs an apiVersion string.
// Returns just the version when group is empty (core group resources).
func joinAPIVersion(group, version string) string {
	if group == "" {
		return version
	}
	return group + "/" + version
}
