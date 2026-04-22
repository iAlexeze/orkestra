package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
)

// HealthServer is Orkestra’s runtime-facing health and admission surface.
// It owns all HTTP/HTTPS endpoints (health, readiness, metrics, conversion,
// validation, mutation, deletion protection) and acts as the lifecycle anchor
// for webhook registration, reconciliation, and shutdown semantics.
//
// The HealthServer is intentionally minimal in responsibilities:
//   - Serve health/readiness for Kubernetes
//   - Expose admission and conversion endpoints when declared in the Katalog
//   - Register and reconcile webhook configurations with the API server
//   - Track runtime state (started, ready, healthy, startup-complete)
//   - Maintain stats for conversion, admission, and deletion protection
//
// All admission behavior is declarative — the Katalog is the single source of truth.
// The HealthServer simply reflects that declaration into runtime behavior.
var _ domain.Komponent = (*HealthServer)(nil)

type HealthServer struct {
	name string

	// HTTP server for health, readiness, and metrics.
	server *http.Server
	mux    *http.ServeMux

	// Optional HTTPS server for conversion + admission webhooks.
	hookSrv *http.Server
	hookMux *http.ServeMux

	// Runtime state flags used by Kubernetes probes and internal readiness gates.
	started atomic.Bool // internal startup indicator
	healthy atomic.Bool
	ready   atomic.Bool
	startup atomic.Bool // Kubernetes startupProbe indicator

	httpPort  string
	httpsPort string
	client    string
	logLevel  string
	startTime time.Time

	// Declarative webhook configuration resolved from Katalog + ENV.
	hookKfg WebhookConfgurationOptions

	// Registry of conversion rules declared in the Katalog.
	conversionRegistry katalog.ConversionRegistry

	// Rolling conversion statistics for observability.
	conversionStats *ConversionStats

	// Rolling admission statistics (validation + mutation).
	admissionStats *AdmissionStats

	// Rolling deletion protection statistics.
	protectionStats *DeletionProtectionStats

	// Webhook controller reconcilliation stats for observability
	webhookStats *WebhookStats

	// Admission registry containing validation and mutation rules.
	admissionRegistry katalog.AdmissionRegistry

	// Options used when registering webhook configurations with the API server.
	hookReg WebhookRegistrationOptions

	// Kubernetes client used to create/update/delete webhook configurations.
	// Set via SetKubeClient after construction.
	kubeClient kubernetes.Interface

	// Katalog drives all conditional behavior (conversion, admission, protection).
	katalog *katalog.Katalog

	// Set of CRD full names protected by deletion protection.
	// Populated only when deletion protection is enabled.
	protectedCRDNames  map[string]struct{}
	deletionProtection atomic.Bool

	// Namespace protection — populated only when namespace protection is enabled.
	namespaceRuleMap    map[string]*NamespaceRules
	namespaceStats      *NamespaceProtectionStats
	namespaceProtection atomic.Bool

	// Full Konfig object for accessing cluster, security, and runtime settings.
	konfig *konfig.Konfig
}

// WebhookConfgurationOptions captures the declarative enablement state for all
// webhook-related capabilities (admission, conversion) as resolved from the
// Katalog and environment. These options determine whether the HealthServer
// exposes HTTPS endpoints and participates in webhook registration.
//
// This struct does *not* describe the webhook spec itself — only whether the
// runtime should activate the corresponding admission surfaces.
type WebhookConfgurationOptions struct {
	WebhooksEnabled  bool   // admission (validation + mutation) enabled
	ConvEnabled      bool   // CRD conversion webhook enabled
	TLSCert          string // certificate used by the HTTPS server
	TLSKey           string // private key used by the HTTPS server
	ConversionWindow int    // rolling window for conversion statistics
}

// httpsCtxTimeout defines the maximum duration allowed for webhook registration
// calls to the Kubernetes API server. This prevents startup from hanging when
// the API server is unreachable or slow.
const (
	httpsCtxTimeout = 30 * time.Second
)

// NewHealthServer constructs the HealthServer and resolves all declarative
// runtime behavior from the Katalog and Konfig. This method performs no I/O;
// it simply materializes the runtime model that Start() will activate.
//
// Responsibilities:
//   - Resolve webhook enablement and failure‑policy precedence (YAML > ENV > defaults)
//   - Initialize all HTTP/HTTPS muxes and stats collectors
//   - Capture conversion/admission registries from the Katalog
//   - Precompute deletion‑protection CRD sets
//   - Initialize all readiness/health/startup flags
//
// The HealthServer returned here is inert — Start() is responsible for
// activating servers, endpoints, and webhook reconciliation.
func NewHealthServer(kubeclient kubernetes.Interface, katalog *katalog.Katalog, kfg *konfig.Konfig) *HealthServer {
	// Resolve admission webhook failure policy: YAML > ENV > default "Ignore".
	// katalog.WebhooksFailurePolicy() already applies this precedence.
	admissionFailurePolicy := admissionv1FailurePolicyType(katalog.WebhooksFailurePolicy())

	// Static registration options used when creating webhook configurations.
	hookReg := WebhookRegistrationOptions{
		FailurePolicy:            admissionFailurePolicy,
		Port:                     kfg.HTTPSPortInt32(),
		ServiceName:              katalog.WebhooksServiceName(),
		ServiceNamespace:         kfg.Cluster().Namespace,
		TLSCertFile:              kfg.Security().Webhooks.TLSCert,
		OrkestraResourceLabels:   kfg.OrkestraResourceLabels(),
		OrkestraResourceSelector: kfg.OrkestraResourceSelector(),
	}

	// Declarative enablement flags and TLS settings resolved from Katalog + ENV.
	hookKfg := WebhookConfgurationOptions{
		WebhooksEnabled:  katalog.IsAdmissionEnabled(),
		ConvEnabled:      katalog.IsConversionEnabled(),
		TLSCert:          kfg.Security().Webhooks.TLSCert,
		TLSKey:           kfg.Security().Webhooks.TLSKey,
		ConversionWindow: katalog.ConversionWindow(),
	}

	// Construct the HealthServer with all registries, muxes, and stats collectors.
	hs := &HealthServer{
		name:               "health server",
		kubeClient:         kubeclient,
		katalog:            katalog,
		konfig:             kfg,
		client:             kfg.Ork().Name,
		httpPort:           kfg.Health().Port,
		httpsPort:          kfg.HTTPSPort(),
		hookKfg:            hookKfg,
		mux:                http.NewServeMux(),
		hookMux:            http.NewServeMux(),
		logLevel:           kfg.Ork().LogLevel,
		hookReg:            hookReg,
		conversionRegistry: katalog.ConversionRegistry(),
		admissionRegistry:  katalog.AdmissionRegistry(),
		conversionStats:    NewConversionStats(hookKfg.ConversionWindow),
		admissionStats:     NewAdmissionStats(hookKfg.ConversionWindow),
		protectionStats:    NewDeletionProtectionStats(),
	}

	// Precompute protected CRD names for deletion‑protection enforcement.
	if katalog.IsDeletionProtectionEnabled() {
		hs.protectedCRDNames = katalog.DeletionProtectedCRDNames()
	}

	// Precompute namespace rule map for namespace protection enforcement.
	if katalog.IsNamespaceProtectionEnabled() {
		rawRules := katalog.NamespaceProtectionRuleMap()
		if len(rawRules) > 0 {
			hs.namespaceRuleMap = make(map[string]*NamespaceRules, len(rawRules))
			for key, entry := range rawRules {
				rules := &NamespaceRules{
					Allowed:    make(map[string]struct{}, len(entry.Allowed)),
					Restricted: make(map[string]struct{}, len(entry.Restricted)),
				}
				for _, ns := range entry.Allowed {
					rules.Allowed[ns] = struct{}{}
				}
				for _, ns := range entry.Restricted {
					rules.Restricted[ns] = struct{}{}
				}
				hs.namespaceRuleMap[key] = rules
			}
		}
		hs.namespaceStats = NewNamespaceProtectionStats()
	}

	// Initialize all runtime state flags.
	hs.ready.Store(false)
	hs.started.Store(false)
	hs.healthy.Store(false)
	hs.deletionProtection.Store(false)
	hs.namespaceProtection.Store(false)

	return hs
}

// EnableConversion activates the CRD conversion webhook at runtime.
// This is an imperative override used primarily in tests or operator-driven
// reconfiguration. It does not register the webhook — Start() handles that.
// It simply marks conversion as enabled and updates the TLS material.
func (h *HealthServer) EnableConversion(certFile, keyFile string) {
	h.hookKfg.ConvEnabled = true
	h.hookKfg.TLSCert = certFile // update certificate for HTTPS server
	h.hookKfg.TLSKey = keyFile   // update private key for HTTPS server
}

// EnableWebhooks activates admission (validation + mutation) webhooks at runtime.
// Like EnableConversion, this is an imperative override and does not perform
// registration. Start() will reflect this enablement into actual webhook
// configuration creation and endpoint exposure.
func (h *HealthServer) EnableWebhooks(certFile, keyFile string) {
	h.hookKfg.WebhooksEnabled = true
	h.hookKfg.TLSCert = certFile // update certificate for HTTPS server
	h.hookKfg.TLSKey = keyFile   // update private key for HTTPS server
}

// Register adds a route to the health server mux.
// Must be called before Start() — routes registered after Start()
// are not guaranteed to be visible depending on ServeMux implementation.
func (hs *HealthServer) Register(path string, handler http.HandlerFunc) {
	hs.mux.Handle(path, hs.logRoutesMiddleware(handler))
}

// Start activates the HealthServer’s full runtime surface. It launches the HTTP
// and HTTPS servers, exposes all declared admission/conversion/protection
// endpoints, performs best‑effort webhook registration, and begins continuous
// reconciliation of webhook configurations.
//
// This is the transition from a declarative model (NewHealthServer) to an active
// runtime. All behavior is driven by the Katalog: only capabilities explicitly
// declared (conversion, validation, mutation, deletion protection) are exposed.
// Startup is intentionally resilient — webhook registration failures never block
// readiness or liveness. Once all endpoints are registered and servers launched,
// Start() marks the runtime as startup‑complete.
func (h *HealthServer) Start(ctx context.Context) error {
	h.startTime = time.Now()

	// Validate TLS prerequisites for conversion and admission.
	// These checks prevent the HTTPS server from starting without certificates.
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

	// Normalize HTTP port format.
	if !strings.HasPrefix(h.httpPort, ":") {
		h.httpPort = ":" + h.httpPort
	}

	// Register health, readiness, and metrics endpoints.
	// Debug mode wraps handlers with request logging.
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

	// Construct the HTTP server (health + metrics).
	h.server = &http.Server{
		Addr:    h.httpPort,
		Handler: h.mux,
	}

	h.started.Store(true)
	h.healthy.Store(true)

	// Launch HTTP server asynchronously.
	go func() {
		logger.Info().Str("port", h.httpPort).Msg("health server listening")
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Str("port", h.httpPort).Msg("health server error")
		}
	}()

	// Determine whether HTTPS should be started based on declared capabilities.
	kat := h.katalog
	admissionRuleExists := false
	startHttpsServer := false

	// Enable deletion protection if declared.
	if kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil {
		h.deletionProtection.Store(true)
	}

	// Enable namespace protection if declared.
	if kat.IsNamespaceProtectionEnabled() && len(kat.NamespaceProtectionGVRs()) > 0 {
		h.namespaceProtection.Store(true)
	}

	logger.Debug().
		Bool("deletionProtection", h.deletionProtection.Load()).
		Bool("namespaceProtection", h.namespaceProtection.Load()).
		Bool("conversion", h.hookKfg.ConvEnabled).
		Bool("webhooks", h.hookKfg.WebhooksEnabled).
		Msg("health server configured")

	// ─────────────────────────────────────────────────────────────────────────
	// HTTPS & Webhook setup – only when running inside a cluster
	// ─────────────────────────────────────────────────────────────────────────
	if !utils.IsRunningInCluster() {
		logger.Debug().Msg("not running in cluster – skipping HTTPS/webhook setup")
	} else {
		// We are in cluster. Register HTTPS endpoints based on declared capabilities.
		if h.deletionProtection.Load() {
			h.hookMux.HandleFunc("/deletion-protection", h.deletionProtectionHandler)
			startHttpsServer = true
			logger.Info().Str("endpoint", "/deletion-protection").Msg("deletion protection endpoint registered")
		}

		// Namespace protection endpoint.
		if h.namespaceProtection.Load() {
			h.hookMux.HandleFunc("/namespace-protection", h.namespaceProtectionHandler)
			startHttpsServer = true
			logger.Info().Str("endpoint", "/namespace-protection").Msg("namespace protection endpoint registered")
		}

		// Conversion endpoint.
		if kat.HasConversionPaths() {
			h.hookMux.HandleFunc("/convert", h.conversionHandler)
			startHttpsServer = true
			logger.Info().Str("endpoint", "/convert").Msg("conversion webhook endpoint registered")
		}

		// Validation endpoint.
		if kat.HasValidationRules() {
			h.hookMux.HandleFunc("/validate", h.validationHandler)
			admissionRuleExists = true
			startHttpsServer = true
			logger.Info().Str("endpoint", "/validate").Msg("validation endpoint registered")
		}

		// Mutation endpoint.
		if kat.HasMutationRules() {
			h.hookMux.HandleFunc("/mutate", h.mutationHandler)
			admissionRuleExists = true
			startHttpsServer = true
			logger.Info().Str("endpoint", "/mutate").Msg("mutation endpoint registered")
		}

		// Normalize HTTPS port format.
		if !strings.HasPrefix(h.httpsPort, ":") {
			h.httpsPort = ":" + h.httpsPort
		}

		// Launch HTTPS server if any endpoint was registered.
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
					Msg("HTTPS server listening")
				if err := h.hookSrv.ListenAndServeTLS(h.hookKfg.TLSCert, h.hookKfg.TLSKey); err != nil && err != http.ErrServerClosed {
					logger.Error().Err(err).Str("addr", h.httpsPort).Msg("HTTPS server error")
				}
			}()
		}

		// Best‑effort admission webhook registration.
		if admissionRuleExists && h.kubeClient != nil && h.admissionRegistry != nil {
			go func() {
				wctx, cancel := context.WithTimeout(context.Background(), httpsCtxTimeout)
				defer cancel()
				if err := RegisterAdmissionWebhooks(wctx, h.kubeClient, h.admissionRegistry, h.hookReg); err != nil {
					logger.Error().Err(err).Msg("admission webhook registration failed – RBAC may be missing")
				}
			}()
		}

		// Deletion protection webhook registration.
		if h.deletionProtection.Load() && h.kubeClient != nil {
			go func() {
				wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				dpGVRs := kat.DeletionProtectionGVRs()
				if len(dpGVRs) == 0 {
					logger.Info().Msg("deletion protection: no GVRs found – webhook not registered")
					return
				}

				caBundle, err := readCABundle(h.hookReg.TLSCertFile)
				if err != nil {
					logger.Error().Err(err).Msg("deletion protection webhook: cannot read CA bundle")
					return
				}

				if err := registerDeletionProtectionWebhook(wctx, h.kubeClient, dpGVRs, caBundle, h.hookReg); err != nil {
					logger.Error().Err(err).Msg("deletion protection webhook registration failed – CRDs not protected")
				} else {
					logger.Info().
						Str("config", deletionProtectionWebhookConfigName).
						Int("rules", len(dpGVRs)).
						Int("protected", len(h.katalog.DeletionProtectedCRDNames())).
						Bool("inCluster", true).
						Msg("deletion protection webhook registered")
				}
			}()
		}

		// Namespace protection webhook registration.
		if h.namespaceProtection.Load() && h.kubeClient != nil {
			go func() {
				wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				npGVRs := kat.NamespaceProtectionGVRs()
				if len(npGVRs) == 0 {
					logger.Info().Msg("namespace protection: no GVRs found – webhook not registered")
					return
				}

				caBundle, err := readCABundle(h.hookReg.TLSCertFile)
				if err != nil {
					logger.Error().Err(err).Msg("namespace protection webhook: cannot read CA bundle")
					return
				}

				svcName := kat.NamespaceProtectionServiceName()
				failurePolicy := kat.NamespaceProtectionFailurePolicy()
				if err := registerNamespaceProtectionWebhook(wctx, h.kubeClient, npGVRs, caBundle, h.hookReg, svcName, failurePolicy); err != nil {
					logger.Error().Err(err).Msg("namespace protection webhook registration failed – rules not enforced")
				} else {
					logger.Info().
						Str("config", namespaceProtectionWebhookConfigName).
						Int("rules", len(npGVRs)).
						Bool("inCluster", true).
						Msg("namespace protection webhook registered")
				}
			}()
		}

		// Start webhook controller if HTTPS server is running and controller enabled.
		if startHttpsServer && kat.IsWebhookControllerEnabled() {
			if err := h.webhookController(); err != nil {
				logger.Error().Err(err).Msg("webhook controller failed to start")
			}
		}
	}

	// Declare startup complete — startupProbe becomes satisfied.
	h.SetStartupComplete()

	h.ready.Store(true)
	return nil
}

// Shutdown gracefully terminates all runtime servers and performs optional,
// declarative cleanup of webhook configurations. This method embodies Orkestra’s
// “runtime that leaves no trace” principle: cleanup is opt‑in and driven entirely
// by the Katalog’s shutdown policy. If cleanupOnShutdown is disabled, the cluster
// is left structurally unchanged.
//
// Responsibilities:
//   - Transition the runtime out of ready/healthy state
//   - Gracefully stop HTTP and HTTPS servers
//   - Optionally unregister admission webhooks (validation + mutation)
//   - Optionally remove the deletion‑protection webhook configuration
//
// Shutdown never blocks startup or reconciliation guarantees — it is best‑effort.
func (h *HealthServer) Shutdown(ctx context.Context) {
	// Immediately mark the runtime as not ready and not healthy.
	h.ready.Store(false)
	h.healthy.Store(false)

	// Shutdown HTTP server (health, ready, metrics).
	if h.server != nil {
		if err := h.server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("http server shutdown error")
		}
	}

	// Shutdown HTTPS server (conversion + admission + deletion protection).
	if h.hookSrv != nil {
		if err := h.hookSrv.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("https conversion server shutdown error")
		}

		// Build cleanup options based on declared Katalog shutdown policy.
		cleanupOpts := WebhookCleanupOptions{}
		kat := h.katalog

		// Admission cleanup is conditional and fully declarative.
		if kat.HasMutationRules() {
			cleanupOpts.mutating = kat.DeletionProtectionCleanupOnShutdown()
		}
		if kat.HasValidationRules() {
			cleanupOpts.validating = kat.DeletionProtectionCleanupOnShutdown()
		}

		// Best‑effort removal of admission webhook configurations.
		if err := UnregisterAdmissionWebhooks(ctx, h.kubeClient, cleanupOpts); err != nil {
			logger.Error().Err(err).Msg("webhook cleanup error")
		}

		// Optional cleanup of deletion‑protection webhook configuration.
		if kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionCleanupOnShutdown() && h.kubeClient != nil {
			if err := cleanupValidatingWebhook(ctx, h.kubeClient, deletionProtectionWebhookConfigName); err != nil {
				logger.Error().Err(err).Msg("deletion protection webhook cleanup error")
			} else {
				logger.Info().
					Str("config", deletionProtectionWebhookConfigName).
					Msg("deletion protection webhook removed")
			}
		}

		// Optional cleanup of namespace‑protection webhook configuration.
		if kat.IsNamespaceProtectionEnabled() && kat.NamespaceProtectionCleanupOnShutdown() && h.kubeClient != nil {
			if err := cleanupValidatingWebhook(ctx, h.kubeClient, namespaceProtectionWebhookConfigName); err != nil {
				logger.Error().Err(err).Msg("namespace protection webhook cleanup error")
			} else {
				logger.Info().
					Str("config", namespaceProtectionWebhookConfigName).
					Msg("namespace protection webhook removed")
			}
		}
	}
}

// Name returns the configured runtime name.
func (h *HealthServer) Name() string {
	return h.name
}

// StartupComplete reports whether the startup sequence has finished.
func (h *HealthServer) StartupComplete() bool {
	return h.startup.Load()
}

// SetStartupComplete marks the startup sequence as finished.
func (h *HealthServer) SetStartupComplete() {
	h.startup.Store(true)
}

// SetReady marks the server as ready to serve traffic.
func (h *HealthServer) SetReady() {
	h.ready.Store(true)
}

// Degraded transitions the server out of ready state without marking it unhealthy.
func (h *HealthServer) Degraded() {
	if h.ready.Load() {
		h.ready.Store(false)
	}
}

// Unhealthy marks the server as unhealthy, signaling a fatal condition.
func (h *HealthServer) Unhealthy() {
	h.healthy.Store(false)
}

// Started reports whether the HTTP/HTTPS servers have begun serving.
func (h *HealthServer) Started() bool {
	return h.started.Load()
}

// SetStarted marks the server as having begun serving traffic.
func (h *HealthServer) SetStarted() {
	h.started.Store(true)
}

// Healthy reports whether the server is healthy.
func (h *HealthServer) Healthy() bool {
	return h.healthy.Load()
}

// Ready reports whether the server is ready for admission and health checks.
func (h *HealthServer) Ready() bool {
	return h.ready.Load()
}

// Uptime returns human-readable uptime since the server started.
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

// GetNamespaceStats returns the namespace protection statistics for use in handlers.
func (h *HealthServer) GetNamespaceStats() *NamespaceProtectionStats {
	return h.namespaceStats
}

// GetProtectionStats returns the deletion protection statistics for use in handlers.
func (h *HealthServer) GetProtectionStats() *DeletionProtectionStats {
	return h.protectionStats
}

// GetWebhookStats returns the webhook statistics for use in handlers.
func (h *HealthServer) GetWebhookStats() *WebhookStats {
	return h.webhookStats
}
