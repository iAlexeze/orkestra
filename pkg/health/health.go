package health

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/config"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
)

var _ domain.Component = (*HealthServer)(nil)

type HealthServer struct {
	server *http.Server
	ready  atomic.Bool
	port   string
	client string
	cfg    *config.Config
}

func NewHealthServer(cfg *config.Config) *HealthServer {
	client := cfg.App().Name
	if client == "" {
		client = "service"
	}

	hs := &HealthServer{
		client: client,
		port:   cfg.Health().Port,
		cfg:    cfg,
	}

	// server is not ready on startup. modified when client is ready to process requests
	hs.ready.Store(false)
	return hs
}

func (h *HealthServer) Start(ctx context.Context) error {
	if !strings.HasPrefix(h.port, ":") {
		logger.Debug().Msgf("normalizing port from %s to :%s", h.port, h.port)
		h.port = ":" + h.port
	}

	h.server = &http.Server{
		Addr:    h.port,
		Handler: h.routes(),
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

func (h *HealthServer) routes() *http.ServeMux {
	mux := http.NewServeMux()

	if h.cfg.IsDev() {
		mux.Handle("/health", h.logRouteMiddleware(http.HandlerFunc(h.healthHandler)))
		mux.Handle("/ready", h.logRouteMiddleware(http.HandlerFunc(h.readyHandler)))
	} else {
		mux.HandleFunc("/health", h.healthHandler)
		mux.HandleFunc("/ready", h.readyHandler)
	}
	return mux
}
