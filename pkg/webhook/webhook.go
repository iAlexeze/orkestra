// Package webhook implements Orkestra's admission and conversion webhook surface
// as a domain.Komponent that can be registered and managed alongside other
// runtime components.
//
// # Responsibility
//
// The webhook package owns everything required to operate Kubernetes admission
// and conversion webhooks:
//
//   - TLS HTTPS server that handles /validate, /mutate, /convert,
//     /deletion-protection, and /namespace-protection
//   - Webhook configuration registration and reconciliation with the API server
//   - All HTTP handlers for admission review processing
//   - Periodic controller that keeps webhook configurations in sync with the Katalog
//
// The health package (pkg/health) handles only HTTP health and readiness probes.
// This package handles only the HTTPS webhook surface.
//
// # TLS
//
// TLS certificates are provisioned by cmd/internal.ensureSecurity before Start()
// is called. The certificate file paths are read from konfig.Security().Webhooks
// after ensureSecurity writes them there. WebhookServer never generates its own
// certificates — that is the caller's responsibility.
//
// # Lifecycle
//
// WebhookServer implements domain.Komponent:
//
//	New(kubeClient, katalog, konfig)
//	  → Start(ctx)    — validate TLS, register endpoints, start HTTPS server,
//	                    register webhook configs, start controller
//	  → Shutdown(ctx) — stop HTTPS server, clean up webhook configs on shutdown
//
// All registration and reconciliation errors are logged and non-fatal. The HTTPS
// server starts only when at least one webhook capability is declared in the Katalog.
// When no capabilities are declared, Start() is a no-op.
package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
	"k8s.io/client-go/kubernetes"
)

var _ domain.Komponent = (*WebhookServer)(nil)

// certManagerIface is the subset of certmanager.Manager that WebhookServer needs
// for TLS secret cleanup on graceful shutdown.
type certManagerIface interface {
	DeleteCertificateAndSecret(ctx context.Context, namespace, secretName string) error
}

// WebhookServer is Orkestra's HTTPS admission and conversion webhook surface.
// It owns the HTTPS server, all webhook handlers, registration, and the
// webhook controller that reconciles configurations with the API server.
//
// Created via NewWebhookServer and registered as a domain.Komponent in
// cmd/internal/konstructor.go. It starts after the HealthServer so that
// /ready is live before webhook registration runs.
type WebhookServer struct {
	name string

	// HTTPS server for all webhook endpoints.
	server *http.Server
	mux    *http.ServeMux

	// Lifecycle flags
	started atomic.Bool

	// Context derived from Start() — cancelled on Shutdown().
	ctx    context.Context
	cancel context.CancelFunc

	// TLS and port config
	tlsCert   string
	tlsKey    string
	httpsPort string

	// Webhook registration options passed to registration functions.
	hookReg WebhookRegistrationOptions

	// Declarative enablement flags resolved from Katalog + konfig.
	webhooksEnabled  bool // admission (validation + mutation)
	convEnabled      bool // CRD conversion
	conversionWindow int  // rolling window size for conversion stats

	// Runtime protection state — set in Start() based on Katalog.
	deletionProtection  atomic.Bool
	namespaceProtection atomic.Bool

	// Kubernetes client for webhook configuration registration.
	kubeClient kubernetes.Interface

	// Katalog drives all conditional behavior.
	katalog *katalog.Katalog

	// Registries — populated from Katalog at construction time.
	conversionRegistry katalog.ConversionRegistry
	admissionRegistry  katalog.AdmissionRegistry

	// Stats — types live in pkg/health for compatibility with kordinator.
	// Always initialized so callers never receive nil even when webhooks are disabled.
	conversionStats *health.ConversionStats
	admissionStats  *health.AdmissionStats
	protectionStats *health.DeletionProtectionStats
	webhookStats    *health.WebhookStats
	namespaceStats  *health.NamespaceProtectionStats

	// Deletion protection — set of CRD full names (plural.group) that are protected.
	protectedCRDNames map[string]struct{}

	// Namespace protection — map of CRD key (plural.group) → NamespaceRules.
	namespaceRuleMap map[string]*NamespaceRules

	// certMgr handles TLS secret cleanup on graceful shutdown.
	// Nil when the user provided explicit TLS_CERT/TLS_KEY env vars.
	certMgr certManagerIface

	// Full konfig for namespace access during shutdown cleanup.
	konfig *konfig.Konfig
}

// NewWebhookServer constructs a WebhookServer and resolves all declarative
// webhook behavior from the Katalog and Konfig. No I/O is performed — Start()
// activates servers, registration, and reconciliation.
func NewWebhookServer(kubeClient kubernetes.Interface, kat *katalog.Katalog, kfg *konfig.Konfig) *WebhookServer {
	admissionFailurePolicy := admissionv1FailurePolicyType(kat.WebhooksFailurePolicy())

	hookReg := WebhookRegistrationOptions{
		FailurePolicy:            admissionFailurePolicy,
		Port:                     kfg.HTTPSPortInt32(),
		ServiceName:              kat.WebhooksServiceName(),
		ServiceNamespace:         kfg.Cluster().Namespace,
		TLSCertFile:              kfg.Security().Webhooks.TLSCert,
		OrkestraResourceLabels:   kfg.OrkestraResourceLabels(),
		OrkestraResourceSelector: kfg.OrkestraResourceSelector(),
	}

	convWindow := kat.ConversionWindow()

	ws := &WebhookServer{
		name:               "webhook server",
		kubeClient:         kubeClient,
		katalog:            kat,
		konfig:             kfg,
		httpsPort:          kfg.HTTPSPort(),
		tlsCert:            kfg.Security().Webhooks.TLSCert,
		tlsKey:             kfg.Security().Webhooks.TLSKey,
		hookReg:            hookReg,
		webhooksEnabled:    kat.IsAdmissionEnabled(),
		convEnabled:        kat.IsConversionEnabled(),
		conversionWindow:   convWindow,
		mux:                http.NewServeMux(),
		conversionRegistry: kat.ConversionRegistry(),
		admissionRegistry:  kat.AdmissionRegistry(),
		conversionStats:    health.NewConversionStats(convWindow),
		admissionStats:     health.NewAdmissionStats(convWindow),
		protectionStats:    health.NewDeletionProtectionStats(),
		webhookStats:       health.NewWebhookStats(),
	}

	// Precompute deletion-protection CRD name set.
	if kat.IsDeletionProtectionEnabled() {
		ws.protectedCRDNames = kat.DeletionProtectedCRDNames()
	}

	// Precompute namespace rule map.
	if kat.IsNamespaceProtectionEnabled() {
		rawRules := kat.NamespaceProtectionRuleMap()
		if len(rawRules) > 0 {
			ws.namespaceRuleMap = make(map[string]*NamespaceRules, len(rawRules))
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
				ws.namespaceRuleMap[key] = rules
			}
		}
		ws.namespaceStats = health.NewNamespaceProtectionStats()
	}

	return ws
}

// SetCertManager provides the cert manager for TLS secret cleanup on graceful shutdown.
// Set only when Orkestra generated self-signed certificates (certMgr == nil when the
// user provided explicit TLS_CERT/TLS_KEY).
func (ws *WebhookServer) SetCertManager(m certManagerIface) {
	ws.certMgr = m
}

// Start activates the WebhookServer. It registers all declared webhook endpoints,
// launches the HTTPS server, performs best-effort webhook registration with the
// API server, and starts the reconciliation controller.
//
// Start is a no-op when no webhook capabilities are declared in the Katalog or
// when the process is not running inside a Kubernetes cluster.
func (ws *WebhookServer) Start(ctx context.Context) error {
	ws.ctx, ws.cancel = context.WithCancel(ctx)
	ws.started.Store(true)

	kat := ws.katalog

	// Resolve protection state from the Katalog.
	if kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil {
		ws.deletionProtection.Store(true)
	}
	if kat.IsNamespaceProtectionEnabled() && len(kat.NamespaceProtectionGVRs()) > 0 {
		ws.namespaceProtection.Store(true)
	}

	logger.Debug().
		Bool("deletionProtection", ws.deletionProtection.Load()).
		Bool("namespaceProtection", ws.namespaceProtection.Load()).
		Bool("conversion", ws.convEnabled).
		Bool("webhooks", ws.webhooksEnabled).
		Msg("webhook server configured")

	if !utils.IsRunningInCluster() {
		logger.Debug().Msg("not running in cluster — skipping webhook HTTPS setup")
		return nil
	}

	startHTTPS := false
	admissionRuleExists := false

	// Register endpoints based on declared capabilities.
	if ws.deletionProtection.Load() {
		ws.mux.HandleFunc("/deletion-protection", ws.deletionProtectionHandler)
		startHTTPS = true
		logger.Info().Str("endpoint", "/deletion-protection").Msg("deletion protection endpoint registered")
	}
	if ws.namespaceProtection.Load() {
		ws.mux.HandleFunc("/namespace-protection", ws.namespaceProtectionHandler)
		startHTTPS = true
		logger.Info().Str("endpoint", "/namespace-protection").Msg("namespace protection endpoint registered")
	}
	if kat.HasConversionPaths() {
		ws.mux.HandleFunc("/convert", ws.conversionHandler)
		startHTTPS = true
		logger.Info().Str("endpoint", "/convert").Msg("conversion webhook endpoint registered")
	}
	if kat.HasValidationRules() {
		ws.mux.HandleFunc("/validate", ws.validationHandler)
		admissionRuleExists = true
		startHTTPS = true
		logger.Info().Str("endpoint", "/validate").Msg("validation endpoint registered")
	}
	if kat.HasMutationRules() {
		ws.mux.HandleFunc("/mutate", ws.mutationHandler)
		admissionRuleExists = true
		startHTTPS = true
		logger.Info().Str("endpoint", "/mutate").Msg("mutation endpoint registered")
	}

	if !startHTTPS {
		logger.Debug().Msg("no webhook endpoints declared — HTTPS server not started")
		return nil
	}

	// Validate TLS prerequisites.
	if ws.tlsCert == "" {
		return fmt.Errorf("webhook server: TLS_CERT is required when webhook endpoints are active")
	}
	if ws.tlsKey == "" {
		return fmt.Errorf("webhook server: TLS_KEY is required when webhook endpoints are active")
	}

	httpsAddr := ws.httpsPort
	if !strings.HasPrefix(httpsAddr, ":") {
		httpsAddr = ":" + httpsAddr
	}

	ws.server = &http.Server{
		Addr:    httpsAddr,
		Handler: ws.mux,
	}

	go func() {
		logger.Info().
			Str("addr", httpsAddr).
			Str("cert", ws.tlsCert).
			Msg("webhook HTTPS server listening")
		if err := ws.server.ListenAndServeTLS(ws.tlsCert, ws.tlsKey); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Str("addr", httpsAddr).Msg("webhook HTTPS server error")
		}
	}()

	// Best-effort admission webhook registration.
	if admissionRuleExists && ws.kubeClient != nil && ws.admissionRegistry != nil {
		go func() {
			wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := RegisterAdmissionWebhooks(wctx, ws.kubeClient, ws.admissionRegistry, ws.hookReg); err != nil {
				logger.Error().Err(err).Msg("admission webhook registration failed — RBAC may be missing")
			}
		}()
	}

	// Deletion protection webhook registration.
	if ws.deletionProtection.Load() && ws.kubeClient != nil {
		go func() {
			wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			dpGVRs := kat.DeletionProtectionGVRs()
			if len(dpGVRs) == 0 {
				logger.Info().Msg("deletion protection: no GVRs — webhook not registered")
				return
			}

			caBundle, err := readCABundle(ws.hookReg.TLSCertFile)
			if err != nil {
				logger.Error().Err(err).Msg("deletion protection webhook: cannot read CA bundle")
				return
			}

			if err := registerDeletionProtectionWebhook(wctx, ws.kubeClient, dpGVRs, caBundle, ws.hookReg); err != nil {
				logger.Error().Err(err).Msg("deletion protection webhook registration failed")
			} else {
				logger.Info().
					Str("config", deletionProtectionWebhookConfigName).
					Int("rules", len(dpGVRs)).
					Int("protected", len(ws.katalog.DeletionProtectedCRDNames())).
					Msg("deletion protection webhook registered")
			}
		}()
	}

	// Namespace protection webhook registration.
	if ws.namespaceProtection.Load() && ws.kubeClient != nil {
		go func() {
			wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			npGVRs := kat.NamespaceProtectionGVRs()
			if len(npGVRs) == 0 {
				logger.Info().Msg("namespace protection: no GVRs — webhook not registered")
				return
			}

			caBundle, err := readCABundle(ws.hookReg.TLSCertFile)
			if err != nil {
				logger.Error().Err(err).Msg("namespace protection webhook: cannot read CA bundle")
				return
			}

			svcName := kat.NamespaceProtectionServiceName()
			failurePolicy := kat.NamespaceProtectionFailurePolicy()
			if err := registerNamespaceProtectionWebhook(wctx, ws.kubeClient, npGVRs, caBundle, ws.hookReg, svcName, failurePolicy); err != nil {
				logger.Error().Err(err).Msg("namespace protection webhook registration failed")
			} else {
				logger.Info().
					Str("config", namespaceProtectionWebhookConfigName).
					Int("rules", len(npGVRs)).
					Msg("namespace protection webhook registered")
			}
		}()
	}

	// Start the housekeeper to keep webhook configurations in sync.
	if kat.IsWebhookControllerEnabled() {
		if err := ws.housekeeper(ws.ctx); err != nil {
			logger.Error().Err(err).Msg("housekeeper failed to start")
		}
	}

	return nil
}

// Shutdown gracefully stops the HTTPS server and performs declarative cleanup
// of webhook configurations as declared in the Katalog.
func (ws *WebhookServer) Shutdown(ctx context.Context) {
	if ws.cancel != nil {
		ws.cancel()
	}

	if ws.server == nil {
		return
	}

	if err := ws.server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("webhook HTTPS server shutdown error")
	}

	kat := ws.katalog

	// Cleanup admission webhook.
	cleanupOpts := WebhookCleanupOptions{}
	if kat.HasMutationRules() {
		cleanupOpts.mutating = kat.WebhookCleanupOnShutdown()
	}
	if kat.HasValidationRules() {
		cleanupOpts.validating = kat.WebhookCleanupOnShutdown()
	}
	if err := UnregisterAdmissionWebhooks(ctx, ws.kubeClient, cleanupOpts); err != nil {
		logger.Error().Err(err).Msg("webhook cleanup error")
	}

	// Cleanup deletion-protection webhook.
	if kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionCleanupOnShutdown() && ws.kubeClient != nil {
		if err := cleanupValidatingWebhook(ctx, ws.kubeClient, deletionProtectionWebhookConfigName); err != nil {
			logger.Error().Err(err).Msg("deletion protection webhook cleanup error")
		} else {
			logger.Info().Str("config", deletionProtectionWebhookConfigName).Msg("deletion protection webhook removed")
		}
	}

	// Cleanup namespace-protection webhook.
	if kat.IsNamespaceProtectionEnabled() && kat.NamespaceProtectionCleanupOnShutdown() && ws.kubeClient != nil {
		if err := cleanupValidatingWebhook(ctx, ws.kubeClient, namespaceProtectionWebhookConfigName); err != nil {
			logger.Error().Err(err).Msg("namespace protection webhook cleanup error")
		} else {
			logger.Info().Str("config", namespaceProtectionWebhookConfigName).Msg("namespace protection webhook removed")
		}
	}

	// Cleanup the TLS secret when auto-generated certs and cleanup is enabled.
	shouldCleanupTLS :=
		kat.WebhookCleanupOnShutdown() ||
			kat.NamespaceProtectionCleanupOnShutdown() ||
			kat.DeletionProtectionCleanupOnShutdown()

	if ws.certMgr != nil && shouldCleanupTLS && ws.konfig != nil {
		ns := ws.konfig.Cluster().Namespace
		if err := ws.certMgr.DeleteCertificateAndSecret(ctx, ns, konfig.DefaultInternalTLSName()); err != nil {
			logger.Error().Err(err).Msg("tls secret cleanup error")
		} else {
			logger.Info().Str("namespace", ns).Msg("tls secret removed on shutdown")
		}
	}
}

// Name returns the component name.
func (ws *WebhookServer) Name() string { return ws.name }

// Started reports whether Start() has been called.
func (ws *WebhookServer) Started() bool { return ws.started.Load() }

// ── Stats getters — called from konstructor.go to wire into BuildCRDInfoHandler ─

// GetConversionStats returns the conversion statistics instance.
func (ws *WebhookServer) GetConversionStats() *health.ConversionStats { return ws.conversionStats }

// GetAdmissionStats returns the admission statistics instance.
func (ws *WebhookServer) GetAdmissionStats() *health.AdmissionStats { return ws.admissionStats }

// GetProtectionStats returns the deletion-protection statistics instance.
func (ws *WebhookServer) GetProtectionStats() *health.DeletionProtectionStats {
	return ws.protectionStats
}

// GetWebhookStats returns the webhook reconciliation statistics instance.
func (ws *WebhookServer) GetWebhookStats() *health.WebhookStats { return ws.webhookStats }

// GetNamespaceStats returns the namespace-protection statistics instance.
func (ws *WebhookServer) GetNamespaceStats() *health.NamespaceProtectionStats {
	return ws.namespaceStats
}
