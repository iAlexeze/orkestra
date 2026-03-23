package health

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ domain.Komponent = (*HealthServer)(nil)

type HealthServer struct {
	server *http.Server
	mux    *http.ServeMux

	// startup probes
	started atomic.Bool

	// health probes
	healthy atomic.Bool

	// readiness probes
	ready atomic.Bool

	port      string
	client    string
	logLevel  string
	startTime time.Time
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
	hs.started.Store(false)
	hs.healthy.Store(false)
	return hs
}

// Register adds a route to the health server mux.
// Must be called before Start() — routes registered after Start()
// are not guaranteed to be visible depending on ServeMux implementation.
func (hs *HealthServer) Register(path string, handler http.HandlerFunc) {
	hs.mux.Handle(path, hs.logRoutesMiddleware(handler))
}

func (h *HealthServer) Start(ctx context.Context) error {
	h.startTime = time.Now()
	if !strings.HasPrefix(h.port, ":") {
		h.port = ":" + h.port
	}

	// Register built-in routes on h.mux — same mux Register() uses
	if strings.ToLower(h.logLevel) == "debug" {
		h.mux.Handle("/healthz", h.logRoutesMiddleware(http.HandlerFunc(h.healthHandler)))
		h.mux.Handle("/readyz", h.logRoutesMiddleware(http.HandlerFunc(h.readyHandler)))
	} else {
		h.mux.HandleFunc("/healthz", h.healthHandler)
		h.mux.HandleFunc("/readyz", h.readyHandler)
	}
	h.mux.Handle("/metrics", promhttp.Handler())

	// h.mux now has: /healthz, /readyz, /metrics + all /katalog/* routes
	h.server = &http.Server{
		Addr:    h.port,
		Handler: h.mux,
	}

	h.started.Store(true)
	h.healthy.Store(true)
	go func() {
		logger.Info().Msgf("health server listening on %s", h.port)
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
	h.healthy.Store(false)
}

func (h *HealthServer) Name() string {
	return "health server"
}

func (h *HealthServer) SetReady() {
	h.ready.Store(true)
}

func (h *HealthServer) Degraded() {
	if h.ready.Load() {
		h.ready.Store(false)
	}
}

func (h *HealthServer) Unhealthy() {
	h.healthy.Store(false)
}

func (h *HealthServer) Started() bool {
	return h.started.Load()
}

func (h *HealthServer) SetStarted() {
	h.started.Store(true)
}

func (h *HealthServer) Healthy() bool {
	return h.healthy.Load()
}

func (h *HealthServer) Ready() bool {
	return h.ready.Load()
}

func (h *HealthServer) Uptime() string {
	if h.startTime.IsZero() {
		return "unknown"
	}
	return time.Since(h.startTime).Round(time.Second).String()
}
