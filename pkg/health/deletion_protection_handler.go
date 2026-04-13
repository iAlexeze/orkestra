// health/deletion_protection_handler.go
//
// /deletion-protection webhook handler.
//
// Registered only when security.deletionProtection.enabled: true in the Katalog.
// Separate from /validate — different semantics, different failure policy (Fail vs Ignore).
//
// Intercepts DELETE operations on:
//   - customresourcedefinitions owned by this operator
//   - the Orkestra deployment itself
//
// All other resources and operations are allowed immediately.
//
// failurePolicy: Fail — if Orkestra is unreachable, DELETE is blocked.
// This is intentional: a down Orkestra means protection is still active.
// To decommission: set deletionProtection.enabled: false, redeploy, then delete.
package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
)

// deletionProtectionHandler is the HTTP handler for /deletion-protection.
// It is registered on the HTTPS mux when deletion protection is enabled.
func (h *HealthServer) deletionProtectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	var review AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		logger.Error().Err(err).Msg("deletion-protection: failed to decode AdmissionReview")
		http.Error(w, "invalid AdmissionReview", http.StatusBadRequest)
		return
	}

	if review.Request == nil {
		http.Error(w, "missing request", http.StatusBadRequest)
		return
	}

	req := review.Request

	// Allow all non-DELETE operations immediately — this webhook is DELETE-only.
	// The webhook rules already filter to DELETE, but guard here for safety.
	if req.Operation != "DELETE" {
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	// Check: is this a CRD we protect?
	isCRD := req.Resource.Group == "apiextensions.k8s.io" &&
		req.Resource.Resource == "customresourcedefinitions"

	if isCRD {
		if h.isProtectedCRD(req.Name) {
			logger.Info().
				Str("crd", req.Name).
				Str("uid", req.UID).
				Msg("deletion-protection: blocking CRD deletion")

			metrics.RecordDeletionProtectionBlocked(req.Name)
			_ = time.Since(start)

			h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
				UID:     req.UID,
				Allowed: false,
				Status: &AdmissionStatus{
					Message: fmt.Sprintf(
						"[orkestra] CRD %q is protected from deletion.\n"+
							"To delete it: set security.deletionProtection.enabled: false "+
							"in the Katalog, redeploy Orkestra, then delete the CRD.",
						req.Name,
					),
					Code: 403,
				},
			})
			return
		}
		// CRD not managed by us — allow
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	// Check: is this the Orkestra deployment?
	// The webhook ObjectSelector already narrows to the Orkestra deployment,
	// but double-check here for clarity.
	isDeployment := req.Resource.Group == "apps" &&
		req.Resource.Resource == "deployments"

	if isDeployment {
		logger.Info().
			Str("deployment", req.Name).
			Str("namespace", req.Namespace).
			Str("uid", req.UID).
			Msg("deletion-protection: blocking Orkestra deployment deletion")

		metrics.RecordDeletionProtectionBlocked("orkestra-deployment")

		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Status: &AdmissionStatus{
				Message: "[orkestra] The Orkestra operator deployment is protected from deletion. " +
					"Set security.deletionProtection.enabled: false in the Katalog first.",
				Code: 403,
			},
		})
		return
	}

	// Anything else — allow
	h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
		UID: req.UID, Allowed: true,
	})
}

// isProtectedCRD returns true when the given CRD full name
// (e.g. "pipelines.platform.io") is managed by this Orkestra instance.
// Uses the cached protected CRD name set built from the Katalog at startup.
func (h *HealthServer) isProtectedCRD(crdName string) bool {
	if h.protectedCRDNames == nil {
		return false
	}
	_, ok := h.protectedCRDNames[crdName]
	return ok
}
