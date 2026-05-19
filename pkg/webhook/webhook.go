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
//     /deletion-protection, /namespace-protection, and /strict-mode-protection
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
	"github.com/orkspace/orkestra/pkg/labels"
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
	strictMode          atomic.Bool

	// Kubernetes client for webhook configuration registration.
	kubeClient kubernetes.Interface

	// Katalog drives all conditional behavior.
	katalog *katalog.Katalog

	// Registries — populated from Katalog at construction time.
	conversionRegistry katalog.ConversionRegistry
	admissionRegistry  katalog.AdmissionRegistry

	// Stats — per-CRD maps keyed by GVR string ("group/version/resource").
	// Pre-populated from the Katalog in NewWebhookServer so each CRD gets its
	// own counters. infraProtectionStats covers self-protection and Orkestra
	// infra resources (Deployment, Service, etc.) that have no CRD GVR.
	admissionStats  map[string]*health.AdmissionStats
	conversionStats map[string]*health.ConversionStats
	protectionStats map[string]*health.DeletionProtectionStats
	namespaceStats  map[string]*health.NamespaceProtectionStats
	infraProtStats  *health.DeletionProtectionStats // webhook self + Orkestra infra
	strictModeStats *health.DeletionProtectionStats // process-global; strict mode is not per-CRD
	webhookStats    *health.WebhookStats

	// Reverse-lookup tables built from the Katalog for handlers that identify
	// the target CRD by name/kind rather than GVR.
	crdNameToGVRKey map[string]string // "plural.group" → "group/version/resource"
	kindToGVRKey    map[string]string // "Kind" → "group/version/resource"

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
		FailurePolicy:          admissionFailurePolicy,
		Port:                   kfg.HTTPSPortInt32(),
		ServiceName:            kat.WebhooksServiceName(),
		ServiceNamespace:       kfg.Cluster().Namespace,
		TLSCertFile:            kfg.Security().Webhooks.TLSCert,
		OrkestraResourceLabels: labels.OrkestraResourceLabels(),
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
		admissionStats:     make(map[string]*health.AdmissionStats),
		conversionStats:    make(map[string]*health.ConversionStats),
		protectionStats:    make(map[string]*health.DeletionProtectionStats),
		namespaceStats:     make(map[string]*health.NamespaceProtectionStats),
		infraProtStats:     health.NewDeletionProtectionStats(),
		webhookStats:       health.NewWebhookStats(),
		strictModeStats:    health.NewDeletionProtectionStats(),
		crdNameToGVRKey:    make(map[string]string),
		kindToGVRKey:       make(map[string]string),
	}

	// Pre-populate per-CRD stat instances and reverse-lookup tables from the Katalog.
	// Every CRD gets its own counters so the gateway /katalog can return accurate
	// per-CRD breakdowns without mixing traffic across resources.
	for _, crd := range kat.All() {
		gvr := crd.GVR()
		gvrKey := crdGVRKey(gvr.Group, gvr.Version, gvr.Resource)
		ws.admissionStats[gvrKey] = health.NewAdmissionStats(convWindow)
		ws.conversionStats[gvrKey] = health.NewConversionStats(convWindow)
		ws.protectionStats[gvrKey] = health.NewDeletionProtectionStats()
		ws.namespaceStats[gvrKey] = health.NewNamespaceProtectionStats()
		crdFullName := crd.APITypes.Plural + "." + crd.APITypes.Group
		ws.crdNameToGVRKey[crdFullName] = gvrKey
		ws.kindToGVRKey[crd.APITypes.Kind] = gvrKey
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
	}

	return ws
}

// crdGVRKey formats a GVR triple into a canonical string key used throughout
// the webhook stats maps: "group/version/resource" (or "version/resource" for core group).
func crdGVRKey(group, version, resource string) string {
	if group == "" {
		return version + "/" + resource
	}
	return group + "/" + version + "/" + resource
}

// ── Per-CRD stat accessors ────────────────────────────────────────────────────
// These helpers return the per-CRD stats instance for a given GVR key.
// A nil-safe fallback is created on first miss (defensive; should not occur for
// CRDs declared in the Katalog).

func (ws *WebhookServer) admissionStatsFor(gvrKey string) *health.AdmissionStats {
	if s, ok := ws.admissionStats[gvrKey]; ok {
		return s
	}
	s := health.NewAdmissionStats(ws.conversionWindow)
	ws.admissionStats[gvrKey] = s
	return s
}

func (ws *WebhookServer) conversionStatsFor(gvrKey string) *health.ConversionStats {
	if s, ok := ws.conversionStats[gvrKey]; ok {
		return s
	}
	s := health.NewConversionStats(ws.conversionWindow)
	ws.conversionStats[gvrKey] = s
	return s
}

func (ws *WebhookServer) protectionStatsFor(gvrKey string) *health.DeletionProtectionStats {
	if s, ok := ws.protectionStats[gvrKey]; ok {
		return s
	}
	s := health.NewDeletionProtectionStats()
	ws.protectionStats[gvrKey] = s
	return s
}

func (ws *WebhookServer) namespaceStatsFor(gvrKey string) *health.NamespaceProtectionStats {
	if s, ok := ws.namespaceStats[gvrKey]; ok {
		return s
	}
	s := health.NewNamespaceProtectionStats()
	ws.namespaceStats[gvrKey] = s
	return s
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
	if kat.IsStrictModeEnabled() && utils.IsRunningInCluster() {
		ws.strictMode.Store(true)
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
	if ws.strictMode.Load() {
		ws.mux.HandleFunc("/strict-mode-protection", ws.strictModeProtectionHandler)
		startHTTPS = true
		logger.Info().Str("endpoint", "/strict-mode-protection").Msg("strict mode protection endpoint registered")
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

	// Strict mode webhook registration.
	if ws.strictMode.Load() && ws.kubeClient != nil {
		go func() {
			wctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			caBundle, err := readCABundle(ws.hookReg.TLSCertFile)
			if err != nil {
				logger.Error().Err(err).Msg("strict mode webhook: cannot read CA bundle")
				return
			}

			if err := registerStrictModeProtectionWebhook(wctx, ws.kubeClient, caBundle, ws.hookReg); err != nil {
				logger.Error().Err(err).Msg("strict mode protection webhook registration failed")
			} else {
				logger.Info().
					Str("config", strictModeProtectionWebhookConfigName).
					Msg("strict mode protection webhook registered")
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

	// Cleanup strict-mode-protection webhook.
	// Strict mode cleanup mirrors deletion protection: only clean up when the
	// parent deletionProtection.cleanupOnShutdown is set, since strict mode is
	// a sub-feature of deletion protection.
	if kat.IsStrictModeEnabled() && kat.DeletionProtectionCleanupOnShutdown() && ws.kubeClient != nil {
		if err := cleanupValidatingWebhook(ctx, ws.kubeClient, strictModeProtectionWebhookConfigName); err != nil {
			logger.Error().Err(err).Msg("strict mode protection webhook cleanup error")
		} else {
			logger.Info().Str("config", strictModeProtectionWebhookConfigName).Msg("strict mode protection webhook removed")
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

// ── Stats getters — used by BuildGatewayKatalogHandler ───────────────────────
// All stat getters are keyed by GVR string ("group/version/resource") so the
// gateway /katalog handler can serve accurate per-CRD breakdowns.

// AdmissionStatsFor returns the admission stats for the CRD identified by gvrKey.
// Returns nil when no stats exist for that key (CRD has no admission webhooks).
func (ws *WebhookServer) AdmissionStatsFor(gvrKey string) *health.AdmissionStats {
	return ws.admissionStats[gvrKey]
}

// ConversionStatsFor returns the conversion stats for the CRD identified by gvrKey.
func (ws *WebhookServer) ConversionStatsFor(gvrKey string) *health.ConversionStats {
	return ws.conversionStats[gvrKey]
}

// ProtectionStatsFor returns the deletion-protection stats for the CRD identified by gvrKey.
func (ws *WebhookServer) ProtectionStatsFor(gvrKey string) *health.DeletionProtectionStats {
	return ws.protectionStats[gvrKey]
}

// NamespaceStatsFor returns the namespace-protection stats for the CRD identified by gvrKey.
func (ws *WebhookServer) NamespaceStatsFor(gvrKey string) *health.NamespaceProtectionStats {
	return ws.namespaceStats[gvrKey]
}

// InfraProtectionStats returns the process-level deletion-protection stats that
// cover the webhook configuration itself and Orkestra infra resources (Deployment,
// Service, etc.) — events not attributable to a specific CRD GVR.
func (ws *WebhookServer) InfraProtectionStats() *health.DeletionProtectionStats {
	return ws.infraProtStats
}

// WebhookControllerStats returns the webhook reconciliation statistics.
func (ws *WebhookServer) WebhookControllerStats() *health.WebhookStats { return ws.webhookStats }

// StrictModeStats returns the strict-mode-protection statistics (process-global).
func (ws *WebhookServer) StrictModeStats() *health.DeletionProtectionStats {
	return ws.strictModeStats
}

// GVRKey formats group/version/resource into the canonical stats key.
// Exported so the gateway handler can build keys from crd.GVR() without
// duplicating the formatting logic.
func GVRKey(group, version, resource string) string {
	return crdGVRKey(group, version, resource)
}
