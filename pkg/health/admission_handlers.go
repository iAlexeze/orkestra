// health/admission_handlers.go
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── /validate handler ──────────────────────────────────────────────────────
//
// The Kubernetes API server POSTs an AdmissionReview to this endpoint for
// every CREATE and UPDATE operation on a resource covered by the
// ValidatingWebhookConfiguration that Orkestra registered at startup.
//
// Orkestra evaluates the object against the validation rules declared in
// the Katalog for the object's GVK. Rules with action: deny cause rejection.
// Rules with action: warn allow the operation but add warnings to the response.
//
// Flow:
//   POST /validate
//     → decode AdmissionReview
//     → look up rules by GVR
//     → evaluate all rules against the object
//     → build response (allow or deny, with warnings)
//     → encode and return AdmissionReview

func (h *HealthServer) validationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	// Decode the incoming AdmissionReview
	var review AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		logger.Error().Err(err).Msg("admission/validate: failed to decode AdmissionReview")
		http.Error(w, "invalid AdmissionReview", http.StatusBadRequest)
		return
	}

	if review.Request == nil {
		http.Error(w, "missing request in AdmissionReview", http.StatusBadRequest)
		return
	}

	req := review.Request

	logger.Debug().
		Str("uid", req.UID).
		Str("kind", req.Kind.Kind).
		Str("name", req.Name).
		Str("namespace", req.Namespace).
		Str("operation", req.Operation).
		Msg("admission/validate: received request")

	// Build the response — default to allowed
	resp := &AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
	}

	// Look up the validation rules for this GVR
	gvrKey := gvrToKey(req.Resource)
	rules := h.admissionRegistry.GetValidationRules(gvrKey)

	if rules == nil || len(rules.Rules) == 0 {
		// No rules for this resource — allow immediately
		// This should not happen if the webhook configuration is correct,
		// since webhook endpoints is conditionally created.
		// But it's a safe fallback.
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	// Unmarshal the object for evaluation
	var obj map[string]interface{}
	if err := json.Unmarshal(req.Object, &obj); err != nil {
		logger.Error().Err(err).
			Str("uid", req.UID).
			Str("kind", req.Kind.Kind).
			Str("name", req.Name).
			Str("namespace", req.Namespace).
			Msg("admission/validate: failed to unmarshal object")
		resp.Allowed = false
		resp.Status = &AdmissionStatus{
			Message: fmt.Sprintf("internal error: could not parse object: %v", err),
			Code:    500,
		}
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	// Evaluate all validation rules
	denials, warnings := h.evaluateValidationRules(obj, rules, req.Kind.Kind)

	// Warnings — add to response regardless of denials
	// action: warn rules always surface to the user
	for _, w := range warnings {
		resp.Warnings = append(resp.Warnings,
			fmt.Sprintf("orkestra: field %q: %s", w.Field, w.Message))
	}

	// Record stats and metrics now that we know the outcome
	duration := time.Since(start)
	crdName := req.Kind.Kind

	if len(denials) > 0 {
		h.admissionStats.RecordValidationDenied(duration)
		metrics.RecordValidationOutcome(crdName, "denied", metrics.MetricSourceAdmission)
		for _, d := range denials {
			metrics.RecordValidationViolation(crdName, d.Field, d.RuleType, "deny", metrics.MetricSourceAdmission)
		}
		for _, w := range warnings {
			metrics.RecordValidationViolation(crdName, w.Field, w.RuleType, "warn", metrics.MetricSourceAdmission)
		}
	} else if len(warnings) > 0 {
		h.admissionStats.RecordValidationWarned(duration)
		metrics.RecordValidationOutcome(crdName, "warned", metrics.MetricSourceAdmission)
		for _, w := range warnings {
			metrics.RecordValidationViolation(crdName, w.Field, w.RuleType, "warn", metrics.MetricSourceAdmission)
		}
	} else {
		h.admissionStats.RecordValidationAllowed(duration)
		metrics.RecordValidationOutcome(crdName, "allowed", metrics.MetricSourceAdmission)
	}
	metrics.RecordValidationDurationSeconds(crdName, duration.Seconds())

	// Denials — reject the operation
	if len(denials) > 0 {
		resp.Allowed = false
		msgs := make([]string, 0, len(denials))
		for _, d := range denials {
			msgs = append(msgs, fmt.Sprintf("field %q: %s (got: %q)", d.Field, d.Message, d.Got))
		}
		resp.Status = &AdmissionStatus{
			Message: fmt.Sprintf(
				"\n\n[Orkestra Validation] validation failed\n\n"+
					"%s/%s/%s was blocked due to the following policies:\n"+
					" %s\n\n", req.Kind.Kind, req.Name, req.Namespace, strings.Join(msgs, "; ")),
			Code: 400,
		}

		logger.Info().
			Str("kind", req.Kind.Kind).
			Str("name", req.Name).
			Str("namespace", req.Namespace).
			Str("uid", req.UID).
			Int("denials", len(denials)).
			Int("warnings", len(warnings)).
			Msg("admission/validate: rejected")
	} else if len(warnings) > 0 {
		logger.Info().
			Str("kind", req.Kind.Kind).
			Str("name", req.Name).
			Str("namespace", req.Namespace).
			Int("warnings", len(warnings)).
			Msg("admission/validate: allowed with warnings")
	} else {
		logger.Debug().
			Str("kind", req.Kind.Kind).
			Str("name", req.Name).
			Msg("admission/validate: allowed")
	}

	h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
}

// ── /mutate handler ────────────────────────────────────────────────────────
//
// The Kubernetes API server POSTs an AdmissionReview to this endpoint for
// every CREATE and UPDATE operation on a resource covered by the
// MutatingWebhookConfiguration that Orkestra registered at startup.
//
// Mutation fires BEFORE validation in the Kubernetes admission chain.
// This ensures defaults are applied before validation sees the object —
// a field that would fail a min: "1" rule because it is absent will
// receive its default: "1" from mutation before validation evaluates it.
//
// Orkestra computes the diff between the original object and the mutated
// object, builds a JSON patch (RFC 6902), and returns it in the response.
// The API server applies the patch to the object before proceeding.
//
// Flow:
//   POST /mutate
//     → decode AdmissionReview
//     → look up mutation rules by GVR
//     → apply rules to a copy of the object
//     → diff original vs mutated → JSON patch
//     → return AdmissionReview with patch

func (h *HealthServer) mutationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	var review AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		logger.Error().Err(err).Msg("admission/mutate: failed to decode AdmissionReview")
		http.Error(w, "invalid AdmissionReview", http.StatusBadRequest)
		return
	}

	if review.Request == nil {
		http.Error(w, "missing request in AdmissionReview", http.StatusBadRequest)
		return
	}

	req := review.Request

	logger.Debug().
		Str("uid", req.UID).
		Str("kind", req.Kind.Kind).
		Str("name", req.Name).
		Str("namespace", req.Namespace).
		Str("operation", req.Operation).
		Msg("admission/mutate: received request")

	// Default response — allow with no patch
	resp := &AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
	}

	// Look up mutation rules for this GVR
	gvrKey := gvrToKey(req.Resource)
	rules := h.admissionRegistry.GetMutationRules(gvrKey)

	if rules == nil || len(rules.Rules) == 0 {
		// No rules — return allow with no patch
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	// Unmarshal the incoming object
	var original map[string]interface{}
	if err := json.Unmarshal(req.Object, &original); err != nil {
		logger.Error().Err(err).
			Str("uid", req.UID).
			Str("kind", req.Kind.Kind).
			Str("name", req.Name).
			Str("namespace", req.Namespace).
			Msg("admission/mutate: failed to unmarshal object")
		// Allow — don't block on mutation failure
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	// Apply mutation rules to a copy of the object
	// We work on a copy so we can diff original vs mutated
	mutated := deepCopyMap(original)
	changes, err := h.applyMutationRules(context.Background(), mutated, rules, req.Kind.Kind)
	if err != nil {
		logger.Error().Err(err).
			Str("kind", req.Kind.Kind).
			Str("name", req.Name).
			Msg("admission/mutate: error applying rules — allowing without mutation")
		// Never block on mutation errors — allow without patch
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	crdName := req.Kind.Kind

	if len(changes) == 0 {
		// No changes needed — allow with no patch
		duration := time.Since(start)
		h.admissionStats.RecordMutationSkipped(duration)
		metrics.RecordMutationOutcome(crdName, "skipped", metrics.MetricSourceAdmission)
		metrics.RecordMutationDurationSeconds(crdName, duration.Seconds())

		logger.Debug().
			Str("kind", crdName).
			Str("name", req.Name).
			Msg("admission/mutate: no changes — allowed without patch")
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	// Build JSON patch from the changes
	// We use the changes slice directly rather than diffing the full objects —
	// more precise, avoids false diffs from map ordering
	patch, err := buildJSONPatch(changes)
	if err != nil {
		logger.Error().Err(err).
			Str("kind", crdName).
			Str("name", req.Name).
			Msg("admission/mutate: failed to build patch — allowing without mutation")
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
		return
	}

	resp.Patch = patch
	resp.PatchType = ptrString(jsonPatchType)

	duration := time.Since(start)
	h.admissionStats.RecordMutationApplied(duration)
	metrics.RecordMutationOutcome(crdName, "applied", metrics.MetricSourceAdmission)
	for _, c := range changes {
		metrics.RecordMutationFieldApplied(crdName, c.Field, c.ChangeType, metrics.MetricSourceAdmission)
	}
	metrics.RecordMutationDurationSeconds(crdName, duration.Seconds())

	logger.Info().
		Str("kind", crdName).
		Str("name", req.Name).
		Int("changes", len(changes)).
		Msg("admission/mutate: defaults applied")

	h.writeAdmissionResponse(w, review.APIVersion, review.Kind, resp)
}

// ── Shared helpers ────────────────────────────────────────────────────────

// writeAdmissionResponse encodes and writes the AdmissionReview response.
// The response wraps the AdmissionResponse in the same AdmissionReview envelope.
func (h *HealthServer) writeAdmissionResponse(
	w http.ResponseWriter,
	apiVersion, kind string,
	resp *AdmissionResponse,
) {
	review := AdmissionReview{
		APIVersion: apiVersion,
		Kind:       kind,
		Response:   resp,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(review); err != nil {
		logger.Error().Err(err).Str("uid", resp.UID).Msg("admission: failed to encode response")
	}
}

// gvrToKey builds a consistent string key from a GroupVersionResource.
// Used to look up rules in the admission registry.
// Format: "group/version/resource" — e.g. "demo.orkestra.io/v1alpha1/websites"
// For core group resources: "v1/pods"
func gvrToKey(gvr metav1.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}

// deepCopyMap performs a deep copy of a map[string]interface{}.
// Needed so that mutation rules work on a copy — the original is preserved
// for diffing.
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	// Marshal and unmarshal — correct for any JSON-representable structure
	// and simpler than recursive type-switch copying.
	b, _ := json.Marshal(src)
	var dst map[string]interface{}
	_ = json.Unmarshal(b, &dst)
	return dst
}

// JSONPatchOp is one operation in a JSON Patch document (RFC 6902).
type JSONPatchOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// buildJSONPatch converts a slice of field changes into a JSON patch document.
// Each change becomes one "add" or "replace" operation at the field's path.
//
// Dot-notation field paths are converted to JSON Pointer paths (RFC 6901):
//
//	"spec.replicas" → "/spec/replicas"
//	"metadata.labels.team" → "/metadata/labels/team"
func buildJSONPatch(changes []fieldChange) ([]byte, error) {
	ops := make([]JSONPatchOp, 0, len(changes))
	for _, change := range changes {
		// Convert dot-notation to JSON Pointer
		ptr := "/" + strings.ReplaceAll(change.Field, ".", "/")

		op := "replace"
		if change.OldValue == "" {
			op = "add" // field was absent — use add
		}

		ops = append(ops, JSONPatchOp{
			Op:    op,
			Path:  ptr,
			Value: change.NewValue,
		})
	}
	return json.Marshal(ops)
}

// fieldChange records one field mutation for patch construction.
type fieldChange struct {
	Field      string
	OldValue   string
	NewValue   string
	ChangeType string // "default" or "override" — for metrics
}
