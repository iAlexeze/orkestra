package kontroller

import (
	"context"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// retryMissingCRDs runs continuously to detect and activate CRDs that were missing at startup.
//
// It runs forever because CRDs can be installed after Orkestra starts.
// The loop stops only when the context is cancelled (leadership lost or shutdown).
//
// Flow:
//   - Periodically checks the missing map
//   - When a missing CRD appears in the cluster, activateCRD is called
//   - Uses exponential backoff to avoid API server pressure
//
// Note: This loop handles activation only. Deactivation is not implemented —
//
//	if a CRD is deleted after startup, the informer continues running.
//	But workers are drained through deactivateCRD.
func (k *DependencyKontroller) retryMissingCRDs(ctx context.Context) {
	ticker := time.NewTicker(PostStartRetryInterval)
	defer ticker.Stop()

	// backoff starts small (fast retries)
	backoff := PostStartBackoff

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			runtimeMissing := make(map[string]*informer.InformerEntry)
			// ───────────────────────────────────────────────
			// 1. Detect CRDs that disappeared at runtime
			// ───────────────────────────────────────────────
			registered := k.informerFactory.Registered()

			for gvkStr, entry := range registered {
				if entry == nil || entry.Missing {
					continue
				}

				ok, _ := k.crdExists(entry.GVK)
				if !ok {
					logger.Warn().Str("gvk", gvkStr).
						Msg("CRD disappeared at runtime — marking missing")

					entry.Missing = true
					runtimeMissing[gvkStr] = entry
					k.informerFactory.SetMissingOnStartup(runtimeMissing)
					k.crdHealthMap[gvkStr].SetMissingAtRuntime()
					// k.crdHealthMap[gvkStr].SetWorkersActive(0)

					// Stop workers (queue stays alive)
					if !k.deactivated[gvkStr] {
						k.crdHealthMap[gvkStr].StartedAt()
						logger.Info().Str("gvk", gvkStr).Msg("stopping workers")
						k.deactivateCRD(gvkStr)
					}
				}
			}

			// ───────────────────────────────────────────────
			// 2. Handle CRDs currently missing
			// ───────────────────────────────────────────────
			missing := k.informerFactory.Missing()

			if len(missing) == 0 {
				// No missing CRDs → system healthy → slow mode
				// Reset backoff so next missing event is fast again
				backoff = PostStartBackoff

				logger.Debug().Msg("retry loop: no missing CRDs")
				continue
			}

			// Missing CRDs → fast mode
			// Reset backoff so we retry quickly
			backoff = PostStartBackoff

			logger.Debug().Msgf("retry loop: checking %d missing CRD(s)", len(missing))

			stillMissing := make(map[string]*informer.InformerEntry)

			for gvkStr, entry := range missing {
				ok, _ := k.crdExists(entry.GVK)
				if ok {
					// CRD reappeared → activate immediately
					k.activateCRD(ctx, entry)
					k.deactivated[gvkStr] = false
				} else {
					// Still missing → keep tracking it
					stillMissing[gvkStr] = entry
					k.crdHealthMap[gvkStr].SetMissingAtRuntime()

					logger.Debug().Msgf("retry loop: %s still not available", gvkStr)
				}
			}

			// Update missing map
			k.informerFactory.SetMissingOnStartup(stillMissing)

			if len(stillMissing) == 0 {
				logger.Info().Msg("retry loop: all CRDs activated")
				if k.anyOnline.Load() {
					k.hs.SetReady()
					k.orkHealth.SetOrkReady()
				}
			} else {
				logger.Info().Msgf("retry loop: %d CRD(s) still missing", len(stillMissing))
			}

			// ───────────────────────────────────────────────
			// 3. Exponential backoff (only when still missing)
			// ───────────────────────────────────────────────
			if len(stillMissing) > 0 {
				time.Sleep(backoff)

				// Cap at 1 minute
				if backoff < time.Minute {
					backoff *= 2
				}
			}
		}
	}
}

// activateCRD brings a previously missing CRD online after it appears in the cluster.
//
// The informer for this CRD was already created during factory initialization
// but was never started because the CRD did not exist at startup.
// This method starts the existing informer, launches its worker pool, and signals
// any dependents that this CRD is now ready.
//
// This method does NOT recreate the informer — it reuses the one created during
// factory setup, which avoids the "sharedIndexInformer started twice" warning.
//
// Flow:
//  1. Start the existing (but not yet running) informer
//  2. Remove from missing map so it won't be retried again
//  3. Start worker goroutines for this CRD
//  4. Update health tracking to reflect started state
//  5. Close the ready channel to unblock any CRDs that depend on this one
//
// Parameters:
//   - ctx: context for cancellation propagation
//   - entry: the informer entry from the missing map (contains the pre-created informer)
func (k *DependencyKontroller) activateCRD(ctx context.Context, entry *informer.InformerEntry) {
	gvkStr := entry.GVK.String()
	name := entry.Name

	logger.Info().Msgf("activating CRD %s (%s)", name, gvkStr)

	// Safety check — the informer should exist because it was created during factory init
	if entry.Informer == nil {
		logger.Error().Msgf("activateCRD: informer for %s is nil", gvkStr)
		return
	}

	// Start the existing informer (it was created but not started at startup)
	// Only start informer if it was never started (startup-missing case).
	if entry.WasNeverStarted { // you can track this with a bool on InformerEntry
		go entry.Informer.Run(ctx.Done())
		entry.WasNeverStarted = false
		logger.Info().Msgf("activateCRD: informer started for %s", name)
	}

	// Mark as no longer missing so the retry loop won't keep processing it
	entry.Missing = false
	k.informerFactory.RemoveMissing(gvkStr)

	// Start the worker pool for this CRD
	workers := k.katalog.GetWorkers(gvkStr, k.defaultWorkers)
	k.startCRDWorkers(ctx, gvkStr, workers)

	// Update health tracking
	k.deactivated[gvkStr] = false
	k.crdHealthMap[gvkStr].SetStarted()
	k.crdHealthMap[gvkStr].SetWorkersActive(workers)

	// Signal dependents that this CRD is now ready
	// The ready channel was created during RunOrDie and may still be open
	if ch, exists := k.readyCh[gvkStr]; exists {
		select {
		case <-ch:
			// Channel already closed (should not happen, but safe)
			logger.Debug().Msgf("activateCRD: ready channel for %q was already closed", name)
		default:
			close(ch)
			deps := k.depGraph.GetDependents(name)
			if len(deps) > 0 {
				logger.Info().Msgf("activateCRD: closed ready channel for %q — unblocking %d dependent(s): %s",
					name, len(deps), strings.Join(deps, ", "))
			}
		}
	}

	logger.Info().Msgf("CRD %s activated", name)
}

// deactivateCRD deactivates a missing crd at runtime
func (k *DependencyKontroller) deactivateCRD(gvk string) {
	// Only cancel context, don’t ShutDown the queue
	k.mu.RLock()
	cancel, okCancel := k.cancelFuncs[gvk]
	wg, okWG := k.wgs[gvk]
	k.mu.RUnlock()

	if okCancel {
		cancel()
	}

	if !okWG {
		return
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		k.crdHealthMap[gvk].SetWorkersActive(0)
		k.deactivated[gvk] = true
		logger.Info().Str("gvk", gvk).Msg("workers drained cleanly")
	case <-time.After(k.drainTimeout):
		logger.Warn().Str("gvk", gvk).Msg("drain timeout exceeded")
	}
}

// crdExists checks if a CRD is present in the cluster by querying the API server.
// Returns (true, nil) if the CRD exists, (false, nil) if not, (false, error) on failure.
func (k *DependencyKontroller) crdExists(gvk *schema.GroupVersionKind) (bool, error) {
	return utils.WaitForCRD(
		k.kube.RestConfig(),
		gvk.Group,
		gvk.Kind,
		gvk.Version,
	) == nil, nil
}
