// pkg/reconciler/run_status.go
//
// Status management — three layers:
//
// Layer 1 — Automatic Ready condition.
//
//	Written after every reconcile regardless of Katalog declarations.
//	reason: ReconcileSucceeded on success, ReconcileError on failure.
//	No Katalog declaration required. Requires subresources.status in the CRD.
//
// Layer 2 — Declarative status fields.
//
//	Written from status.fields declarations in the Katalog.
//	Template expressions resolved against the live CR object map.
//	Optional when: conditions gate individual field writes — this is what
//	makes declarative state machines possible.
//	Written only on successful reconcile.
//
// Layer 3 — Child resource propagation.
//
//	Children are read after runTemplateReconcile and injected into the
//	resolver context via WithChildren. The returned resolver is used for
//	all subsequent status resolution.
//	Status field expressions reference children as:
//	  {{ .children.deployment.status.readyReplicas }}
//	  {{ .children.cronjob.status.lastScheduleTime }}
//	Children are keyed by lowercase Kind — matches kubectl conventions.
package reconciler

import (
	"context"
	//	"fmt"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
)

// patchStatusWithChildren is the top-level status entry point called from
// reconcileImpl after the reconcile work completes.
//
// Always called — even when reconcile returned an error — so that
// Ready=False is written on failure. The error argument controls Layer 1.
func (r *GenericReconciler[PTR]) patchStatusWithChildren(
	ctx context.Context,
	obj PTR,
	resolver *orktmpl.Resolver,
	reconcileErr error,
) {
	// ── Layer 3: extend resolver with child resource state ─────────────────
	// WithChildren returns a new Resolver — the original is not modified.
	// The new resolver's data map includes a "children" key so that
	// status field expressions can reference child status:
	//   {{ .children.cronjob.status.lastScheduleTime }}
	if reconcileErr == nil && (r.operatorBox.OnCreate != nil || r.operatorBox.OnReconcile != nil) {
		children := ReadChildren(ctx, r.kube, obj, resolver, r.operatorBox, r.crd)
		resolver = resolver.WithChildren(children) // ← reassign — WithChildren returns new resolver
	}

	// ── Layer 1 + 2: patch status ──────────────────────────────────────────
	if err := runStatusPatch(ctx, r, obj, resolver, reconcileErr); err != nil {
		logger.FromContext(ctx).Warn().Err(err).
			Str("name", obj.GetName()).
			Msg("status: patch failed — continuing")
	}
}

// runStatusPatch writes Layer 1 (Ready condition) and Layer 2 (declared fields).
//
// Works for both unstructured and typed CRDs. Previously, typed objects hit an
// early return here because PatchStatus was assumed to require *unstructured.Unstructured.
// PatchStatus only needs domain.Object (for GetName/GetNamespace), so obj is passed
// directly — objectToMap in the template package already handles the JSON round-trip
// for typed spec access, so there is no longer any structural gap between the two modes.
func runStatusPatch[PTR domain.Object](
	ctx context.Context,
	r *GenericReconciler[PTR],
	obj PTR,
	resolver *orktmpl.Resolver,
	reconcileErr error,
) error {
	// ── Layer 1: Ready condition ───────────────────────────────────────────
	// Always written — on success and failure — so operators can monitor
	// the Ready condition without knowing anything else about the CRD.
	// Except for some builtins or explicitly required
	if r.crd.SkipStatusSubresource() {
		return nil
	}

	patch := map[string]interface{}{}

	cond := buildReadyCondition(reconcileErr, obj.GetGeneration())
	patch["conditions"] = []interface{}{cond}

	// Only patch if necessary
	if !r.crd.SkipObservedGeneration() {
		patch["observedGeneration"] = obj.GetGeneration()
	}

	// ── Layer 2: Declared status fields (conditional) ─────────────────────
	// Only written on successful reconcile. Errors in field resolution are
	// logged as warnings and do not fail the reconcile.
	logger.FromContext(ctx).Debug().
		Str("name", obj.GetName()).
		Bool("has_status_config", r.operatorBox.Status != nil && r.operatorBox.Status.HasFields()).
		Bool("reconcile_error", reconcileErr != nil).
		AnErr("reconcile_err", reconcileErr).
		Msg("status: layer2 evaluation")

	if r.operatorBox.Status != nil && r.operatorBox.Status.HasFields() && reconcileErr == nil {
		resolved, err := resolver.ResolveStatusFields(r.operatorBox.Status.Fields)
		if err != nil {
			logger.FromContext(ctx).Warn().Err(err).
				Str("name", obj.GetName()).
				Msg("status: some fields failed to resolve")
		}
		logger.FromContext(ctx).Debug().
			Str("name", obj.GetName()).
			Interface("resolved_fields", resolved).
			Msg("status: layer2 resolved fields")
		for k, v := range resolved {
			patch[k] = v
		}
	}

	return r.kube.PatchStatus(ctx, obj, r.crd.GVR(), patch)
}

// buildReadyCondition constructs the standard Kubernetes Ready condition map.
//
// Signature: (reconcileErr error, generation int64)
// — err nil   → Ready=True,  reason=ReconcileSucceeded
// — err non-nil → Ready=False, reason=ReconcileError, message=err.Error() truncated
//
// The condition is returned as map[string]interface{} for direct inclusion
// in the status patch — avoids an extra metav1.Condition → unstructured conversion.
func buildReadyCondition(reconcileErr error, generation int64) map[string]interface{} {
	now := time.Now().UTC().Format(time.RFC3339)

	if reconcileErr == nil {
		return map[string]interface{}{
			"type":               "Ready",
			"status":             "True",
			"reason":             "ReconcileSucceeded",
			"message":            "",
			"lastTransitionTime": now,
			"observedGeneration": generation,
		}
	}

	msg := reconcileErr.Error()
	if len(msg) > 256 {
		msg = msg[:253] + "..."
	}

	return map[string]interface{}{
		"type":               "Ready",
		"status":             "False",
		"reason":             "ReconcileError",
		"message":            msg,
		"lastTransitionTime": now,
		"observedGeneration": generation,
	}
}

// parseNumeric parses a string as float64 for numeric comparisons.
// Used by the gt and lt operators on child status fields like
// .children.job.status.succeeded which arrive as "1", "0", etc.
// func parseNumeric(s string) (float64, error) {
// 	var f float64
// 	_, err := fmt.Sscanf(s, "%f", &f)
// 	return f, err
// }
