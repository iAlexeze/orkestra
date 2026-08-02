package applyapi

import (
	"strings"

	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
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
		base = utils.DeepCopyMap(obj)
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
		for _, path := range cfg.Exclude {
			// Each entry may itself be a template expression — evaluate it.
			// toList returns a []string from a comma-separated value, but here
			// we handle the case where a single entry resolves to a plain path.
			resolved, err := resolver.Resolve(path)
			if err != nil || resolved == "<no value>" {
				continue // silent — exclusion failures never break the response
			}
			utils.DeleteNestedPath(base, strings.TrimSpace(resolved))
		}
	}

	return base
}
