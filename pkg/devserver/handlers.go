package devserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
)

func registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/health", handle(healthHandler))
	mux.HandleFunc("/status/", handle(statusHandler))
	mux.HandleFunc("/config/", handle(configHandler))
	mux.HandleFunc("/sign", handle(signHandler))
	mux.HandleFunc("/auth/token", handle(authTokenHandler))
	mux.HandleFunc("/resources/", handle(resourcesHandler))
	mux.HandleFunc("/flags/", handle(flagsHandler))
}

// handle wraps a handler with debug logging.
func handle(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
		logger.Debug().Str("method", r.Method).Str("path", r.URL.Path).Msg("dev server request")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// GET /health → 200
func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /status/200 → 200, GET /status/503 → 503
func statusHandler(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/status/")
	switch code {
	case "200":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case "503":
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown status code"})
	}
}

// GET /config/:name → JSON config blob
func configHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/config/")
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     name,
		"env":      "production",
		"debug":    "false",
		"replicas": 2,
	})
}

// POST /sign → 200 signed, or 403 if the image is nginx:not-secure.
// Simulates a signing policy that rejects images flagged as insecure.
func signHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Image string `json:"image"`
	}
	json.NewDecoder(r.Body).Decode(&payload) //nolint:errcheck

	if payload.Image == "nginx:not-secure" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":  "image rejected by signing policy",
			"image":  payload.Image,
			"reason": "image is flagged as insecure — update spec.image to a trusted tag",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"signed": true})
}

// POST /auth/token → plain-string fake bearer token
func authTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("dev-token-abc123")) //nolint:errcheck
}

// GET /resources/:name → JSON resource stub
func resourcesHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/resources/")
	writeJSON(w, http.StatusOK, map[string]any{
		"name":   name,
		"status": "active",
		"ready":  true,
	})
}

// GET /flags/:name → feature flags JSON
func flagsHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/flags/")
	writeJSON(w, http.StatusOK, map[string]any{
		"name":        name,
		"v2Enabled":   true,
		"betaEnabled": false,
	})
}
