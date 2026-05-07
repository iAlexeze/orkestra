package controlcenter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Rendering helpers
// ─────────────────────────────────────────────────────────────────────────────

// renderTemplate parses and executes a named template from the embedded FS.
// Writes directly to w. On error, renders a 500 page.
func (cc *ControlCenter) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	tmpl, err := template.New(name).Funcs(templateFuncs).ParseFS(Assets, TemplateDir+"/"+name)
	if err != nil {
		log.Printf("ERROR: parse %s: %v", name, err)
		cc.renderError(w, nil, fmt.Sprintf("Template error: %v", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		log.Printf("ERROR: execute %s: %v", name, err)
		cc.renderError(w, nil, fmt.Sprintf("Render error: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// renderError renders an inline error page. r is optional (may be nil).
func (cc *ControlCenter) renderError(w http.ResponseWriter, _ *http.Request, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	message = strings.ReplaceAll(message, "\n", "<br>")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Error – Orkestra Control Center</title>
<link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
<link rel="icon" type="image/png" href="/controlcenter/assets/static/logo.png">
<link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
<script>(function(){var t=localStorage.getItem('cc-theme')||'dark';document.documentElement.setAttribute('data-theme',t);})();</script>
</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg-base)">
  <div style="text-align:center;padding:40px;max-width:480px">
    <div style="font-size:36px;margin-bottom:16px;opacity:0.6">⚠</div>
    <h1 style="font-size:20px;font-weight:700;color:var(--text-primary);margin-bottom:8px">Something went wrong</h1>
    <p style="font-size:12px;font-family:monospace;color:var(--text-muted);margin-bottom:24px;word-break:break-all">%s</p>
    <a href="/controlcenter" style="display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:var(--accent);color:#fff;border-radius:6px;text-decoration:none;font-size:13px">
      ← Control Center
    </a>
  </div>
</body></html>`, message)
}

// handleNotFound renders a 404 page.
func (cc *ControlCenter) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>404 – Orkestra Control Center</title>
<link rel="icon" type="image/png" href="/controlcenter/assets/static/logo.png">
<link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
<script>(function(){var t=localStorage.getItem('cc-theme')||'dark';document.documentElement.setAttribute('data-theme',t);})();</script>
</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg-base)">
  <div style="text-align:center;padding:40px;max-width:400px">
    <div style="font-size:60px;font-weight:700;color:var(--text-muted);margin-bottom:8px">404</div>
    <h1 style="font-size:18px;font-weight:600;color:var(--text-primary);margin-bottom:8px">Page not found</h1>
    <p style="font-size:13px;color:var(--text-muted);margin-bottom:24px">The page you're looking for doesn't exist.</p>
    <a href="/controlcenter" style="display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:var(--accent);color:#fff;border-radius:6px;text-decoration:none;font-size:13px">
      ← Back to Control Center
    </a>
  </div>
</body></html>`)
}

// encodeInstance encodes an instance URL for use in URL paths.
// Strips http:// and replaces : with - for safe path embedding.
func encodeInstance(url string) string {
	s := strings.TrimPrefix(url, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.TrimSuffix(s, "/")
	return s
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		return
	}

	// Encode response as JSON
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

}

type H map[string]interface{}
