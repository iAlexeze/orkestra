// pkg/gateway/applyapi/server.go
//
// ApplyAPIServer wires the Apply API handlers onto a health.HealthServer.
// Called from cmd/internal/gateway.go after the kubeclient is ready.
//
// Route layout:
//
//	POST   /api/v1/apply                          → applyHandler
//	GET    /api/v1/resources/{kind}/{ns}[/{name}] → resourcesHandler
//	DELETE /api/v1/resources/{kind}/{ns}/{name}   → resourcesHandler
//	GET    /api/v1/schema/                        → schemaHandler (service catalog — IDP-enabled CRDs)
//	GET    /api/v1/schema/{kind}                  → schemaHandler (CRD schema + idpFields)
//
// All routes are wrapped by AuthMiddleware before registration.
package applyapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"

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
	// auth closes over s so that token lookups always read the current TokenSet
	// even after a ReloadTokens call swaps the pointer.
	auth := func(h http.Handler) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			name := s.matches(strings.TrimSpace(bearer))
			if name == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r.WithContext(contextWithTokenName(r.Context(), name)))
		})
	}

	// POST /api/v1/apply
	reg.Register("/api/v1/apply", auth(applyHandler(s.kube, s.buildCRDLookup())))

	// GET/DELETE /api/v1/resources/...
	reg.Register("/api/v1/resources/", auth(resourcesHandler(s.kube, s.buildKindMapper())))

	// GET /api/v1/schema/...
	reg.Register("/api/v1/schema/", auth(schemaHandler(s.kube, s.buildCRDLookup(), s.buildCatalogLister())))

	logger.Info().Msg("apply API routes registered: /api/v1/apply, /api/v1/resources/, /api/v1/schema/")
}

// buildKindMapper returns a KindMapper that resolves kind/plural names to GVRs
// from the Katalog's enabled CRDs.
func (s *ApplyAPIServer) buildKindMapper() KindMapper {
	index := make(map[string]schema.GroupVersionResource)
	if s.kat == nil {
		return func(kind string) (schema.GroupVersionResource, error) {
			return schema.GroupVersionResource{}, fmt.Errorf("no CRD registered for kind %q", kind)
		}
	}
	for _, crd := range s.kat.Enabled() {
		key := strings.ToLower(crd.APITypes.Kind)
		index[key] = crd.GVR()
		// Also index by plural for callers that use the resource name.
		index[strings.ToLower(crd.APITypes.Plural)] = crd.GVR()
	}
	return func(kind string) (schema.GroupVersionResource, error) {
		gvr, ok := index[strings.ToLower(kind)]
		if !ok {
			return schema.GroupVersionResource{}, fmt.Errorf("no CRD registered for kind %q", kind)
		}
		return gvr, nil
	}
}

// buildCatalogLister returns a CatalogLister of all IDP-enabled CRDEntries.
func (s *ApplyAPIServer) buildCatalogLister() CatalogLister {
	if s.kat == nil {
		return func() []*orktypes.CRDEntry { return nil }
	}
	var entries []*orktypes.CRDEntry
	for i := range s.kat.Enabled() {
		crd := s.kat.Enabled()[i]
		if !crd.IDPEnabled() {
			continue
		}
		crdCopy := crd
		entries = append(entries, &crdCopy)
	}
	return func() []*orktypes.CRDEntry { return entries }
}

// buildCRDLookup returns a CRDLookup that finds a CRDEntry by kind name.
// Only returns entries where idp.enabled: true.
func (s *ApplyAPIServer) buildCRDLookup() CRDLookup {
	index := make(map[string]*orktypes.CRDEntry)
	if s.kat == nil {
		return func(kind string) *orktypes.CRDEntry { return nil }
	}
	for i := range s.kat.Enabled() {
		crd := s.kat.Enabled()[i]
		if !crd.IDPEnabled() {
			continue
		}
		key := strings.ToLower(crd.APITypes.Kind)
		crdCopy := crd // avoid loop-variable capture
		index[key] = &crdCopy
	}
	return func(kind string) *orktypes.CRDEntry {
		return index[strings.ToLower(kind)]
	}
}
