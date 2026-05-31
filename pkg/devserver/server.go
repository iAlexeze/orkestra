// Package devserver provides a lightweight mock HTTP server for local development.
// Started via `ork run --dev-server`, it listens on :9999 and serves all endpoints
// used by the external: and full-stack example katalogs — no real services needed.
package devserver

import (
	"fmt"
	"net"
	"net/http"

	"github.com/orkspace/orkestra/pkg/logger"
)

const Port = 9999

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
	fmt.Printf("  GET  /health              → 200 healthy\n")
	fmt.Printf("  GET  /status/200          → 200 ok\n")
	fmt.Printf("  GET  /status/503          → 503 unavailable\n")
	fmt.Printf("  GET  /config/:name        → JSON config blob\n")
	fmt.Printf("  POST /sign                → 200 signed (ignores token)\n")
	fmt.Printf("  POST /auth/token          → fake bearer token\n")
	fmt.Printf("  GET  /resources/:name     → JSON resource stub\n")
	fmt.Printf("  GET  /flags/:name         → feature flags JSON\n")
	fmt.Printf("──────────────────────────────────────────\n\n")
}
