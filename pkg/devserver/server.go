// Package devserver provides a lightweight mock HTTP server for local development.
// Started via `ork run --dev-server`, it listens on :9999 and serves all endpoints
// used by the external: and full-stack example katalogs — no real services needed.
package devserver

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/orkspace/orkestra/pkg/logger"
)

const Port = 9999

// flagDefaults holds the out-of-box value for each known flag name.
// A flag not in this map defaults to "false".
var flagDefaults = map[string]bool{
	"v2Enabled":   true,
	"betaEnabled": false,
}

// flagState holds per-app per-flag overrides set via the toggle endpoint.
// key: "appName/flagName", value: current bool state.
var flagState sync.Map

// certState holds per-cert issued/pending overrides set via the toggle endpoint.
// key: cert name, value: true = issued, false = pending. Default is issued.
var certState sync.Map

// certGet returns whether the cert is currently issued (true) or pending (false).
func certGet(name string) bool {
	if v, ok := certState.Load(name); ok {
		return v.(bool)
	}
	return true // default: issued
}

// certToggle flips the cert between issued and pending, returns the new state.
func certToggle(name string) bool {
	next := !certGet(name)
	certState.Store(name, next)
	return next
}

// flagGet returns the current bool value for a flag, preferring any toggle
// override over the hardcoded default.
func flagGet(app, flag string) bool {
	key := app + "/" + flag
	if v, ok := flagState.Load(key); ok {
		return v.(bool)
	}
	if def, ok := flagDefaults[flag]; ok {
		return def
	}
	return false
}

// flagToggle flips the current value for a flag and returns the new state.
func flagToggle(app, flag string) bool {
	next := !flagGet(app, flag)
	flagState.Store(app+"/"+flag, next)
	return next
}

// Start binds the dev server on the given port and returns immediately.
// The server runs in its own goroutine for the lifetime of the process.
func Start(port int) error {
	mux := http.NewServeMux()
	registerHandlers(mux)

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("dev server: bind %s: %w", addr, err)
	}

	printDevBanner(port)

	go func() {
		if err := http.Serve(ln, mux); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("dev server error")
		}
	}()

	return nil
}

func printDevBanner(port int) {
	fmt.Printf("\n──────────────────────────────────────────\n")
	fmt.Printf("  Dev Server running on :%d\n", port)
	fmt.Printf("──────────────────────────────────────────\n")
	fmt.Printf("  GET  /health                    → 200 healthy\n")
	fmt.Printf("  GET  /status/200                → 200 ok\n")
	fmt.Printf("  GET  /status/503                → 503 unavailable\n")
	fmt.Printf("  GET  /config/:name              → JSON config blob\n")
	fmt.Printf("  POST /sign                      → 200 signed (ignores token)\n")
	fmt.Printf("  POST /auth/token                → fake bearer token\n")
	fmt.Printf("  GET  /resources/:name           → JSON resource stub\n")
	fmt.Printf("  GET  /flags/:name               → all flags JSON\n")
	fmt.Printf("  GET  /flags/:name/:flag         → single flag value (plain text)\n")
	fmt.Printf("  POST /flags/:name/:flag/toggle  → flip flag, returns new value\n")
	fmt.Printf("  GET  /sbom/:image               → SBOM report (nginx:vulnerable → high CVEs, nginx:unknown → 404)\n")
	fmt.Printf("  POST /cosign/verify             → 200 verified (nginx:unsigned → 403)\n")
	fmt.Printf("  GET  /vault/v1/secret/data/:path → Vault KV v2 (expired/missing in path → 403/404)\n")
	fmt.Printf("  POST /v1/data/:policy           → OPA decision (namespace=forbidden or name contains 'deny' → deny)\n")
	fmt.Printf("  GET  /certs/:name/status        → cert status: issued (200) or pending (202)\n")
	fmt.Printf("  POST /certs/:name/toggle        → flip cert between issued and pending\n")
	fmt.Printf("──────────────────────────────────────────\n\n")
}
