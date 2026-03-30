package template

import (
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResolveLabels evaluates template expressions in label and annotation values.
// Keys are never template expressions — only values are resolved.
func (r *Resolver) ResolveLabels(labels []orktypes.ResourceLabel) ([]orktypes.ResourceLabel, error) {
	resolved := make([]orktypes.ResourceLabel, 0, len(labels))
	for _, l := range labels {
		v, err := r.Resolve(l.Value)
		if err != nil {
			return nil, fmt.Errorf("label %q: %w", l.Key, err)
		}
		resolved = append(resolved, orktypes.ResourceLabel{Key: l.Key, Value: v})
	}
	return resolved, nil
}

// ResolveStringSlice resolves template expressions in each element of a string slice.
// Each element is resolved independently — one failing element does not affect others.
// Used for toNamespaces where each namespace may be a template expression.
//
// Example:
//
//	input:  ["{{ .metadata.namespace }}", "monitoring", "{{ .spec.extraNamespace }}"]
//	output: ["my-app", "monitoring", "platform"]
func (r *Resolver) ResolveStringSlice(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(values))
	for i, v := range values {
		rv, err := r.Resolve(v)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		// Skip empty results — a template that resolves to "" means
		// the field was not set on the CR. This avoids not creating a namespace named "".
		if rv != "" {
			resolved = append(resolved, rv)
		}
	}
	return resolved, nil
}

// OwnerName returns the CR name. Used by registry Resolve() for default naming.
func (r *Resolver) OwnerName() string { return r.ownerName }

// OwnerNamespace returns the CR namespace.
func (r *Resolver) OwnerNamespace() string { return r.ownerNamespace }

// ── Internal ──────────────────────────────────────────────────────────────────

// objectToMap converts a domain.Object to map[string]interface{} for template execution.
//
// Fast path: *unstructured.Unstructured already has the full object map including
// all spec fields — used directly with zero allocation overhead.
//
// Typed objects: only metadata fields are extracted. Spec fields are not accessible
// without reflection or JSON round-trip. Typed object users should use Typed mode
// hooks with 'Go' for full spec access rather than YAML template expressions.
func objectToMap(obj domain.Object) (map[string]interface{}, error) {
	// Fast path — unstructured has full map natively
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u.Object, nil
	}

	// Typed fallback — metadata only
	// spec fields not available without reflection on typed objects
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":        obj.GetName(),
			"namespace":   obj.GetNamespace(),
			"labels":      obj.GetLabels(),
			"annotations": obj.GetAnnotations(),
		},
	}, nil
}

// resolveRawValue navigates a dot-notation path extracted from a template
// expression and returns the raw value — preserving the original type
// ([]interface{}, map, string, float64, bool) rather than stringifying it.
//
// Input: "{{ .spec.targetNamespaces }}" → navigates data["spec"]["targetNamespaces"]
// Returns nil when the path does not exist.
func resolveRawValue(data map[string]interface{}, expr string) interface{} {
	// Extract the path from "{{ .spec.targetNamespaces }}"
	path := strings.TrimSpace(expr)
	path = strings.TrimPrefix(path, "{{")
	path = strings.TrimSuffix(path, "}}")
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, ".")

	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	var current interface{} = data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}
