package main

import (
	"context"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/config"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/leader"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		logger.Fatal().AnErr("failed to load configurations", err)
		utils.Exit(err)
	}

	// initilaize logger
	logger.Init(cfg.App().LogLevel)

	// define root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create domain components and build manager
	startup := buildManager(cfg, ctx)

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
		func(ctx context.Context) { startup.controller.RunOrDie(ctx) }, // controller run
		leader.Options{
			Namespace:     cfg.Cluster().Namespace,
			LeaseDuration: cfg.Leader().LeaseDuration,
			RenewDeadline: cfg.Leader().RenewDeadline,
			RetryPeriod:   cfg.Leader().RetryPeriod,
		})

	// start leader election as postStartHook AFTER manager is ready
	startup.manager.AddPostStartHook(leader, func(ctx context.Context) {
		logger.Info().Msg("starting leader election...")
		leader.Start(ctx)
	})

	// Keep running until cancelled
	startup.manager.Wait()
}
