package kordinator

import (
	"context"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/informer"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/queue"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
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
// dependenciesReady returns true if all declared dependencies are currently
// satisfied (i.e., the required channel is already closed).
// This check is non‑blocking.
// func (k *DependencyKordinator) dependenciesReady(crd orktypes.CRDEntry, nameToGVK map[string]string) bool {
// 	for depName, depCond := range crd.DependsOn {
// 		depGVK, ok := nameToGVK[depName]
// 		if !ok {
// 			logger.Error().Str("crd", crd.Name).Str("dependency", depName).Msg("dependency GVK not found")
// 			return false
// 		}
// 		switch strings.ToLower(depCond.Condition) {
// 		case string(orktypes.DependencyConditionHealthy):
// 			select {
// 			case <-k.healthyCh[depGVK]:
// 				// channel closed → dependency healthy
// 			default:
// 				return false
// 			}
// 		default: // started
// 			select {
// 			case <-k.startedCh[depGVK]:
// 				// channel closed → dependency started
// 			default:
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// retryMissingCRDs runs continuously to detect and activate CRDs that were missing at startup
// or deferred because dependencies were not ready.
func (k *DependencyKordinator) retryMissingCRDs(ctx context.Context) {
	ticker := time.NewTicker(PostStartRetryInterval)
	defer ticker.Stop()

	backoff := PostStartBackoff
	nameToGVK := k.NameToGVKMap()

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
					logger.Warn().Str("gvk", gvkStr).Msg("CRD disappeared at runtime — marking missing")
					entry.Missing = true
					runtimeMissing[gvkStr] = entry
					k.informerFactory.SetMissingOnStartup(runtimeMissing)
					k.crdHealthMap[gvkStr].SetMissingAtRuntime()
					k.crdHealthMap[gvkStr].SetDegraded()
					k.orkHealth.SetKatalogDegraded()

					// Degrade dependents that require healthy condition
					deps := k.depGraph.GetDependents(entry.Name)
					for _, dep := range deps {
						crd := k.depGraph.GetNode(dep).CRD
						if crd.DependsOn[entry.Name].Condition == string(orktypes.DependencyConditionHealthy) {
							k.crdHealthMap[crd.GVK().String()].SetDegraded()
						} else {
							logger.Info().Str("gvk", gvkStr).Msgf("dependency %s is unhealthy", entry.Name)
						}
					}

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
				backoff = PostStartBackoff
				logger.Debug().Msg("retry loop: no missing CRDs")
				// Continue to Phase 3 before marking all online
			} else {
				backoff = PostStartBackoff
				logger.Debug().Msgf("retry loop: checking %d missing CRD(s)", len(missing))
				stillMissing := make(map[string]*informer.InformerEntry)

				for gvkStr, entry := range missing {
					ok, _ := k.crdExists(entry.GVK)
					if ok {
						k.activateCRD(ctx, entry)
						k.deactivated[gvkStr] = false
						k.crdHealthMap[gvkStr].SetStarted()
					} else {
						stillMissing[gvkStr] = entry
						k.crdHealthMap[gvkStr].SetMissingAtRuntime()
						k.crdHealthMap[gvkStr].SetDegraded()
						k.orkHealth.SetKatalogDegraded()
						logger.Debug().Msgf("retry loop: %s still not available", gvkStr)
					}
				}
				k.informerFactory.SetMissingOnStartup(stillMissing)

				if len(stillMissing) > 0 {
					logger.Info().Msgf("retry loop: %d CRD(s) still missing", len(stillMissing))
					k.allOnline.Store(false)
					k.orkHealth.SetKatalogDegraded()
				}
			}

			// ───────────────────────────────────────────────────────────
			// 3. Activate deferred CRDs (skipped at startup because
			//    dependencies were not ready)
			// ───────────────────────────────────────────────────────────
			for _, name := range k.depGraph.StartupOrder() {
				gvk := nameToGVK[name]

				k.mu.RLock()
				started := k.started[gvk]
				k.mu.RUnlock()
				if started {
					continue // already running
				}

				// Skip if it's already in the missing map (handled by Phase 2)
				if k.informerFactory.IsMissing(gvk) {
					continue
				}

				crd := k.depGraph.GetNode(name).CRD
				if !k.dependenciesReady(crd, nameToGVK) {
					continue // still waiting for dependencies
				}

				// CRD exists and dependencies are ready → activate
				logger.Info().Str("crd", name).Msg("dependencies now satisfied, activating")
				entry, ok := k.informerFactory.Registered()[gvk]
				if !ok || entry == nil {
					logger.Error().Str("crd", name).Str("gvk", gvk).Msg("no informer entry found for deferred CRD")
					continue
				}
				k.activateCRD(ctx, entry)
				k.deactivated[gvk] = false
				k.crdHealthMap[gvk].SetStarted()
			}

			// ───────────────────────────────────────────────
			// 4. Update overall health
			// ───────────────────────────────────────────────
			if len(k.informerFactory.Missing()) == 0 && k.allCRDsStarted() {
				k.allOnline.Store(true)
				k.orkHealth.SetKatalogReady()
				logger.Info().Msg("retry loop: all CRDs active")
			}

			// Exponential backoff only when CRDs are still missing
			if len(k.informerFactory.Missing()) > 0 {
				time.Sleep(backoff)
				if backoff < time.Minute {
					backoff *= 2
				}
			}
		}
	}
}

// allCRDsStarted returns true if every CRD in the graph has started.
func (k *DependencyKordinator) allCRDsStarted() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, node := range k.depGraph.GetNodes() {
		gvk := node.CRD.GroupVersionKind.String()
		if !k.started[gvk] {
			return false
		}
	}
	return true
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
func (k *DependencyKordinator) activateCRD(ctx context.Context, entry *informer.InformerEntry) {
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
	k.crdHealthMap[gvkStr].SetTotalWorkers(int32(workers))

	// Signal dependents that this CRD has now started
	// The started channel was created during Kordinate and may still be open
	if ch, exists := k.startedCh[gvkStr]; exists {
		select {
		case <-ch:
			// Channel already closed (should not happen, but safe)
			logger.Debug().Msgf("activateCRD: started channel for %q was already closed", name)
		default:
			close(ch)
			k.crdHealthMap[gvkStr].SetStarted()
			deps := k.depGraph.GetDependents(name)
			if len(deps) > 0 {
				logger.Info().Msgf("activateCRD: closed started channel for %q — unblocking %d dependent(s): %s",
					name, len(deps), strings.Join(deps, ", "))
			}
		}
	}

	// Check dependencies whose dependency condition is 'healthy' and leave healthy channel open
	deps := k.depGraph.GetDependents(name)
	if len(deps) > 0 {
		for _, dep := range deps {
			crd := k.NameToCRD(dep)
			if crd.DependsOn.ConditionHealthy(name) {
				if ch, exists := k.healthyCh[crd.GVK().String()]; exists {
					select {
					case <-ch:
						// Channel already closed (should not happen, but safe)
						logger.Debug().Msgf("activateCRD: healthy channel for %q was already closed", name)
					default:
						if !k.crdHealthMap[crd.GVK().String()].IsHealthy() {
							continue
						}
						close(ch)
						k.crdHealthMap[crd.GVK().String()].SetStarted()
						logger.Info().Msgf("activateCRD: closed healthy channel for %q", name)
					}
				}
			}
		}
	}

	logger.Info().Msgf("CRD %s activated", name)
}

// deactivateCRD — drain without permanent shutdown
func (k *DependencyKordinator) deactivateCRD(gvk string) {
	k.mu.RLock()
	cancel, okCancel := k.cancelFuncs[gvk]
	wg, okWG := k.wgs[gvk]
	k.mu.RUnlock()

	if okCancel {
		cancel() // signals workers via context
	}

	// Add a single sentinel item to each worker's queue to unblock
	// any worker currently waiting in GetWithContext.
	// Workers check ctx.Done() after dequeuing — sentinel is dropped.
	if wq, ok := k.queueReg.For(gvk); ok {
		numWorkers := k.katalog.GetWorkers(gvk, k.defaultWorkers)
		for i := 0; i < numWorkers; i++ {
			wq.Queue.Add(queue.QueueItem{GVK: gvk, Key: drainSentinel})
		}
	}

	if !okWG {
		return
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		k.crdHealthMap[gvk].ResetWorkerCounts()
		k.crdHealthMap[gvk].workerStates.Range(func(key, value interface{}) bool {
			k.crdHealthMap[gvk].workerStates.Store(key, WorkerStateStopped)
			return true
		})
		k.deactivated[gvk] = true
		logger.Info().Str("gvk", gvk).Msg("workers drained cleanly")
	case <-time.After(drainTimeout):
		logger.Warn().Str("gvk", gvk).Msg("drain timeout exceeded")
	}
}

// crdExists checks if a CRD is present in the cluster by querying the API server.
// Returns (true, nil) if the CRD exists, (false, nil) if not, (false, error) on failure.
func (k *DependencyKordinator) crdExists(gvk *schema.GroupVersionKind) (bool, error) {
	return utils.WaitForCRD(
		k.kube.RestConfig(),
		gvk.Group,
		gvk.Kind,
		gvk.Version,
	) == nil, nil
}
