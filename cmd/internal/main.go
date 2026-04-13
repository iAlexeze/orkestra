package internal

import (
	"context"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/konductor"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/merger"
	"github.com/ialexeze/orkestra/pkg/utils"
)

func Konduct(kfg *konfig.Konfig, m *merger.Merger, ctx context.Context) {
	// create domain komponent and build orkestra
	startup := konstructOrkestra(kfg, m, ctx)

	// ── Shutdown hooks ────────────────────────────────────────────────────────
	// Webhook cleanup (deletion protection, validation, mutation) is already
	// handled by HealthServer.Shutdown() — it owns the webhook lifecycle and
	// runs as part of the komponent shutdown sequence.
	//
	// Only register hooks here for things HealthServer does not know about.

	// RBAC cleanup — HealthServer has no knowledge of RBAC so this cannot go there.
	// Only fires when cleanupOnShutdown: true (default: false).
	// Leave RBAC in place across normal restarts — only clean up for
	// test environments or explicitly ephemeral operators.
	if startup.katalog.IsRBACEnabled() && startup.katalog.RBACCleanupOnShutdown() {
		saName := startup.katalog.Metadata().Name + "-operator"
		if saName == "-operator" {
			saName = "orkestra-operator"
		}
		rbacBundle := startup.katalog.BuildRBACBundle(kfg.Cluster().Namespace, saName)
		startup.orkestra.OnShutdown(func(ctx context.Context) {
			logger.Info().Msg("shutdown: removing RBAC")
			if err := katalog.DeleteRBAC(ctx, startup.kube.Clientset(), rbacBundle); err != nil {
				logger.Warn().Err(err).Msg("shutdown: RBAC cleanup failed")
			}
		})
	}

	// ── Start ─────────────────────────────────────────────────────────────────
	go func() {
		if err := startup.orkestra.Start(ctx); err != nil {
			logger.Fatal().AnErr("orkestra startup error", err)
			utils.Exit(err)
		}
	}()

	ko := konductor.NewKonductorElection(
		startup.kube,
		startup.event,
		func(ctx context.Context) { startup.kord.Kordinate(ctx) },
		func(konductor string) {
			// Banner prints here — konductor is the actual winner
			printBanner(startup, konductor)
		},
		konductor.Options{
			Namespace:     kfg.Konductor().Namespace,
			LeaseDuration: kfg.Konductor().LeaseDuration,
			RenewDeadline: kfg.Konductor().RenewDeadline,
			RetryPeriod:   kfg.Konductor().RetryPeriod,
		})

	// start konductor election as postStartHook after  orkestra is ready
	startup.orkestra.AddPostStartHook(ko, func(ctx context.Context) {
		logger.Info().Msg("starting konductor election...")
		ko.Start(ctx)
	})

	// Keep running until cancelled
	startup.orkestra.Wait()
}
