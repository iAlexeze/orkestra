// cmd/internal/gateway.go
//
// Gateway startup — a minimal process that handles only TLS/security
// setup and serves admission/conversion webhooks. No reconcilers, no informer
// factory, no konductor election (webhook servers are stateless and can run
// as multiple replicas).
package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/certmanager"
	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	ork "github.com/orkspace/orkestra/pkg/orkestra"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/orkspace/orkestra/pkg/webhook"
)

// KonductGateway starts the gateway — TLS + WebhookServer only.
// No konductor election — the gateway is stateless and supports multiple replicas.
func KonductGateway(kfg *konfig.Konfig, m *merger.Merger, ctx context.Context) {

	if !utils.IsRunningInCluster() {
		fmt.Println("orkestra: ork gateway only runs inside a Kubernetes pod. Use 'ork run' for local development.")
		os.Exit(1)
	}

	// ── 1. Katalog ────────────────────────────────────────────────────────────
	// Needed to know which CRDs require webhooks.
	kat := katalog.NewKatalog(kfg, m)

	if registryURL := kfg.RegistryConfig().RegistryURL; registryURL != "" {
		m.SetRegistryURL(registryURL)
		logger.Info().Str("registry", registryURL).Msg("registry URL configured from ORK_REGISTRY")
	}

	// ── 2. Scheme ─────────────────────────────────────────────────────────────
	scheme, err := katalog.NewSchemeRegistry(kat)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to build scheme registry")
	}

	// ── 3. Kubeclient ─────────────────────────────────────────────────────────
	kube := kubeclient.NewKubeclient(kfg, scheme)

	if err := kube.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start kubeclient")
	}

	// ── 4. Security ───────────────────────────────────────────────────────────
	// TLS cert management + CRD conversion webhook patching.
	// Only runs in-cluster; the guard at the top of this function ensures we
	// never reach this point outside a pod.
	var certMgr certmanager.Manager
	var tlsCert, tlsKey string
	var secErr error
	tlsCert, tlsKey, certMgr, secErr = ensureSecurity(ctx, kfg, kat, kube)
	if secErr != nil {
		logger.Fatal().Err(secErr).Msg("security setup failed")
	}

	if tlsCert != "" {
		logger.Debug().
			Str("cert_file", tlsCert).
			Str("cert_key", tlsKey).
			Msg("passing generated TLS cert to webhook server")
		kfg.Security().Webhooks.TLSCert = tlsCert
		kfg.Security().Webhooks.TLSKey = tlsKey
	}

	// ── 5. HealthServer — minimal (probes + metrics only, no katalog routes) ─
	hs := health.NewHealthServer(kfg)

	// ── 6. WebhookServer ──────────────────────────────────────────────────────
	ws := webhook.NewWebhookServer(kube.Clientset(), kat, kfg)
	if certMgr != nil {
		ws.SetCertManager(certMgr)
	}

	// ── 7. Komponent list ─────────────────────────────────────────────────────
	komponents := []domain.Komponent{
		hs,   // 1. HTTP server — /ready, /livez probes
		ws,   // 2. HTTPS webhook server — /validate, /mutate, /convert
		kube, // 3. REST clients — already started, managed for Stop()
	}

	// ── 8. Orkestra ───────────────────────────────────────────────────────────
	o := ork.NewOrkestra(
		kfg.Katalog().ShutdownGracePeriod,
		kfg.Ork().LogLevel,
	)
	o.Register(komponents)

	// ── Start and wait (no leader election) ──────────────────────────────────
	go func() {
		if err := o.Start(ctx); err != nil {
			logger.Fatal().AnErr("gateway startup error", err)
			utils.Exit(err)
		}
	}()

	logger.Info().Msg("gateway started — serving webhooks")

	o.Wait()
}
