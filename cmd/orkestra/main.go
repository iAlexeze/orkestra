package main

import (
	"context"

	"github.com/orkspace/orkestra/cmd/cli"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
)

func main() {
	kfg, err := konfig.Init()
	if err != nil {
		logger.Fatal().AnErr("failed to load configurations", err)
		utils.Exit(err)
	}

	// define root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli.Execute(kfg, ctx)
}
