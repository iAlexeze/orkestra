package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/orkspace/orkestra-cc/cc"
)

// setupRoutes configures all HTTP routes for the control center
func setupRoutes(cc *controlcenter.ControlCenter, version, commit, buildDate string) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoint - always returns OK for the control center itself
	mux.HandleFunc("/controlcenter/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"orkestra-control-center","version":"%s"}`, version)
	})

	// Readiness check endpoint - checks if at least one backend is healthy
	mux.HandleFunc("/controlcenter/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if cc.IsReady() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ready","service":"orkestra-control-center"}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not ready","service":"orkestra-control-center","reason":"no healthy backends"}`)
		}
	})

	// Version endpoint
	mux.HandleFunc("/controlcenter/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"version":"%s","commit":"%s","buildDate":"%s"}`, version, commit, buildDate)
	})

	// 404 handler
	mux.HandleFunc("/controlcenter/404", handleNotFound)

	// Control center handler for all other routes under /controlcenter
	mux.Handle("/controlcenter/", http.StripPrefix("/controlcenter", cc))

	// Root redirect
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Skip if it's a controlcenter path (already handled above)
		if strings.HasPrefix(r.URL.Path, "/controlcenter") {
			return
		}
		// Handle favicon.ico gracefully
		if r.URL.Path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/controlcenter", http.StatusMovedPermanently)
	})

	return mux
}

// handleNotFound returns a custom 404 page
func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>404 – Orkestra Control Center</title>
    <link rel="icon" type="image/png" href="/controlcenter/assets/static/logo.png">
    <link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
    <script>(function(){var t=localStorage.getItem('cc-theme')||'dark';document.documentElement.setAttribute('data-theme',t);})();</script>
</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg-base)">
    <div style="text-align:center;padding:40px;max-width:400px">
        <div style="font-size:60px;font-weight:700;color:var(--text-muted);margin-bottom:8px">404</div>
        <h1 style="font-size:18px;font-weight:600;color:var(--text-primary);margin-bottom:8px">Page not found</h1>
        <p style="font-size:13px;color:var(--text-muted);margin-bottom:6px">The page you're looking for doesn't exist or has been moved.</p>
        <a href="/controlcenter" style="display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:var(--accent);color:#fff;border-radius:6px;text-decoration:none;font-size:13px;margin-top:16px">
            ← Back to Control Center
        </a>
    </div>
</body>
</html>`)
}
