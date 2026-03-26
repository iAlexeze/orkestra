package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ domain.Komponent = (*HealthServer)(nil)

type HealthServer struct {
	server *http.Server
	mux    *http.ServeMux

	// optional conversion HTTPS server
	convServer *http.Server
	convMux    *http.ServeMux

	// startup probes
	started atomic.Bool
	healthy atomic.Bool
	ready   atomic.Bool

	port      string
	client    string
	logLevel  string
	startTime time.Time

	// conversion options
	convOpts ConversionOptions

	// katalog for conversion rules
	katalog katalog.ConversionRegistry
}

type ConversionOptions struct {
	ConvEnabled bool
	ConvCert    string
	ConvKey     string
}

// NewHealthServer creates a new health server.
func NewHealthServer(client, port, logLevel string, katalog katalog.ConversionRegistry, convOpts ConversionOptions) *HealthServer {
	if client == "" {
		client = "service"
	}

	hs := &HealthServer{
		client:   client,
		port:     port,
		convOpts: convOpts,
		mux:      http.NewServeMux(),
		convMux:  http.NewServeMux(),
		logLevel: logLevel,
		katalog:  katalog,
	}

	hs.ready.Store(false)
	hs.started.Store(false)
	hs.healthy.Store(false)
	return hs
}

func (h *HealthServer) EnableConversion(certFile, keyFile string) {
	h.convOpts.ConvEnabled = true
	h.convOpts.ConvCert = certFile
	h.convOpts.ConvKey = keyFile
}

// Register adds a route to the health server mux.
// Must be called before Start() — routes registered after Start()
// are not guaranteed to be visible depending on ServeMux implementation.
func (hs *HealthServer) Register(path string, handler http.HandlerFunc) {
	hs.mux.Handle(path, hs.logRoutesMiddleware(handler))
}

func (h *HealthServer) Start(ctx context.Context) error {
	h.startTime = time.Now()
	// Validate conversion options
	if h.convOpts.ConvEnabled {
		if h.convOpts.ConvCert == "" {
			return fmt.Errorf("conversion server error: TLS_CERT is required")
		}
		if h.convOpts.ConvKey == "" {
			return fmt.Errorf("conversion server error: TLS_KEY is required")
		}
	}

	if !strings.HasPrefix(h.port, ":") {
		h.port = ":" + h.port
	}

	// health + ready + metrics on HTTP
	if strings.ToLower(h.logLevel) == "debug" {
		h.mux.Handle("/health", h.logRoutesMiddleware(http.HandlerFunc(h.healthHandler)))
		h.mux.Handle("/ready", h.logRoutesMiddleware(http.HandlerFunc(h.readyHandler)))
	} else {
		h.mux.HandleFunc("/health", h.healthHandler)
		h.mux.HandleFunc("/ready", h.readyHandler)
	}
	h.mux.Handle("/metrics", promhttp.Handler())

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

	// optional HTTPS conversion server
	if h.convOpts.ConvEnabled {
		h.convMux.HandleFunc("/convert", h.conversionHandler)

		h.convServer = &http.Server{
			Addr:    ":8443",
			Handler: h.convMux,
		}

		go func() {
			logger.Info().Msg("conversion https server listening on :8443")
			fmt.Printf("\n\nCert: %s\n", h.convOpts.ConvCert)
			fmt.Printf("Key: %s\n\n", h.convOpts.ConvKey)
			if err := h.convServer.ListenAndServeTLS(h.convOpts.ConvCert, h.convOpts.ConvKey); err != nil && err != http.ErrServerClosed {
				logger.Error().Err(err).Msg("conversion https server error")
			}
		}()
	}

	return nil
}

func (h *HealthServer) Shutdown(ctx context.Context) {
	if h.server != nil {
		if err := h.server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("http server shutdown error")
		}
	}
	if h.convServer != nil {
		if err := h.convServer.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("https conversion server shutdown error")
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
