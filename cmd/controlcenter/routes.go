package main

import (
	"fmt"
	"net/http"
	"strings"

	controlcenter "github.com/orkspace/orkestra-cc/pkg"
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
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>404 - Orkestra Control Center</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="icon" type="image/png" href="/controlcenter/assets/static/logo.png">
</head>
<body class="bg-gray-50">
    <div class="min-h-screen flex items-center justify-center px-4">
        <div class="text-center">
            <div class="text-6xl mb-4">🔍</div>
            <h1 class="text-4xl font-bold text-gray-900 mb-2">404</h1>
            <p class="text-gray-600 mb-4">Page not found</p>
            <p class="text-sm text-gray-500 mb-6">The page you're looking for doesn't exist or has been moved.</p>
            <a href="/controlcenter" class="inline-flex items-center px-4 py-2 bg-gray-900 text-white rounded-lg text-sm hover:bg-gray-800 transition">
                ← Back to Control Center
            </a>
        </div>
    </div>
</body>
</html>`)
}
