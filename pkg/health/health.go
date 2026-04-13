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
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
)

var _ domain.Komponent = (*HealthServer)(nil)

type HealthServer struct {
	name string

	// HTTP server
	server *http.Server
	mux    *http.ServeMux

	// optional conversion HTTPS server
	hookSrv *http.Server
	hookMux *http.ServeMux

	// startup probes
	started atomic.Bool // for orkestra internal startup probes
	healthy atomic.Bool
	ready   atomic.Bool
	startup atomic.Bool // for kubernetes startup probes

	httpPort  string
	httpsPort string
	client    string
	logLevel  string
	startTime time.Time

	// webhook options
	hookKfg WebhookConfgurationOptions

	// registry for conversion rules
	conversionRegistry katalog.ConversionRegistry

	// conversion stats
	conversionStats *ConversionStats

	// admission stats (validation + mutation)
	admissionStats *AdmissionStats

	// Admission (Validation and Mutation)
	admissionRegistry katalog.AdmissionRegistry

	hookReg WebhookRegistrationOptions
	// kubeClient is used for webhook configuration registration.
	// Set via SetKubeClient after the HealthServer is constructed.
	kubeClient kubernetes.Interface

	// katalog for conditional endpoints
	katalog *katalog.Katalog

	// protectedCRDNames is the set of CRD full names (e.g. "pipelines.platform.io")
	// that the /deletion-protection handler will block from deletion.
	// Populated at startup when security.deletionProtection.enabled: true.
	// nil when deletion protection is disabled.
	protectedCRDNames  map[string]struct{}
	deletionProtection atomic.Bool
}

type WebhookConfgurationOptions struct {
	WebhooksEnabled  bool
	ConvEnabled      bool
	TLSCert          string
	TLSKey           string
	ConversionWindow int
}

const (
	httpsCtxTimeout = 30 * time.Second
)

// NewHealthServer creates a new health server.
func NewHealthServer(kubeclient kubernetes.Interface, katalog *katalog.Katalog, kfg *konfig.Konfig) *HealthServer {
	hookReg := WebhookRegistrationOptions{
		FailurePolicy:    kfg.WebhookRegistration().FailurePolicyType,
		Port:             kfg.WebhookConfig().PortInt,
		ServiceName:      kfg.WebhookRegistration().ServiceName,
		ServiceNamespace: kfg.WebhookRegistration().ServiceNamespace,
		TLSCertFile:      kfg.WebhookRegistration().TLSCert,
	}

	hookKfg := WebhookConfgurationOptions{
		WebhooksEnabled:  kfg.WebhookConfig().EnableWebhooks,
		ConvEnabled:      kfg.WebhookConfig().EnableConversion,
		TLSCert:          kfg.WebhookConfig().TLSCert,
		TLSKey:           kfg.WebhookConfig().TLSKey,
		ConversionWindow: kfg.WebhookConfig().ConversionWindow,
	}

	hs := &HealthServer{
		name:               "health server",
		kubeClient:         kubeclient,
		katalog:            katalog,
		client:             kfg.Ork().Name,
		httpPort:           kfg.Health().Port,
		httpsPort:          kfg.WebhookConfig().Port,
		hookKfg:            hookKfg,
		mux:                http.NewServeMux(),
		hookMux:            http.NewServeMux(),
		logLevel:           kfg.Ork().LogLevel,
		hookReg:            hookReg,
		conversionRegistry: katalog.ConversionRegistry(),
		admissionRegistry:  katalog.AdmissionRegistry(),
		conversionStats:    NewConversionStats(hookKfg.ConversionWindow),
		admissionStats:     NewAdmissionStats(hookKfg.ConversionWindow),
	}

	// Populate protected CRD names from Katalog — used by /deletion-protection handler
	if katalog.IsDeletionProtectionEnabled() {
		hs.protectedCRDNames = katalog.ProtectedCRDNames()
	}

	hs.ready.Store(false)
	hs.started.Store(false)
	hs.healthy.Store(false)
	hs.deletionProtection.Store(false)
	return hs
}

func (h *HealthServer) EnableConversion(certFile, keyFile string) {
	h.hookKfg.ConvEnabled = true
	h.hookKfg.TLSCert = certFile
	h.hookKfg.TLSKey = keyFile
}

func (h *HealthServer) EnableWebhooks(certFile, keyFile string) {
	h.hookKfg.WebhooksEnabled = true
	h.hookKfg.TLSCert = certFile
	h.hookKfg.TLSKey = keyFile
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
	if h.hookKfg.ConvEnabled {
		if h.hookKfg.TLSCert == "" {
			return fmt.Errorf("conversion server error: TLS_CERT is required for ENABLE_CONVERSION")
		}
		if h.hookKfg.TLSKey == "" {
			return fmt.Errorf("conversion server error: TLS_KEY is required for ENABLE_CONVERSION")
		}
	}

	if h.hookKfg.WebhooksEnabled {
		if h.hookKfg.TLSCert == "" {
			return fmt.Errorf("webhook server error: TLS_CERT is required for ENABLE_ADMISSION_WEBHOOK")
		}
		if h.hookKfg.TLSKey == "" {
			return fmt.Errorf("webhook server error: TLS_KEY is required for ENABLE_ADMISSION_WEBHOOK")
		}
	}

	if !strings.HasPrefix(h.httpPort, ":") {
		h.httpPort = ":" + h.httpPort
	}

	// health + ready + metrics on HTTP
	if strings.ToLower(h.logLevel) == "debug" {
		h.mux.Handle("/startup", h.logRoutesMiddleware(http.HandlerFunc(h.startupHandler)))
		h.mux.Handle("/health", h.logRoutesMiddleware(http.HandlerFunc(h.healthHandler)))
		h.mux.Handle("/ready", h.logRoutesMiddleware(http.HandlerFunc(h.readyHandler)))
	} else {
		h.mux.HandleFunc("/startup", h.startupHandler)
		h.mux.HandleFunc("/health", h.healthHandler)
		h.mux.HandleFunc("/ready", h.readyHandler)
	}
	h.mux.Handle("/metrics", promhttp.Handler())

	h.server = &http.Server{
		Addr:    h.httpPort,
		Handler: h.mux,
	}

	h.started.Store(true)
	h.healthy.Store(true)

	// HTTP server
	go func() {
		logger.Info().Str("port", h.httpPort).Msg("health server listening")
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Str("port", h.httpPort).Msg("health server error")
		}
	}()

	// HTTPS server — started when conversion, webhooks, or both are enabled.
	// All routes must be registered on hookMux BEFORE the server goroutine
	// starts to avoid a data race on ServeMux.
	kat := h.katalog
	admissionRuleExists := false
	startHttpsServer := false

	if h.katalog.IsDeletionProtectionEnabled() && h.katalog.DeletionProtectionGVRs() != nil {
		h.deletionProtection.Store(true)
	}

	logger.Debug().
		Bool("deletionProtection", h.deletionProtection.Load()).
		Bool("conversion", h.hookKfg.ConvEnabled).
		Bool("webhooks", h.hookKfg.WebhooksEnabled).
		Msg("health server")

	if h.hookKfg.ConvEnabled || h.hookKfg.WebhooksEnabled || h.deletionProtection.Load() {
		// Register /deletion-protection if enabled in the Katalog
		if h.deletionProtection.Load() {
			h.hookMux.HandleFunc("/deletion-protection", h.deletionProtectionHandler)
			startHttpsServer = true

			logger.Info().
				Str("addr", h.httpsPort).
				Str("endpoint", "/deletion-protection").
				Msg("deletion protection endpoint registered")
		}

		// Register /convert if conversion paths exist in the Katalog
		if kat.HasConversionPaths() {
			h.hookMux.HandleFunc("/convert", h.conversionHandler)
			startHttpsServer = true

			logger.Info().
				Str("addr", h.httpsPort).
				Str("endpoint", "/convert").
				Msg("conversion webhook endpoint registered")
		}

		// Register /validate if validation rules exist in the Katalog
		if kat.HasValidationRules() {
			h.hookMux.HandleFunc("/validate", h.validationHandler)
			admissionRuleExists = true
			startHttpsServer = true

			logger.Info().
				Str("addr", h.httpsPort).
				Str("endpoint", "/validate").
				Msg("validation endpoint registered")
		}

		// Register /mutate if mutation rules exist in the Katalog
		if kat.HasMutationRules() {
			h.hookMux.HandleFunc("/mutate", h.mutationHandler)
			admissionRuleExists = true
			startHttpsServer = true

			logger.Info().
				Str("addr", h.httpsPort).
				Str("endpoint", "/mutate").
				Msg("mutation endpoint registered")
		}

		if !strings.HasPrefix(h.httpPort, ":") {
			h.httpsPort = ":" + h.httpsPort
		}

		if startHttpsServer {
			h.hookSrv = &http.Server{
				Addr:    h.httpsPort,
				Handler: h.hookMux,
			}

			go func() {
				logger.Info().
					Str("addr", h.httpsPort).
					Str("cert_file", h.hookKfg.TLSCert).
					Str("key_file", h.hookKfg.TLSKey).
					Msg("https server listening")
				if err := h.hookSrv.ListenAndServeTLS(h.hookKfg.TLSCert, h.hookKfg.TLSKey); err != nil && err != http.ErrServerClosed {
					logger.Error().Err(err).Str("addr", h.httpsPort).Msg("https server error")
				}
			}()
		}

		// Register ValidatingWebhookConfiguration and MutatingWebhookConfiguration
		// with the API server. Best-effort — failures are logged but do not block
		// startup. Re-trigger by restarting Orkestra.
		//
		// Provision only if an admission rule actually exists
		if admissionRuleExists && h.kubeClient != nil && h.admissionRegistry != nil {
			go func() {
				wctx, cancel := context.WithTimeout(context.Background(), httpsCtxTimeout)
				defer cancel()
				if err := RegisterWebhooks(wctx, h.kubeClient, h.admissionRegistry, h.hookReg); err != nil {
					logger.Error().Err(err).
						Msg("webhook configuration registration failed — admission interception will not work. Check RBAC for admissionregistration.k8s.io")
				}
			}()
		} else {
			// log reason
			if !h.hookKfg.WebhooksEnabled {
				logger.Debug().Msg("webhook not enabled")
			}
			if !admissionRuleExists {
				logger.Debug().Msg("admission rules empty")
			}
			if h.kubeClient == nil {
				logger.Debug().Msg("kube client not set")
			}
			if h.admissionRegistry == nil {
				logger.Debug().Msg("admission registry not set")
			}
		}

		// Register deletion protection webhook if enabled
		if h.deletionProtection.Load() && h.kubeClient != nil {
			go func() {
				wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				dpGVRs := h.katalog.DeletionProtectionGVRs()
				if len(dpGVRs) == 0 {
					// Not in cluster — webhook not registered, protection is local-only
					logger.Info().Msg("deletion protection: running outside cluster — webhook not registered (ork run mode)")
					return // or continue — skip the registration block
				}

				caBundle, err := readCABundle(h.hookReg.TLSCertFile)
				if err != nil {
					logger.Error().Err(err).Msg("deletion protection webhook: cannot read CA bundle")
					return
				}

				if err := registerDeletionProtectionWebhook(wctx, h.kubeClient, dpGVRs, caBundle, h.hookReg); err != nil {
					logger.Error().Err(err).
						Msg("deletion protection webhook registration failed — CRDs are not protected")
				} else {
					logger.Info().
						Str("config", deletionProtectionWebhookConfigName).
						Int("rules", len(dpGVRs)).
						Int("protected", len(h.katalog.ProtectedCRDNames())).
						Bool("inCluster", true).
						Msg("deletion protection webhook registered")
				}
			}()
		}
	}

	// At this point:
	// - HTTP server goroutine is running (or starting)
	// - HTTPS server goroutine is running if needed
	// - health/ready flags are true
	// We now declare startup complete.
	h.SetStartupComplete()

	h.ready.Store(true)
	return nil
}

func (h *HealthServer) Shutdown(ctx context.Context) {
	// Report state
	h.ready.Store(false)
	h.healthy.Store(false)

	// Shutdown HTTP server
	if h.server != nil {
		if err := h.server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("http server shutdown error")
		}
	}

	// Shutdown HTTPS server
	if h.hookSrv != nil {
		if err := h.hookSrv.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("https conversion server shutdown error")
		}

		// Build cleanup options
		cleanupOpts := WebhookCleanupOptions{}
		if h.katalog.HasMutationRules() {
			cleanupOpts.mutating = true
		}
		if h.katalog.HasValidationRules() {
			cleanupOpts.validating = true
		}

		// Unregister Webhooks
		if err := UnregisterWebhooks(ctx, h.kubeClient, cleanupOpts); err != nil {
			logger.Error().Err(err).Msg("webhook cleanup error")
		}

		// Cleanup deletion protection webhook
		if h.katalog.IsDeletionProtectionEnabled() && h.kubeClient != nil {
			if err := cleanupValidatingWebhook(ctx, h.kubeClient, deletionProtectionWebhookConfigName); err != nil {
				logger.Error().Err(err).Msg("deletion protection webhook cleanup error")
			} else {
				logger.Info().
					Str("config", deletionProtectionWebhookConfigName).
					Msg("deletion protection webhook removed")
			}
		}
	}
}

func (h *HealthServer) Name() string {
	return h.name
}

func (h *HealthServer) StartupComplete() bool {
	return h.startup.Load()
}

func (h *HealthServer) SetStartupComplete() {
	h.startup.Store(true)
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
