package main

import (
	"context"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/leader"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
)

func main() {
	kfg, err := konfig.Init()
	if err != nil {
		logger.Fatal().AnErr("failed to load configurations", err)
		utils.Exit(err)
	}

	// initilaize logger
	logger.Init(kfg.App().LogLevel)

	// define root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create domain components and build manager
	startup := buildManager(kfg, ctx)

	// Start all manager components
	go func() {
		if err = startup.manager.Start(ctx); err != nil {
			logger.Fatal().AnErr("manager startup error", err)
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

	// start leader election as postStartHook AFTER manager is ready
	startup.manager.AddPostStartHook(leader, func(ctx context.Context) {
		logger.Info().Msg("starting leader election...")
		leader.Start(ctx)
	})

	// Keep running until cancelled
	startup.manager.Wait()
}
