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

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
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

	// Self-protection: block deletion of the deletion-protection webhook itself.
	// This closes the bootstrap gap: the webhook must protect itself before it can be deleted.
	if req.Kind.Kind == "ValidatingWebhookConfiguration" &&
		req.Name == deletionProtectionWebhookConfigName {

		logger.Info().
			Str("webhook", req.Name).
			Str("uid", req.UID).
			Msg("deletion-protection: blocking deletion of the deletion-protection webhook")

		metrics.RecordDeletionProtectionBlocked(deletionProtectionWebhookConfigName)
		h.protectionStats.RecordBlocked()

		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Status: &AdmissionStatus{
				Message: fmt.Sprintf(
					"\n\n[Orkestra Security] The deletion-protection webhook \"%s\" is itself protected.\n\n"+

						"To disable deletion protection entirely:\n"+
						"- Set security.deletionProtection.enabled: false in the Katalog\n"+
						"- Redeploy Orkestra, then delete the webhook.\n\n", deletionProtectionWebhookConfigName,
				),
				Code: 403,
			},
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
			h.protectionStats.RecordBlocked()
			_ = time.Since(start)

			h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
				UID:     req.UID,
				Allowed: false,
				Status: &AdmissionStatus{
					Message: fmt.Sprintf(
						"\n\n[orkestra Security] CRD %q is protected from deletion.\n\n"+
							"To delete it:\n"+
							"- Set security.deletionProtection.enabled: false in the Katalog\n"+
							"- Redeploy Orkestra, then delete the CRD.\n\n",
						req.Name,
					),
					Code: 403,
				},
			})
			return
		}
		// CRD not managed by us — allow
		h.protectionStats.RecordAllowed()
		h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	// Non-CRD Orkestra resource (deployment, service, ingress).
	// The webhook ObjectSelector already filtered this to resources carrying
	// the Orkestra labels — if we got here, it is ours and we always block.
	logger.Info().
		Str("resource", req.Resource.Resource).
		Str("name", req.Name).
		Str("namespace", req.Namespace).
		Str("uid", req.UID).
		Msg("deletion-protection: blocking Orkestra resource deletion")

	metrics.RecordDeletionProtectionBlocked("orkestra-" + req.Resource.Resource)
	h.protectionStats.RecordBlocked()

	h.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
		UID:     req.UID,
		Allowed: false,
		Status: &AdmissionStatus{
			Message: fmt.Sprintf(
				"\n\n[Orkestra Security] The Orkestra %s %q is protected from deletion.\n\n"+
					"To disable:\n"+
					"- Set security.deletionProtection.enabled: false in the Katalog first.\n"+
					"- Redeploy Orkestra, then delete the resource.\n\n",
				req.Resource.Resource, req.Name,
			),
			Code: 403,
		},
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
