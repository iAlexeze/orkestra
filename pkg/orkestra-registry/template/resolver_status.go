// pkg/orkestra-registry/template/resolver_status.go
package template

import (
	"fmt"
	"strings"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ResolveStatusFields evaluates template expressions in declarative status field
// declarations and returns a map[string]interface{} ready to merge into the
// CR's status patch.
//
// Paths are relative to status and support dot-notation for nesting:
//
//	"phase"          → { "phase": "Running" }
//	"database.host"  → { "database": { "host": "..." } }
//	"ready"          → { "ready": "true" }
//
// The returned map is the body of the status subresource patch — it is merged
// into the existing status rather than replacing it. Fields not declared here
// are untouched.
//
// Errors in individual field expressions are collected and returned together
// so the caller sees all problems in one reconcile, not one at a time.
func (r *Resolver) ResolveStatusFields(fields []orktypes.StatusFieldSpec) (map[string]interface{}, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	result := map[string]interface{}{}
	var errs []string

	for _, f := range fields {
		if f.Path == "" {
			errs = append(errs, "status field with empty path — skipped")
			continue
		}

		// ── Evaluate when: conditions ──────────────────────────────────────
		// evaluateConditions lives in this package (resolver_conditions.go).
		// r.data already includes .children.* if WithChildren was called.
		if len(f.When) > 0 && !evaluateConditions(r.data, f.When) {
			continue
		}

		resolved, err := r.Resolve(f.Value)
		if err != nil {
			errs = append(errs, fmt.Sprintf("status.%s: %v", f.Path, err))
			continue
		}

		if err := setNestedStatusField(result, f.Path, resolved); err != nil {
			errs = append(errs, fmt.Sprintf("status.%s: %v", f.Path, err))
		}
	}

	if len(errs) > 0 {
		return result, fmt.Errorf("resolving status fields:\n  %s",
			strings.Join(errs, "\n  "))
	}

	return result, nil
}

// setNestedStatusField sets a value at a dot-notation path inside a
// map[string]interface{}, creating intermediate maps as needed.
//
// "phase"          → dst["phase"] = value
// "database.host"  → dst["database"]["host"] = value
// "a.b.c"          → dst["a"]["b"]["c"] = value
//
// If an intermediate path segment already contains a non-map value, it is
// replaced with a map. This avoids panics from type-assertion failures.
func setNestedStatusField(dst map[string]interface{}, path string, value string) error {
	parts := strings.SplitN(path, ".", 2)

	if len(parts) == 1 {
		// Terminal — set directly
		dst[path] = value
		return nil
	}

	// Intermediate — recurse into or create the child map
	key := parts[0]
	rest := parts[1]

	var child map[string]interface{}
	if existing, ok := dst[key]; ok {
		child, ok = existing.(map[string]interface{})
		if !ok {
			// Existing value is a scalar — replace with map
			child = map[string]interface{}{}
		}
	} else {
		child = map[string]interface{}{}
	}

	dst[key] = child
	return setNestedStatusField(child, rest, value)
}
