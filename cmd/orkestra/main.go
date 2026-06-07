// Package main is the entry point for the Orkestra CLI (ork).
//
// Orkestra is a declarative operator runtime for Kubernetes. You declare your
// operator in YAML — CRDs, reconcile templates, status, conditions, lifecycle
// hooks — and Orkestra runs the reconcile loop: informers, workqueue, worker
// pool, finalizers, RBAC, and drift correction are all provided.
//
// The runtime supports two authoring modes:
//
//   - Dynamic mode — zero Go code. The full operator runs from katalog.yaml alone.
//   - Typed mode — Go types and hooks for type-safe field access and custom business
//     logic, compiled into the same binary with make build.
//
// Key commands:
//
//	ork run          Start the Orkestra runtime with a Katalog or Komposer
//	ork simulate     Run the reconciler in memory — no cluster required
//	ork e2e          Declarative end-to-end tests against a real cluster
//	ork validate     Validate any Orkestra document (Katalog, Komposer, Motif, E2E, Simulate)
//	ork registry     Publish and pull operator patterns as OCI artifacts
//	ork control      Launch the live control center
//	ork gate         Start the admission and conversion webhook server
//
// The production binary (built with -tags runtime) contains only ork run.
// All developer commands are excluded from the runtime image.
//
// Documentation: https://orkestra.sh
// Source: https://github.com/orkspace/orkestra
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
