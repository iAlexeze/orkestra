// pkg/gateway/applyapi/server.go
//
// ApplyAPIServer wires the Apply API handlers onto a health.HealthServer.
// Called from cmd/internal/gateway.go after the kubeclient is ready.
//
// Route layout:
//
//			POST   /api/v1/apply                          → applyHandler
//		 	GET    /api/v1/resources/{kind}/{ns}          → resourcesHandler — list
//			GET    /api/v1/resources/{kind}/{ns}[/{name}] → resourcesHandler — get
//			DELETE /api/v1/resources/{kind}/{ns}/{name}   → resourcesHandler — delete
//			GET    /api/v1/schema/                        → schemaHandler — (service catalog — IDP-enabled CRDs)
//		 	GET    /api/v1/schema?target=<t>              → schemaHandler — schema for target
//	  	GET    /api/v1/raw-schema?kind=<k>            → schemaHandler — raw Kubernetes OpenAPI spec schema
//
// All routes are wrapped by AuthMiddleware before registration.
package applyapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Registrar is the subset of health.HealthServer used here.
// Defined as an interface so the server package does not import pkg/health.
type Registrar interface {
	Register(path string, handler http.HandlerFunc)
}

// ApplyAPIServer holds the resolved token set and wired handlers.
type ApplyAPIServer struct {
	mu     sync.RWMutex
	tokens *TokenSet
	kube   kubeclient.KubeClient
	kat    *katalog.Katalog
	ownNS  string
}

// NewApplyAPIServer resolves all tokens (bootstrapping/rotating Secrets as
// needed) and returns a ready-to-register server.
func NewApplyAPIServer(ctx context.Context, kat *katalog.Katalog, kube kubeclient.KubeClient, ownNS string) (*ApplyAPIServer, error) {
	if !kat.HasIDPEnabled() {
		return nil, nil // not enabled — caller skips registration
	}

	tokens, err := LoadTokens(ctx, kat.Gateway.ApplyAPI.Auth.Tokens, kube, ownNS)
	if err != nil {
		return nil, fmt.Errorf("loading Apply API tokens: %w", err)
	}

	return &ApplyAPIServer{
		tokens: tokens,
		kube:   kube,
		kat:    kat,
		ownNS:  ownNS,
	}, nil
}

// matches returns the token name for value, reading through the reload mutex.
func (s *ApplyAPIServer) matches(value string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens.Matches(value)
}

// ReloadTokens re-resolves all secretRef tokens (recreating any missing secrets)
// and atomically replaces the in-memory TokenSet. Called by the housekeeper when
// a token secret is found to be missing during reconciliation.
func (s *ApplyAPIServer) ReloadTokens(ctx context.Context) error {
	ts, err := LoadTokens(ctx, s.kat.Gateway.ApplyAPI.Auth.Tokens, s.kube, s.ownNS)
	if err != nil {
		return fmt.Errorf("reload apply API tokens: %w", err)
	}
	s.mu.Lock()
	s.tokens = ts
	s.mu.Unlock()
	return nil
}

// Register wires all Apply API routes onto the given Registrar.
func (s *ApplyAPIServer) Register(reg Registrar) {
	var notes orktypes.NoteRegistry
	if s.kat != nil {
		notes = s.kat.Notes
	}
	if !s.kat.HasIDPEnabled() {
		return
	}

	// auth closes over s so that token lookups always read the current TokenSet
	// even after a ReloadTokens call swaps the pointer.
	auth := func(h http.Handler) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			name := s.matches(strings.TrimSpace(bearer))
			if name == "" {
				writeJSONError(w, http.StatusUnauthorized, "Unauthorized", "invalid token")
				return
			}
			h.ServeHTTP(w, r.WithContext(contextWithTokenName(r.Context(), name)))
		})
	}

	// POST /api/v1/apply
	reg.Register("/api/v1/apply", auth(applyHandler(s.kube, s.kat, notes)))

	// GET/DELETE /api/v1/resources/...
	reg.Register("/api/v1/resources/", auth(resourcesHandler(s.kube, s.kat, notes)))

	// GET /api/v1/schema/...
	reg.Register("/api/v1/schema", auth(schemaHandler(s.kat)))

	// GET /api/v1/raw-schema
	reg.Register("/api/v1/raw-schema", auth(rawSchemaHandler(s.kube, s.kat)))

	logger.Info().Msg("apply API routes registered: /api/v1/apply, /api/v1/resources/, /api/v1/schema, /api/v1/raw-schema")
}
