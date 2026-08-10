package intake

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/orkspace/orkestra/pkg/gateway/api"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// maxIntakeBodyBytes caps every intake request body — the same 1 MiB limit
// POST /api/v1/apply itself uses.
const maxIntakeBodyBytes = 1 << 20

// NewGenericHandler returns the http.HandlerFunc for one generic webhook
// entry. Any caller that can send a JSON body and an HMAC-SHA256 signature —
// PagerDuty, Datadog, an internal system — reaches the same
// ApplyTargetFields pipeline a direct POST /api/v1/apply call does.
func NewGenericHandler(
	src ResolvedGenericSource,
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "only POST requests are supported")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxIntakeBodyBytes))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request", "failed to read request body")
			return
		}

		if !verifyHMACSHA256(src.Secret, body, r.Header.Get("X-Signature-256")) {
			logger.FromContext(r.Context()).Warn().
				Str("entry", src.Config.Name).
				Msg("intake: generic webhook signature verification failed")
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid signature")
			return
		}

		var fields map[string]interface{}
		if err := json.Unmarshal(body, &fields); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request", "body must be a JSON object")
			return
		}

		dryRun := r.URL.Query().Get("dryRun") == "true"
		resp, status := api.ApplyTargetFields(
			r.Context(), kube, kat, notes, src.Config.Name, fields, dryRun,
		)
		writeJSON(w, status, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, errMsg, message string) {
	writeJSON(w, status, map[string]string{"error": errMsg, "message": message})
}
