package applyapi

import (
	"strings"

	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// EvaluatePayload applies idp.config.response to a CR object and returns the
// shaped result.
//
// Called in two places:
//   - POST /api/v1/apply  — after SSA succeeds; obj is the submitted CR.
//     .status is absent; payload fields that reference status resolve to "".
//   - GET  /api/v1/resources/... — after the CR is fetched from the API
//     server; obj is the full stored CR including .status written by the runtime.
//
// The caller passes obj.Object (the unstructured map). The returned map is the
// value that should be set as "payload" in the response JSON. When config is
// nil or has no payload and no exclude, nil is returned — callers should omit
// the payload key from the response rather than writing an empty object.
//
// This function does not make Kubernetes API calls. It does not access the
// runtime. It resolves templates against what the caller already has.
func EvaluatePayload(
	obj map[string]interface{},
	crd *orktypes.CRDEntry,
	notes orktypes.NoteRegistry,
) map[string]interface{} {
	if crd == nil || crd.IDP == nil || crd.IDP.Config == nil {
		return nil
	}

	cfg := crd.IDP.Config.Response
	if cfg == nil {
		return nil
	}
	if !cfg.HasPayload() && !cfg.HasExclude() {
		return nil
	}

	resolver := orktmpl.NewResolverFromMap(obj).WithUserNotes(notes)

	// ── Step 1: base ─────────────────────────────────────────────────────────
	// Start with the full CR or an empty map depending on default:.
	var base map[string]interface{}
	if cfg.UseDefault() {
		// Deep copy so we never mutate the original fetched object.
		base = deepCopyMap(obj)
	} else {
		base = make(map[string]interface{})
	}

	// ── Step 2: payload ───────────────────────────────────────────────────────
	// Evaluate each declared expression and merge into base.
	// Failures and missing values produce empty strings — never errors.
	// The caller receives the full payload shape so they know what to poll for.
	for key, expr := range cfg.Payload {
		if expr == "" {
			base[key] = ""
			continue
		}
		resolved, err := resolver.Resolve(expr)
		if err != nil || resolved == "<no value>" {
			base[key] = ""
			continue
		}
		base[key] = strings.TrimSpace(resolved)
	}

	// ── Step 3: exclude ───────────────────────────────────────────────────────
	// Resolve the exclude expression to a comma-separated list of paths,
	// then strip each path from base.
	if cfg.HasExclude() {
		paths := resolveExclude(cfg.Exclude, resolver)
		for _, path := range paths {
			deleteNestedPath(base, path)
		}
	}

	return base
}

// resolveExclude resolves the exclude expression to a slice of dot-notation
// field paths. Accepts a plain comma-separated string or a template expression
// that resolves to one. Returns nil on error — exclusion failures are silent
// (the field stays in the response rather than the whole request failing).
func resolveExclude(expr string, resolver *orktmpl.Resolver) []string {
	if expr == "" {
		return nil
	}

	// If the expression contains template syntax, resolve it first.
	var raw string
	if orktypes.IsTemplate(expr) {
		resolved, err := resolver.Resolve(expr)
		if err != nil || resolved == "<no value>" {
			return nil
		}
		raw = resolved
	} else {
		raw = expr
	}

	// Split comma-separated paths and trim whitespace.
	parts := strings.Split(raw, ",")
	paths := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	return paths
}

// deleteNestedPath removes a dot-notation path from a nested map in place.
// Silently does nothing when the path does not exist — partial paths are not
// errors. Supports arbitrary depth: "metadata.managedFields",
// "status.observedGeneration", "metadata.annotations.internal-key".
func deleteNestedPath(obj map[string]interface{}, path string) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 0 || obj == nil {
		return
	}

	key := parts[0]
	if len(parts) == 1 {
		// Leaf — delete this key.
		delete(obj, key)
		return
	}

	// Intermediate — recurse into the nested map if it exists.
	if nested, ok := obj[key].(map[string]interface{}); ok {
		deleteNestedPath(nested, parts[1])
	}
}

// deepCopyMap returns a shallow-to-one-level deep copy of a map[string]interface{}.
// Nested maps are also copied; slices and scalar values share the same pointer.
// Sufficient for our use case: we only modify top-level keys and nested map keys
// via deleteNestedPath — we never mutate slice elements or scalar values.
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		if nested, ok := v.(map[string]interface{}); ok {
			dst[k] = deepCopyMap(nested)
		} else {
			dst[k] = v
		}
	}
	return dst
}
