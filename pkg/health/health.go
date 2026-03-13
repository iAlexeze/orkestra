package health

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ domain.Komponent = (*HealthServer)(nil)

type HealthServer struct {
	server   *http.Server
	mux      *http.ServeMux
	ready    atomic.Bool
	port     string
	client   string
	logLevel string
}

func NewHealthServer(client, port, logLevel string) *HealthServer {
	if client == "" {
		client = "service"
	}

	hs := &HealthServer{
		client:   client,
		port:     port,
		mux:      http.NewServeMux(),
		logLevel: logLevel,
	}

	// server is not healthy or ready on startup.
	// modified when client is ready to process requests
	hs.ready.Store(false)
	return hs
}

// Register adds a route to the health server mux.
// Must be called before Start() — routes registered after Start()
// are not guaranteed to be visible depending on ServeMux implementation.
func (hs *HealthServer) Register(path string, handler http.HandlerFunc) {
	hs.mux.Handle(path, hs.logRoutesMiddleware(handler))
}

func (h *HealthServer) Start(ctx context.Context) error {
	if !strings.HasPrefix(h.port, ":") {
		h.port = ":" + h.port
	}

	// Register built-in routes on h.mux — same mux Register() uses
	if strings.ToLower(h.logLevel) == "debug" {
		h.mux.Handle("/health", h.logRoutesMiddleware(http.HandlerFunc(h.healthHandler)))
		h.mux.Handle("/ready", h.logRoutesMiddleware(http.HandlerFunc(h.readyHandler)))
	} else {
		h.mux.HandleFunc("/health", h.healthHandler)
		h.mux.HandleFunc("/ready", h.readyHandler)
	}
	h.mux.Handle("/metrics", promhttp.Handler())

	// h.mux now has: /health, /ready, /metrics + all /katalog/* routes
	h.server = &http.Server{
		Addr:    h.port,
		Handler: h.mux,
	}

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("health server error")
		}
	}()

	return nil
}

func (h *HealthServer) Shutdown(ctx context.Context) {
	if h.server != nil {
		if err := h.server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("health server shutdown error")
		}
	}
	h.ready.Store(false)
}

func (h *HealthServer) Name() string {
	return "health server"
}

func (h *HealthServer) SetReady() {
	h.ready.Store(true)
}

func (h *HealthServer) Started() bool {
	return h.ready.Load()
}

func (h *HealthServer) Degraded() bool {
	if h.ready.Load() {
		h.ready.Store(false)
	}
	return false
}
