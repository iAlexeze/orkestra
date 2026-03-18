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
			logger.Debug().Msg("checking for missing CRDs...")

			missing := k.informerFactory.Missing()
			if len(missing) == 0 {
				return
			}

			stillMissing := make(map[string]*informer.InformerEntry)

			for key, entry := range missing {
				gvk := entry.GVK
				logger.Debug().Msgf("checking if CRD %s is now available", gvk)

				err := utils.WaitForCRD(
					k.kube.RestConfig(),
					gvk.Group,
					gvk.Kind,
					gvk.Version,
				)

				if err != nil {
					logger.Debug().Msgf("CRD %s still not available: %v", gvk, err)
					stillMissing[key] = entry
					continue
				}

				// CRD appeared — activate it
				logger.Info().Msgf("✅ CRD found and activating: %s", gvk)
				k.activateCRD(ctx, entry)
			}

			k.informerFactory.SetMissing(stillMissing)

			if len(stillMissing) == 0 {
				logger.Info().Msg("all CRDs activated — stopping retry loop")
				return
			} else {
				logger.Info().Msgf("%d CRD(s) still missing", len(missing))
			}

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
	name := entry.Name
	deps := k.depGraph.GetDependents(name)

	logger.Info().Msgf("activating CRD %s (%s)", name, gvkStr)

	// 1. Create informer dynamically
	inf := k.informerFactory.For(
		k.katalog.NewObjectForGVK(gvkStr).(runtime.Object),
		ctx,
		informer.Options{
			Name:   name,
			Resync: entry.Resync,
		},
	)

	if inf != nil {
		go inf.Run(ctx.Done())
	}

	if inf == nil {
		logger.Warn().Msgf("no informer created for %s (%s) during activation", name, gvkStr)
	} else {
		logger.Info().Msgf("reusing informer created for %s (%s)", name, gvkStr)
	}

	// 2. Start workers
	workers := k.katalog.GetWorkers(gvkStr, k.defaultWorkers)
	logger.Info().Msgf("starting %d workers for %s", workers, gvkStr)
	k.startCRDWorkers(ctx, gvkStr, workers)
	metrics.WorkersActive.WithLabelValues(gvkStr).Set(float64(workers))

	// 3. Mark started
	k.crdHealthMap[gvkStr].SetStarted()

	// 4. Signal dependents (safe close)
	ch := k.readyCh[name]
	if ch != nil {
		select {
		case <-ch:
			// already closed
		default:
			logger.Info().Msgf("🔓 Closing ready channel for %s, unblocking %d dependents: %s",
				name, len(deps), strings.Join(deps, ", "))
			close(ch)
		}
	}

	// 5. Metrics: activation latency
	latency := time.Since(k.startedAt).Seconds()
	metrics.CRDActivationLatency.WithLabelValues(name).Observe(latency)
	metrics.CRDActivationTotal.WithLabelValues(name, "success").Inc()

	logger.Info().Msgf("CRD %s activated successfully in %.3fs", name, latency)
}
