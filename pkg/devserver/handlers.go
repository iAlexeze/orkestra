package devserver

import (
	"encoding/json"
	"fmt"
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

func writePlain(w http.ResponseWriter, status int, s string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	fmt.Fprint(w, s) //nolint:errcheck
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
	writePlain(w, http.StatusOK, "dev-token-abc123")
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

// flagsHandler routes /flags/* requests:
//
//	GET  /flags/:name              → all flags as JSON
//	GET  /flags/:name/:flag        → single flag value as plain text ("true"/"false")
//	POST /flags/:name/:flag/toggle → flip the flag, return new value as plain text
func flagsHandler(w http.ResponseWriter, r *http.Request) {
	// Strip the /flags/ prefix and split remaining path segments.
	rest := strings.TrimPrefix(r.URL.Path, "/flags/")
	rest = strings.TrimSuffix(rest, "/")
	parts := strings.SplitN(rest, "/", 3)

	switch {
	// GET /flags/:name
	case len(parts) == 1:
		name := parts[0]
		writeJSON(w, http.StatusOK, map[string]any{
			"name":        name,
			"v2Enabled":   flagGet(name, "v2Enabled"),
			"betaEnabled": flagGet(name, "betaEnabled"),
		})

	// GET /flags/:name/:flag
	case len(parts) == 2 && r.Method == http.MethodGet:
		name, flag := parts[0], parts[1]
		writePlain(w, http.StatusOK, fmt.Sprintf("%v", flagGet(name, flag)))

	// POST /flags/:name/:flag/toggle
	case len(parts) == 3 && parts[2] == "toggle" && r.Method == http.MethodPost:
		name, flag := parts[0], parts[1]
		next := flagToggle(name, flag)
		writePlain(w, http.StatusOK, fmt.Sprintf("%v", next))

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown flags route"})
	}
}
