// health/namespace_protection_handler.go
//
// /namespace-protection webhook handler.
//
// Registered only when security.namespaceProtection.enabled: true in the Katalog.
// Separate from /validate — different semantics, different failure policy.
//
// Intercepts CREATE and UPDATE operations on CRDs that declare namespace rules
// (allowedNamespaces or restrictedNamespaces). The webhook rules filter by GVR;
// the handler performs the namespace check.
//
// failurePolicy: Fail — if Orkestra is unreachable, CREATE/UPDATE is blocked.
// This ensures namespace rules remain enforced even during transient outages.
package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
)

// namespaceProtectionHandler is the HTTP handler for /namespace-protection.
// It is registered on the HTTPS mux when namespace protection is enabled.
func (h *HealthServer) namespaceProtectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	var review AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		logger.Error().Err(err).Msg("namespace-protection: failed to decode AdmissionReview")
		http.Error(w, "invalid AdmissionReview", http.StatusBadRequest)
		return
	}

	if review.Request == nil {
		http.Error(w, "missing request", http.StatusBadRequest)
		return
	}

	req := review.Request

	// Only CREATE and UPDATE matter — namespace rules apply at admission.
	if req.Operation != "CREATE" && req.Operation != "UPDATE" {
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	// Lookup namespace rules for this CRD.
	rules, ok := h.namespaceRulesForCRD(req.Resource.Group, req.Resource.Resource)
	if !ok {
		// CRD has no namespace rules — allow.
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	ns := req.Namespace

	// Evaluate namespace rules.
	if !rules.IsNamespaceAllowed(ns) {
		logger.Info().
			Str("crd", req.Resource.Resource+"."+req.Resource.Group).
			Str("namespace", ns).
			Str("uid", req.UID).
			Msg("namespace-protection: blocking CR creation/update in forbidden namespace")

		metrics.RecordNamespaceProtectionBlocked(req.Resource.Resource)
		h.namespaceStats.RecordBlocked()

		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Status: &AdmissionStatus{
				Message: fmt.Sprintf(
					"\n\n[Orkestra Security] Namespace %q is not permitted for this CRD.\n\n"+
						"To allow this namespace, update the CRD's allowedNamespaces or restrictedNamespaces.\n\n",
					ns,
				),
				Code: 403,
			},
		})
		return
	}

	// Allowed.
	h.namespaceStats.RecordAllowed()
	_ = time.Since(start)

	h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
		UID: req.UID, Allowed: true,
	})
}

// namespaceRulesForCRD returns the namespace rules for a CRD, if any.
func (h *HealthServer) namespaceRulesForCRD(group, plural string) (*NamespaceRules, bool) {
	if h.namespaceRuleMap == nil {
		return nil, false
	}
	key := plural + "." + group
	r, ok := h.namespaceRuleMap[key]
	return r, ok
}

type NamespaceRules struct {
	Allowed    map[string]struct{}
	Restricted map[string]struct{}
}

func (r *NamespaceRules) IsNamespaceAllowed(ns string) bool {
	// Allowed list takes precedence.
	if len(r.Allowed) > 0 {
		_, ok := r.Allowed[ns]
		return ok
	}

	// Otherwise restricted list applies.
	if len(r.Restricted) > 0 {
		_, forbidden := r.Restricted[ns]
		return !forbidden
	}

	// No rules — everything allowed.
	return true
}
