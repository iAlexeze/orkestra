package main

import (
	"context"

	"github.com/ialexeze/orkestra/cmd/ork"
	"github.com/ialexeze/orkestra/pkg/konfig"
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

	ork.Execute(kfg, ctx)
}
