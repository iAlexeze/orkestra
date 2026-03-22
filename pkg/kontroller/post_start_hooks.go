package kontroller

import (
	"context"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
)

// retryMissingCRDs periodically checks for CRDs that were missing at startup.
// When a CRD appears, it is activated (informer + workers) without restarting Orkestra.
func (k *DependencyKontroller) retryMissingCRDs(ctx context.Context) {
	ticker := time.NewTicker(PostStartRetryInterval)
	defer ticker.Stop()

	backoff := PostStartBackoff

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			missing := k.informerFactory.Missing()
			if len(missing) == 0 {
				logger.Info().Msg("retry loop: no missing CRDs — stopping")
				return
			}

			logger.Debug().Msgf("retry loop: checking %d missing CRD(s)", len(missing))

			stillMissing := make(map[string]*informer.InformerEntry)

			for key, entry := range missing {
				gvk := entry.GVK
				gvkStr := gvk.String()

				err := utils.WaitForCRD(
					k.kube.RestConfig(),
					gvk.Group,
					gvk.Kind,
					gvk.Version,
				)
				if err != nil {
					logger.Debug().Msgf("retry loop: %s still not available", gvkStr)
					stillMissing[key] = entry
					continue
				}

				logger.Info().Msgf("retry loop: CRD %s appeared — activating", gvkStr)
				k.activateCRD(ctx, entry)
			}

			// Update missing map — only what's still missing remains
			k.informerFactory.SetMissing(stillMissing)

			if len(stillMissing) == 0 {
				logger.Info().Msg("retry loop: all CRDs activated")
				// All CRDs now online — mark controller fully ready
				k.hs.SetReady()
				return
			}

			logger.Info().Msgf("retry loop: %d CRD(s) still missing", len(stillMissing))

			// Exponential backoff — cap at 1 minute
			time.Sleep(backoff)
			if backoff < time.Minute {
				backoff *= 2
			}
		}
	}
}

// activateCRD wires up informer + workers for a CRD that appeared after startup.
// It does NOT recreate Orkestra — just brings the CRD online.
func (k *DependencyKontroller) activateCRD(ctx context.Context, entry *informer.InformerEntry) {
	gvk := entry.GVK
	gvkStr := gvk.String()
	name := entry.Name // must match the name used as key in readyCh

	logger.Info().Msgf("activating CRD %s (%s)", name, gvkStr)

	// 1. Create and start informer
	inf := k.informerFactory.For(
		k.katalog.NewObjectForGVK(gvkStr).(runtime.Object),
		ctx,
		informer.Options{
			Name:   name,
			Resync: entry.Resync,
		},
	)

	if inf == nil {
		logger.Warn().Msgf("activateCRD: no informer created for %s", gvkStr)
	} else {
		go inf.Run(ctx.Done())
		logger.Info().Msgf("activateCRD: informer started for %s", name)
	}

	// 2. Start workers
	workers := k.katalog.GetWorkers(gvkStr, k.defaultWorkers)
	k.startCRDWorkers(ctx, gvkStr, workers)

	// Set the workers in health map
	k.crdHealthMap[gvkStr].SetWorkersActive(workers)

	// Emit metrics
	metrics.WorkersActive.WithLabelValues(gvkStr).Set(float64(workers))

	logger.Info().Msgf("activateCRD: %d workers started for %s", workers, name)

	// 3. Mark CRD health as started
	k.crdHealthMap[gvkStr].SetStarted()

	// 4. Close ready channel — This is what unblocks dependents in RunOrDie
	// RunOrDie's loop is blocked at:
	//   select { case <-k.readyCh[dep]: ... }
	// Closing readyCh[name] here unblocks any CRD that lists name in its dependsOn.
	ch, exists := k.readyCh[name]
	if !exists {
		logger.Warn().Msgf("activateCRD: no ready channel for %q — dependents may block forever", name)
		return
	}

	select {
	case <-ch:
		logger.Debug().Msgf("activateCRD: ready channel for %q was already closed", name)
	default:
		close(ch)
		deps := k.depGraph.GetDependents(name)
		logger.Info().Msgf("activateCRD: closed ready channel for %q — unblocking %d dependent(s): %s",
			name, len(deps), strings.Join(deps, ", "))
	}

	// 5. Metrics
	latency := time.Since(k.startedAt).Seconds()
	metrics.CRDActivationLatency.WithLabelValues(name).Observe(latency)
	metrics.CRDActivationTotal.WithLabelValues(name, "success").Inc()

	logger.Info().Msgf("CRD %s activated in %.3fs", name, latency)
}

/*
The flow now:
startupOrder: [A, B, C]   (B depends on A, C depends on B)

RunOrDie loop:
  A → missing → continue (readyCh["A"] stays open)
  B → waits on readyCh["A"] ← BLOCKS HERE (main goroutine)

retryMissingCRDs goroutine:
  ticker fires → A appeared → activateCRD(A)
    → starts A workers
    → closes readyCh["A"]  ← UNBLOCKS B in RunOrDie

RunOrDie loop continues:
  B → readyCh["A"] unblocked → B starts → closes readyCh["B"]
  C → readyCh["B"] unblocked → C starts → closes readyCh["C"]

retryMissingCRDs:
  stillMissing is empty → SetReady() → stops

*/
