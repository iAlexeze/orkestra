// webhook/deletion_protection.go — /deletion-protection webhook handler.
//
// Registered only when security.deletionProtection.enabled: true in the Katalog.
// Intercepts DELETE operations on CRDs owned by this operator and Orkestra's
// own resources (deployment, service, ingress, webhook configurations).
//
// failurePolicy: Fail — if Orkestra is unreachable, DELETE is blocked.
// To decommission: set deletionProtection.enabled: false, redeploy, then delete.
package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
)

func (ws *WebhookServer) deletionProtectionHandler(w http.ResponseWriter, r *http.Request) {
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

	if req.Operation != "DELETE" {
		ws.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	// Self-protection: block deletion of the deletion-protection webhook itself.
	// First test towards sel-protection, didn't work.
	// Now uses the housekeeper
	if req.Kind.Kind == "ValidatingWebhookConfiguration" &&
		req.Name == deletionProtectionWebhookConfigName {

		logger.Info().
			Str("webhook", req.Name).
			Str("uid", req.UID).
			Msg("deletion-protection: blocking deletion of the deletion-protection webhook")

		metrics.RecordDeletionProtectionBlocked(deletionProtectionWebhookConfigName)
		ws.protectionStats.RecordBlocked()

		ws.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID:     req.UID,
			Allowed: false,
			Status: &AdmissionStatus{
				Message: fmt.Sprintf(
					"\n\n[Orkestra Security] The deletion-protection webhook \"%s\" is itself protected.\n\n"+
						"To disable deletion protection entirely:\n"+
						"- Set security.deletionProtection.enabled: false in the Katalog\n"+
						"- Redeploy Orkestra, then delete the webhook.\n\n",
					deletionProtectionWebhookConfigName,
				),
				Code: 403,
			},
		})
		return
	}

	isCRD := req.Resource.Group == "apiextensions.k8s.io" &&
		req.Resource.Resource == "customresourcedefinitions"

	if isCRD {
		if ws.isProtectedCRD(req.Name) {
			logger.Info().
				Str("crd", req.Name).
				Str("uid", req.UID).
				Msg("deletion-protection: blocking CRD deletion")

			metrics.RecordDeletionProtectionBlocked(req.Name)
			ws.protectionStats.RecordBlocked()
			_ = time.Since(start)

			ws.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
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
		ws.protectionStats.RecordAllowed()
		ws.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
			UID: req.UID, Allowed: true,
		})
		return
	}

	// Non-CRD Orkestra resource — always block (ObjectSelector already filtered).
	logger.Info().
		Str("resource", req.Resource.Resource).
		Str("name", req.Name).
		Str("namespace", req.Namespace).
		Str("uid", req.UID).
		Msg("deletion-protection: blocking Orkestra resource deletion")

	metrics.RecordDeletionProtectionBlocked("orkestra-" + req.Resource.Resource)
	ws.protectionStats.RecordBlocked()

	ws.writeAdmissionResponse(w, review.APIVersion, review.Kind, &AdmissionResponse{
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

func (ws *WebhookServer) isProtectedCRD(crdName string) bool {
	if ws.protectedCRDNames == nil {
		return false
	}
	_, ok := ws.protectedCRDNames[crdName]
	return ok
}
