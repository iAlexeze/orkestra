// pkg/reconciler/run_status.go
package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
)

// ── Status patching ────────────────────────────────────────────────────────
//
// patchStatus is called after every reconcile cycle — success or failure.
// It is always best-effort: errors are logged but never returned to the
// workqueue. A failure to update status should never block reconciliation.
//
// Two layers are applied on every call:
//
//   Layer 1 — standard Kubernetes Ready condition and observedGeneration.
//             Always applied unless explicitly disabled in StatusConfig.
//             Requires the CRD to declare: subresources: status: {}
//             If the CRD has no status subresource, the 404 is swallowed silently.
//
//   Layer 2 — declarative status fields from reconciler.status.fields.
//             Applied only when StatusConfig.Fields is non-empty.
//             Resolved by the same template resolver as onCreate templates.
//             Merged with Layer 1 — both patches go in one API call.
//
// The combined patch is applied via a single merge patch to the /status
// subresource. If the CRD has no status subresource, the API server returns
// a 404 which is silently swallowed — reconcile continues unaffected.

// patchStatus writes the status update for obj after a reconcile cycle.
// reconcileErr is nil on success and non-nil on failure.
// Never returns an error — status patching is best-effort.
func (r *GenericReconciler[T]) patchStatus(ctx context.Context, obj T, reconcileErr error) {
	statusPatch := map[string]interface{}{}

	// ── Layer 1: Standard Ready condition ──────────────────────────────────
	if r.rc.Status.ConditionsEnabled() {
		cond := buildReadyCondition(reconcileErr, obj.GetGeneration())
		statusPatch["conditions"] = []interface{}{cond}
		statusPatch["observedGeneration"] = obj.GetGeneration()
	}

	// ── Layer 2: Declarative status fields ─────────────────────────────────
	if r.rc.Status.HasFields() {
		resolver, err := orktmpl.NewResolver(ctx, obj)
		if err != nil {
			logger.FromContext(ctx).Warn().
				Str("name", obj.GetName()).
				Err(err).
				Msg("status: failed to build resolver — declarative fields skipped")
		} else {
			fields, err := resolver.ResolveStatusFields(r.rc.Status.Fields)
			if err != nil {
				// Partial resolution — use whatever resolved successfully
				logger.FromContext(ctx).Warn().
					Str("name", obj.GetName()).
					Err(err).
					Msg("status: some field expressions failed to resolve")
			}
			// Merge declarative fields into the patch.
			// Declarative fields win over Layer 1 on key conflict —
			// if the user declares status.conditions explicitly, that takes priority.
			for k, v := range fields {
				statusPatch[k] = v
			}
		}
	}

	if len(statusPatch) == 0 {
		return
	}

	if err := r.kube.PatchStatus(ctx, obj, r.crd.GVR, statusPatch); err != nil {
		// Not a warning — this is expected when the CRD has no status subresource.
		// Log at debug so it does not pollute logs for CRDs that intentionally
		// have no status.
		logger.FromContext(ctx).Debug().
			Str("name", obj.GetName()).
			Err(err).
			Msg("status: patch failed — CRD may not have a status subresource")
	}
}

// buildReadyCondition constructs the standard Ready condition map.
//
// Format matches the Kubernetes meta/v1 Condition spec:
//
//	type:               Ready
//	status:             "True" | "False"
//	reason:             ReconcileSucceeded | ReconcileError
//	message:            "" | <error message>
//	lastTransitionTime: <RFC3339>
//	observedGeneration: <metadata.generation>
//
// Using map[string]interface{} rather than metav1.Condition to avoid
// a typed import in what must remain a lightweight patch operation.
// The structure is identical to what the API server expects.
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

	// Truncate long error messages — the condition message field has a
	// practical length limit. kubectl describe shows 256 chars cleanly.
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

// ── Integration point in generic.go ──────────────────────────────────────
//
// In reconcileImpl, replace the current success/error handling with:
//
//   err := r.dispatchReconcile(ctx, obj)
//
//   // Status patch is best-effort — always runs, never fails the reconcile.
//   // Called with the reconcile outcome so the Ready condition reflects reality.
//   r.patchStatus(ctx, obj, err)
//
//   if err != nil {
//       logger.FromContext(ctx).Error().Err(err)...
//       r.event.Eventf(obj, corev1.EventTypeWarning, ...)
//       return err
//   }
//
//   r.event.Eventf(obj, corev1.EventTypeNormal, ...)
//   return nil
//
// patchStatus is called before the early return on error so that the
// Ready=False condition is written even when reconciliation fails.
// This is the correct behavior — users should see the error in status
// without needing to query events or logs.

// ── ReconcilerConfig addition ─────────────────────────────────────────────
//
// Add to ReconcilerConfig in pkg/types/types.go:
//
//   // Status declares how Orkestra should manage the CR's status subresource.
//   // Layer 1 (standard conditions) is always active unless explicitly disabled.
//   // Layer 2 (declarative fields) is active when Fields is non-empty.
//   Status *orktypes.StatusConfig `yaml:"status,omitempty"`
//
// The nil check is handled by StatusConfig.ConditionsEnabled() and
// StatusConfig.HasFields() — both are nil-safe.

// ── Example Katalog with full status configuration ────────────────────────
//
//   reconciler:
//     default: true
//
//     # Optional: disable automatic conditions if your CRD schema forbids them
//     status:
//       conditions: true   # default — can be omitted
//
//       # Layer 2: declarative fields
//       fields:
//         - path: phase
//           value: "Running"
//
//         - path: observedReplicas
//           value: "{{ .spec.replicas }}"
//
//         - path: endpoint
//           value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
//
//         - path: version
//           value: "{{ .spec.version }}"
//
//         - path: database.host     # nested → status.database.host
//           value: "{{ .spec.host }}"
//
//         - path: database.port
//           value: "{{ .spec.port }}"
//
// After a successful reconcile, kubectl get website my-site -o yaml shows:
//
//   status:
//     conditions:
//       - type: Ready
//         status: "True"
//         reason: ReconcileSucceeded
//         message: ""
//         lastTransitionTime: "2026-03-28T10:00:00Z"
//         observedGeneration: 3
//     observedGeneration: 3
//     phase: Running
//     observedReplicas: "3"
//     endpoint: my-site.default.svc.cluster.local
//     version: "1.25"
//     database:
//       host: db.platform.svc
//       port: "5432"
//
// After a failed reconcile:
//
//   status:
//     conditions:
//       - type: Ready
//         status: "False"
//         reason: ReconcileError
//         message: "deployment: image pull failed: rpc error..."
//         lastTransitionTime: "2026-03-28T10:01:00Z"
//         observedGeneration: 3
//     observedGeneration: 3
//     # declarative fields are NOT written on error — only on success
//     # this prevents stale status values when the reconcile fails partway through

// statusOnSuccessOnly controls whether declarative fields are written on
// reconcile failure. Default: true — fields only written on success.
// This prevents stale status when the reconcile fails partway through.
// The Ready condition is always written regardless.
const statusOnSuccessOnly = true

// updatedPatchStatus is the full implementation using statusOnSuccessOnly.
// This replaces the simpler patchStatus above once the integration is complete.
func (r *GenericReconciler[T]) updatedPatchStatus(ctx context.Context, obj T, reconcileErr error) {
	statusPatch := map[string]interface{}{}

	// Layer 1: Standard conditions — always written, success or failure
	if r.rc.Status.ConditionsEnabled() {
		cond := buildReadyCondition(reconcileErr, obj.GetGeneration())
		statusPatch["conditions"] = []interface{}{cond}
		statusPatch["observedGeneration"] = obj.GetGeneration()
	}

	// Layer 2: Declarative fields — only written on success
	// Rationale: if the reconcile fails partway through, the spec
	// values the user declared may not reflect actual cluster state.
	// Writing status.phase="Running" when the Deployment failed to create
	// would be actively misleading.
	if reconcileErr == nil && r.rc.Status.HasFields() {
		resolver, err := orktmpl.NewResolver(ctx, obj)
		if err != nil {
			logger.FromContext(ctx).Warn().
				Str("name", obj.GetName()).
				Err(err).
				Msg("status: failed to build resolver — declarative fields skipped")
		} else {
			fields, err := resolver.ResolveStatusFields(r.rc.Status.Fields)
			if err != nil {
				logger.FromContext(ctx).Warn().
					Str("name", obj.GetName()).
					Err(err).
					Msg("status: some field expressions failed")
			}
			for k, v := range fields {
				statusPatch[k] = v
			}
		}
	}

	if len(statusPatch) == 0 {
		return
	}

	if err := r.kube.PatchStatus(ctx, obj, r.crd.GVR, statusPatch); err != nil {
		logger.FromContext(ctx).Debug().
			Str("name", obj.GetName()).
			Err(err).
			Msg("status: patch failed — CRD may not have status subresource declared")
	}
}

// ── Layer 3 design ─────────────────────────────────
//
// Layer 3 adds a "children" context to the resolver, allowing status fields
// to reference child resource state:
//
//   status:
//     fields:
//       - path: readyReplicas
//         value: "{{ .children.deployment.status.readyReplicas }}"
//       - path: loadBalancerIP
//         value: "{{ .children.service.status.loadBalancer.ingress.0.ip }}"
//
// Implementation requires:
//   1. After runTemplateReconcile completes, read back child resources.
//   2. Build a "children" map keyed by resource type and name.
//   3. Extend NewResolver to accept an optional children map.
//   4. Add "children" to the template data map.
//
// The child read-back uses the informer cache where possible (zero API calls
// for resources Orkestra already watches) and falls back to a direct API call
// for resources in different namespaces or of types not watched by this instance.
//
// This is the Layer 3 milestone. Layer 1 and Layer 2 ship first.

// ── patchStatusWithChildren ───────────────────────────────────────────────
func (r *GenericReconciler[T]) patchStatusWithChildren(
	ctx context.Context, obj T, reconcileErr error,
) {
	statusPatch := map[string]interface{}{}

	// Layer 1: standard Ready condition — always
	if r.rc.Status.ConditionsEnabled() {
		cond := buildReadyCondition(reconcileErr, obj.GetGeneration())
		statusPatch["conditions"] = []interface{}{cond}
		statusPatch["observedGeneration"] = obj.GetGeneration()
	}

	// Layer 2 + 3: declarative fields — only on success
	if reconcileErr == nil && r.rc.Status.HasFields() {
		resolver, err := orktmpl.NewResolver(ctx, obj)
		if err != nil {
			logger.FromContext(ctx).Warn().Err(err).
				Msg("status: failed to build resolver")
			goto apply
		}

		// Layer 3: read child resource state and extend the resolver
		// Only reads resources declared in the Katalog — bounded API cost
		if r.rc.OnCreate != nil || r.rc.OnReconcile != nil {
			children := ReadChildren(ctx, r.kube, obj, resolver, r.rc)
			if len(children) > 0 {
				resolver = resolver.WithChildren(children)
			}
		}

		// Layer 2: resolve declarative status fields
		// The resolver now has .children available in template expressions
		fields, err := resolver.ResolveStatusFields(r.rc.Status.Fields)
		if err != nil {
			logger.FromContext(ctx).Warn().Err(err).
				Msg("status: some fields failed to resolve")
		}
		for k, v := range fields {
			statusPatch[k] = v
		}
	}

apply:
	if len(statusPatch) == 0 {
		return
	}

	if err := r.kube.PatchStatus(ctx, obj, r.crd.GVR, statusPatch); err != nil {
		logger.FromContext(ctx).Debug().Err(err).
			Str("name", obj.GetName()).
			Msg("status: patch failed — CRD may not have status subresource")
	}
}

// childrenKey is the template context key for child resource state.
// Reserved for Layer 3.
const childrenKey = "children"

// placeholder to prevent unused const warning during development
var _ = fmt.Sprintf("reserved context key: %s", childrenKey)
