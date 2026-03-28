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
	"k8s.io/client-go/kubernetes"
)

var _ domain.Komponent = (*HealthServer)(nil)

type HealthServer struct {
	server *http.Server
	mux    *http.ServeMux

	// optional conversion HTTPS server
	hookSrv *http.Server
	hookMux *http.ServeMux

	// startup probes
	started atomic.Bool
	healthy atomic.Bool
	ready   atomic.Bool

	port      string
	client    string
	logLevel  string
	startTime time.Time

	// webhook options
	opts WebhookOptions

	// katalog for conversion rules
	conversionRegistry katalog.ConversionRegistry

	// conversion stats
	conversionStats *ConversionStats

	// admission stats (validation + mutation)
	admissionStats *AdmissionStats

	// Admission (Validation and Mutation)
	admissionRegistry katalog.AdmissionRegistry

	webhookOpts WebhookRegistrationOptions
	// kubeClient is used for webhook configuration registration.
	// Set via SetKubeClient after the HealthServer is constructed.
	kubeClient kubernetes.Interface
}

type WebhookOptions struct {
	WebhooksEnabled  bool
	ConvEnabled      bool
	TLSCert          string
	TLSKey           string
	ConversionWindow int
}

// NewHealthServer creates a new health server.
func NewHealthServer(
	kubeclient kubernetes.Interface,
	client, port, logLevel string,
	conversionRegistry katalog.ConversionRegistry,
	admissionRegistry katalog.AdmissionRegistry,
	opts WebhookOptions,
) *HealthServer {
	if client == "" {
		client = "service"
	}

	hs := &HealthServer{
		kubeClient:         kubeclient,
		client:             client,
		port:               port,
		opts:               opts,
		mux:                http.NewServeMux(),
		hookMux:            http.NewServeMux(),
		logLevel:           logLevel,
		conversionRegistry: conversionRegistry,
		admissionRegistry:  admissionRegistry,
		conversionStats:    NewConversionStats(opts.ConversionWindow),
		admissionStats:     NewAdmissionStats(opts.ConversionWindow),
	}

	hs.ready.Store(false)
	hs.started.Store(false)
	hs.healthy.Store(false)
	return hs
}

func (h *HealthServer) EnableConversion(certFile, keyFile string) {
	h.opts.ConvEnabled = true
	h.opts.TLSCert = certFile
	h.opts.TLSKey = keyFile
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
	if h.opts.ConvEnabled {
		if h.opts.TLSCert == "" {
			return fmt.Errorf("conversion server error: TLS_CERT is required for ENABLE_CONVERSION")
		}
		if h.opts.TLSKey == "" {
			return fmt.Errorf("conversion server error: TLS_KEY is required for ENABLE_CONVERSION")
		}
	}

	if h.opts.WebhooksEnabled {
		if h.opts.TLSCert == "" {
			return fmt.Errorf("webhook server error: TLS_CERT is required for ENABLE_WEBHOOKS")
		}
		if h.opts.TLSKey == "" {
			return fmt.Errorf("webhook server error: TLS_KEY is required for ENABLE_WEBHOOKS")
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
		logger.Info().Str("port", h.port).Msg("health server listening")
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Str("port", h.port).Msg("health server error")
		}
	}()

	// HTTPS server — started when conversion, webhooks, or both are enabled.
	// All routes must be registered on hookMux BEFORE the server goroutine
	// starts to avoid a data race on ServeMux.
	if h.opts.ConvEnabled || h.opts.WebhooksEnabled {
		if h.opts.ConvEnabled {
			h.hookMux.HandleFunc("/convert", h.conversionHandler)
		}

		if h.opts.WebhooksEnabled {
			h.hookMux.HandleFunc("/validate", h.validationHandler)
			h.hookMux.HandleFunc("/mutate", h.mutationHandler)
			logger.Info().
				Str("addr", ":8443").
				Strs("endpoints", []string{"/validate", "/mutate"}).
				Msg("admission webhook endpoints registered")
		}

		h.hookSrv = &http.Server{
			Addr:    ":8443",
			Handler: h.hookMux,
		}

		go func() {
			logger.Info().
				Str("addr", ":8443").
				Str("cert_file", h.opts.TLSCert).
				Str("key_file", h.opts.TLSKey).
				Msg("https server listening")
			if err := h.hookSrv.ListenAndServeTLS(h.opts.TLSCert, h.opts.TLSKey); err != nil && err != http.ErrServerClosed {
				logger.Error().Err(err).Str("addr", ":8443").Msg("https server error")
			}
		}()

		// Register ValidatingWebhookConfiguration and MutatingWebhookConfiguration
		// with the API server. Best-effort — failures are logged but do not block
		// startup. Re-trigger by restarting Orkestra.
		if h.opts.WebhooksEnabled && h.kubeClient != nil && h.admissionRegistry != nil {
			go func() {
				wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := RegisterWebhooks(wctx, h.kubeClient, h.admissionRegistry, h.webhookOpts); err != nil {
					logger.Error().Err(err).
						Msg("webhook configuration registration failed — admission interception will not work. Check RBAC for admissionregistration.k8s.io")
				}
			}()
		}
	}

	return nil
}

func (h *HealthServer) Shutdown(ctx context.Context) {
	if h.server != nil {
		if err := h.server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("http server shutdown error")
		}
	}
	if h.hookSrv != nil {
		if err := h.hookSrv.Shutdown(ctx); err != nil {
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

// GetConversionStats returns the conversion statistics for use in handlers.
func (h *HealthServer) GetConversionStats() *ConversionStats {
	return h.conversionStats
}

// GetAdmissionStats returns the admission statistics for use in handlers.
func (h *HealthServer) GetAdmissionStats() *AdmissionStats {
	return h.admissionStats
}
