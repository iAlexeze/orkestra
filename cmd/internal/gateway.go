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
	"strings"

	"github.com/orkspace/orkestra/domain"
	apigateway "github.com/orkspace/orkestra/pkg/gateway/api"
	"github.com/orkspace/orkestra/pkg/gateway/certmanager"
	gwhandlers "github.com/orkspace/orkestra/pkg/gateway/handlers"
	"github.com/orkspace/orkestra/pkg/gateway/webhook"
	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	ork "github.com/orkspace/orkestra/pkg/orkestra"
	"github.com/orkspace/orkestra/pkg/utils"
)

// KonductGateway starts the gateway — TLS + WebhookServer only.
// No konductor election — the gateway is stateless and supports multiple replicas.
func KonductGateway(kfg *konfig.Konfig, m *merger.Merger, ctx context.Context) {

	if !utils.IsRunningInCluster() {
		fmt.Println("orkestra: ork gate only runs inside a Kubernetes pod. Use 'ork run' for local development.")
		os.Exit(1)
	}

	// ── 1a. Instance ────────────────────────────────────────────────────────────
	kfg.SetInstance(konfig.Gateway())

	// ── 1b. Katalog ────────────────────────────────────────────────────────────
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
	var tlsBundle *certmanager.TLSBundle
	tlsCert, tlsKey, certMgr, tlsBundle, secErr = ensureSecurity(ctx, kfg, kat, kube)
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

	// ── 5. HealthServer — HTTP (probes + metrics + /katalog API) ────────────────
	hs := health.NewHealthServer(kfg)

	// ── 6. WebhookServer ──────────────────────────────────────────────────────
	ws := webhook.NewWebhookServer(kube.Clientset(), kat, kfg)
	if certMgr != nil {
		ws.SetCertManager(certMgr)
	}
	if tlsBundle != nil {
		ws.SetCertBundle(tlsBundle.CertPEM, tlsBundle.KeyPEM, tlsBundle.CACertPEM,
			certmanager.DefaultTLSSecretName, kfg.Cluster().Namespace())
	}
	// Wire housekeeper infrastructure reconcilers — keeps namespace labels and
	// CRD conversion caBundles correct throughout the deployment lifecycle.
	WireWebhookHousekeeperInfra(ws, kube, kat, kfg)

	// ── 7. /katalog routes — gateway serves its own stats surface ────────────
	// The control center discovers this endpoint via the "gatewayEndpoint" field
	// in the runtime /katalog response and merges per-CRD stats by GVR key.
	hs.Register("/katalog", gwhandlers.BuildGatewayKatalogHandler(kat, ws))

	// ── /notify — receives pre-throttled notification events from the runtime ─
	// The runtime builds and throttle-checks events; the gateway owns dispatch
	// (SMTP, Slack). Registering here keeps all external I/O off the runtime path.
	hs.Register("/notify", gwhandlers.BuildNotifyHandler(kat))
	for _, crd := range kat.Enabled() {
		crdName := strings.ToLower(crd.Name)
		gvr := crd.GVR()
		gvrKey := webhook.GVRKey(gvr.Group, gvr.Version, gvr.Resource)
		hs.Register(
			"/katalog/"+crdName,
			gwhandlers.BuildGatewayCRDHandler(crd.Name, crd.GVKString(), gvr.String(), gvrKey, ws, kat),
		)
	}
	logger.Debug().Msg("gateway /katalog routes registered")

	// ── 7b. Gateway API ────────────────────────────────────────────────────────
	// Registers POST /api/v1/apply, GET/DELETE /api/v1/resources/, GET /api/v1/schema/
	// only when gateway.api.enabled: true in the Katalog and at least one
	// CRD has serve.enabled: true.
	api, apiErr := apigateway.NewAPIServer(ctx, kat, kube, kfg.Cluster().Namespace())
	if apiErr != nil {
		logger.Fatal().Err(apiErr).Msg("gateway API setup failed")
	}
	if api != nil {
		api.Register(hs)
		ws.SetTokenReloader(api.ReloadTokens)
	}

	// ── 8. Komponent list ─────────────────────────────────────────────────────
	komponents := []domain.Komponent{
		hs,   // 1. HTTP server — /ready, /livez probes
		ws,   // 2. HTTPS webhook server — /validate, /mutate, /convert
		kube, // 3. REST clients — already started, managed for Stop()
	}

	// ── 8. Orkestra ───────────────────────────────────────────────────────────
	o := ork.NewOrkestra(
		kfg.RunningInstance(),
		kfg.Katalog().ShutdownGracePeriod(),
		kfg.Ork().LogLevel(),
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
