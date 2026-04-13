// pkg/reconciler/normalize.go
//
// applyNormalize — canonical spec transformation.
//
// Called as the first step of reconcileImpl, before mutation and validation.
// Returns a deep copy of the CR with normalize.spec fields replaced by their
// rendered values. The informer cache object is never modified.
//
// Pipeline position:
//
//	informer cache → DeepCopy → applyNormalize → mutation → validation → runTemplateReconcile
//
// After applyNormalize, every downstream step sees the normalized spec.
// The raw CR in etcd is unchanged — normalize is purely an in-memory operation.
//
// Example — CronJob accepting both string and map schedule:
//
//	normalize:
//	  spec:
//	    schedule: >
//	      {{ if eq (typeOf .spec.schedule) "map" }}
//	        {{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}
//	      {{ else }}
//	        {{ .spec.schedule }}
//	      {{ end }}
//
// After normalize, .spec.schedule is always a cron string.
// onReconcile templates use {{ .spec.schedule }} directly — no branching needed.
package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	"gopkg.in/yaml.v3"
)

// applyNormalize applies the CRD's normalize.spec templates to a deep copy
// of the CR, returning the normalized copy for all downstream reconcile steps.
//
// Returns the original obj unchanged when:
//   - r.crd.Normalize is nil (no normalize block declared)
//   - r.crd.Normalize.Spec is empty
//
// Returns an error when:
//   - Template resolution fails (bad expression)
//   - Path navigation fails (segment is not a map)
//
// The returned object is always a deep copy — safe to mutate.
func (r *GenericReconciler[T]) applyNormalize(ctx context.Context, obj T) (T, error) {
	if r.crd.Normalize == nil || len(r.crd.Normalize.Spec) == 0 {
		return obj, nil
	}

	log := logger.FromContext(ctx)

	// Deep copy — never touch the informer cache object
	cloned := obj.DeepCopyObject().(T)

	// Build resolver against the raw (pre-normalize) spec.
	// normalize templates see the original field values.
	// This is intentional: normalize is a one-pass transformation.
	// If you need field A to reference normalized field B, declare B first
	// and use a single combined expression for A.
	resolver, err := orktmpl.NewResolver(ctx, cloned)
	if err != nil {
		return obj, fmt.Errorf("normalize: building resolver: %w", err)
	}

	// Get the unstructured map for field writes.
	// UnstructuredContent() only exists on *unstructured.Unstructured (dynamic mode).
	// Typed-mode CRDs do not support normalize — skip silently.
	type unstructuredGetter interface {
		UnstructuredContent() map[string]interface{}
	}
	ug, ok := any(cloned).(unstructuredGetter)
	if !ok {
		log.Debug().Msg("normalize: skipping — object is typed mode, normalize requires dynamic mode")
		return cloned, nil
	}
	content := ug.UnstructuredContent()
	if content == nil {
		content = map[string]interface{}{}
	}

	for fieldPath, tpl := range r.crd.Normalize.Spec {
		rendered, err := resolver.Resolve(tpl)
		if err != nil {
			return obj, fmt.Errorf("normalize spec.%s: %w", fieldPath, err)
		}

		// Trim whitespace — multi-line YAML block scalars (>) leave leading/trailing whitespace
		rendered = strings.TrimSpace(rendered)

		// Parse the rendered string to an appropriate Go type.
		// "3" → int64, "true" → bool, "*/5 * * * *" → string.
		// This ensures the normalized spec is correctly typed for downstream
		// comparison and rendering.
		parsed := parseNormalizedValue(rendered)

		// Write at the dot-notation path inside "spec".
		// All normalize.spec paths are relative to spec — prepend "spec."
		fullPath := "spec." + fieldPath
		if err := setNestedNormalized(content, fullPath, parsed); err != nil {
			return obj, fmt.Errorf("normalize spec.%s: setting field: %w", fieldPath, err)
		}

		log.Debug().
			Str("field", fieldPath).
			Str("rendered", rendered).
			Msg("normalize: field normalized")
	}

	return cloned, nil
}

// parseNormalizedValue converts a rendered template string to the most
// appropriate Go type using YAML parsing rules.
//
// YAML type coercion:
//
//	""         → "" (empty string, not nil)
//	"3"        → int (64-bit) via yaml.Unmarshal
//	"3.14"     → float64
//	"true"     → bool
//	"null"     → nil (treated as empty string to avoid nil surprises)
//	"*/5 * * * *" → string (YAML cannot parse this as a number or bool)
//
// For cron strings specifically: YAML will not misparse them because they
// contain characters (*, /) that are not valid in YAML scalars for numbers.
func parseNormalizedValue(s string) interface{} {
	if s == "" {
		return ""
	}

	// Try YAML unmarshal — handles int, float, bool, null cleanly
	var v interface{}
	if err := yaml.Unmarshal([]byte(s), &v); err == nil && v != nil {
		// Guard: yaml unmarshals plain integers as int — convert to int64
		// for consistency with the unstructured map's expected types
		switch t := v.(type) {
		case int:
			return int64(t)
		case nil:
			return "" // treat null as empty string
		default:
			return t
		}
	}

	// YAML parsing failed or returned nil — treat as plain string
	return s
}

// setNestedNormalized writes a value at a dot-notation path through a
// map[string]interface{}, creating intermediate maps as needed.
//
// "spec.schedule"              → obj["spec"]["schedule"] = value
// "spec.resources.limits.cpu" → obj["spec"]["resources"]["limits"]["cpu"] = value
//
// Returns an error when an intermediate segment is not a map.
func setNestedNormalized(obj map[string]interface{}, path string, value interface{}) error {
	parts := splitDotPath(path)
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	current := obj
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, ok := current[part]
		if !ok {
			// Create missing intermediate map
			newMap := map[string]interface{}{}
			current[part] = newMap
			current = newMap
			continue
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("path segment %q is not a map (got %T)", part, next)
		}
		current = nextMap
	}

	current[parts[len(parts)-1]] = value
	return nil
}

// splitDotPath splits a dot-notation path into segments.
// "spec.schedule.minute" → ["spec", "schedule", "minute"]
func splitDotPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	return append(parts, path[start:])
}
