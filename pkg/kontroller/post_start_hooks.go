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
//	This is an acceptable tradeoff for v1; CRD deletion is a rare administrative action.
func (k *DependencyKontroller) retryMissingCRDs(ctx context.Context) {
	ticker := time.NewTicker(PostStartRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			missing := k.informerFactory.Missing()
			for _, entry := range missing {
				if ok, _ := k.crdExists(entry.GVK); ok {
					k.activateCRD(ctx, entry)
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
	go entry.Informer.Run(ctx.Done())
	logger.Info().Msgf("activateCRD: informer started for %s", name)

	// Mark as no longer missing so the retry loop won't keep processing it
	entry.Missing = false
	k.informerFactory.RemoveMissing(gvkStr)

	// Start the worker pool for this CRD
	workers := k.katalog.GetWorkers(gvkStr, k.defaultWorkers)
	k.startCRDWorkers(ctx, gvkStr, workers)

	// Update health tracking
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
