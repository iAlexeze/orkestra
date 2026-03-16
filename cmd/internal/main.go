package internal

import (
	"context"

	"github.com/ialexeze/orkestra/pkg/konductor"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
)

func Konduct(kfg *konfig.Konfig, ctx context.Context) {
	// create domain komponent and build orkestra
	startup := konstructOrkestra(kfg, ctx)

	// Start all orkestra komponent
	go func() {
		if err := startup.orkestra.Start(ctx); err != nil {
			logger.Fatal().AnErr("orkestra startup error", err)
			utils.Exit(err)
		}
	}()

	ko := konductor.NewKonductorElection(
		startup.kube,
		startup.event,
		func(ctx context.Context) { startup.kontroller.RunOrDie(ctx) }, // kontroller run
		func(konductor string) {
			// Banner prints here — Leader is the actual winner
			printBanner(startup, konductor)
		},
		konductor.Options{
			Namespace:     kfg.Cluster().DefaultNamespace,
			LeaseDuration: kfg.Konductor().LeaseDuration,
			RenewDeadline: kfg.Konductor().RenewDeadline,
			RetryPeriod:   kfg.Konductor().RetryPeriod,
		})

	// start konductor election as postStartHook AFTER orkestra is ready
	startup.orkestra.AddPostStartHook(ko, func(ctx context.Context) {
		logger.Info().Msg("starting konductor election...")
		ko.Start(ctx)
	})

	// Keep running until cancelled
	startup.orkestra.Wait()
}
