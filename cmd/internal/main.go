package internal

import (
	"context"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/leader"
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

	leader := leader.NewLeaderElection(
		startup.kube,
		startup.event,
		func(ctx context.Context) { startup.kontroller.RunOrDie(ctx) }, // kontroller run
		leader.Options{
			Namespace:     kfg.Cluster().Namespace,
			LeaseDuration: kfg.Leader().LeaseDuration,
			RenewDeadline: kfg.Leader().RenewDeadline,
			RetryPeriod:   kfg.Leader().RetryPeriod,
		})

	// start leader election as postStartHook AFTER orkestra is ready
	startup.orkestra.AddPostStartHook(leader, func(ctx context.Context) {
		logger.Info().Msg("starting leader election...")
		leader.Start(ctx)
	})

	// Keep running until cancelled
	startup.orkestra.Wait()
}
