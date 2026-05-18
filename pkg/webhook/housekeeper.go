// webhook/housekeeper.go — self-healing maintenance of Orkestra's admission surface.
//
// The housekeeper runs in the background and ensures that the webhook
// configurations Orkestra owns exist exactly when the Katalog declares them —
// and disappear when they do not.
//
// Declarative Webhook Lifecycle
//
//	security.admission.enabled: true
//	    → create/update the validating and mutating webhook configurations
//
//	security.deletionProtection.enabled: true
//	    → create/update the deletion-protection webhook
//
//	(inverse: when disabled, the housekeeper removes the corresponding webhook)
//
// # Event-Driven Reconciliation
//
// The housekeeper uses a Kubernetes Watch on ValidatingWebhookConfiguration and
// MutatingWebhookConfiguration objects as the fast path. A DELETED event
// triggers an immediate reconcile — the window during which a webhook is absent
// is bounded only by the API server round-trip, not by a poll interval.
// MODIFIED events are intentionally ignored: every reconcile Update fires one,
// and reacting to it would create a tight loop. Content drift is caught by the
// safety ticker instead.
//
// A safety ticker (WEBHOOK_CONTROLLER_SYNC_INTERVAL, default 30 s) runs in
// parallel as a backstop for drift that Watch silently misses: partial mutations,
// silent stream drops on some managed clusters, and token expiry.
//
// A buffered trigger channel (capacity 1) coalesces bursts — any number of rapid
// DELETE/MODIFY events produce exactly one reconcile call.
//
// # Cleanup Semantics
//
// Webhook configurations are not removed on pod restart by default. Cleanup is
// opt-in via security.admission.cleanupOnShutdown and
// security.deletionProtection.cleanupOnShutdown. This avoids race conditions
// during rollout where a new leader sees an existing webhook while the old leader
// is still shutting down.
//
// The housekeeper returns only fatal initialization errors. Reconciliation itself
// is continuous and non-blocking.
package webhook

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
)

const (
	highTimeout = 30 * time.Second
	lowTimeout  = 10 * time.Second

	// watchRetryDelay is the pause before re-establishing a dropped Watch stream.
	watchRetryDelay = 5 * time.Second

	// orkestraWebhookLabelSelector narrows the Watch to Orkestra-owned webhook
	// configurations — unrelated webhooks in the same cluster are ignored.
	orkestraWebhookLabelSelector = "app.kubernetes.io/tag=orkestra-internal"
)

// housekeeper continuously reconciles all admission webhook configurations
// declared by the Katalog. Runs background goroutines until ctx is cancelled.
// Called once on leader startup. Returns only fatal initialization errors.
func (ws *WebhookServer) housekeeper(ctx context.Context) error {
	if ws.kubeClient == nil || ws.katalog == nil {
		logger.Debug().Msg("housekeeper disabled: kube client or katalog not set")
		return nil
	}

	kat := ws.katalog
	hasAdmission := kat.HasValidationRules() || kat.HasMutationRules()
	hasDeletionProtection := kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil
	hasNamespaceProtection := kat.IsNamespaceProtectionEnabled() && len(kat.NamespaceProtectionGVRs()) > 0
	hasStrictMode := kat.IsStrictModeEnabled()

	if !hasAdmission && !hasDeletionProtection && !hasNamespaceProtection && !hasStrictMode {
		logger.Debug().Msg("housekeeper disabled: no admission or protection declared")
		return nil
	}

	// Buffered capacity 1: bursts of Watch events collapse into one reconcile.
	trigger := make(chan struct{}, 1)

	// Start housekeepers only for the features that require them.
	// Validation, deletion‑protection, namespace‑protection, and strict‑mode all use
	// ValidatingWebhookConfiguration, so they share the same watcher.
	// Mutation rules use a separate MutatingWebhookConfiguration watcher.
	if kat.HasValidationRules() || hasDeletionProtection || hasNamespaceProtection || hasStrictMode {
		go ws.watchValidatingWebhooks(ctx, trigger)
	}

	if kat.HasMutationRules() {
		go ws.watchMutatingWebhooks(ctx, trigger)
	}

	go func() {
		safetyTicker := time.NewTicker(kat.WebhookControllerSyncInterval())
		defer safetyTicker.Stop()

		// Reconcile immediately on startup — don't wait for the first tick or event.
		ws.reconcileAll()

		for {
			select {
			case <-ctx.Done():
				logger.Debug().Msg("housekeeper stopping")
				return
			case <-trigger:
				logger.Warn().Msg("housekeeper: change detected — reconciling")
				ws.reconcileAll()
			case <-safetyTicker.C:
				ws.reconcileAll()
			}
		}
	}()

	logger.Info().Msg("housekeeper started")
	return nil
}

// reconcileAll records metrics and drives all reconcile functions.
func (ws *WebhookServer) reconcileAll() {
	if ws.webhookStats != nil {
		ws.webhookStats.RecordReconciled()
	}
	metrics.RecordWebhookReconciled("housekeeper")

	ws.reconcileAdmissionWebhooks()
	ws.reconcileDeletionProtectionWebhook()
	ws.reconcileNamespaceProtectionWebhook()
	ws.reconcileStrictModeWebhook()
}

// watchValidatingWebhooks watches ValidatingWebhookConfiguration objects owned
// by Orkestra and sends to trigger on DELETED or MODIFIED events.
// Reconnects automatically when the stream expires or drops.
func (ws *WebhookServer) watchValidatingWebhooks(ctx context.Context, trigger chan<- struct{}) {
	for {
		watcher, err := ws.kubeClient.AdmissionregistrationV1().
			ValidatingWebhookConfigurations().
			Watch(ctx, metav1.ListOptions{
				LabelSelector: orkestraWebhookLabelSelector,
			})
		if err != nil {
			logger.Warn().Err(err).Msg("housekeeper watch (validating): failed to start, retrying")
			select {
			case <-ctx.Done():
				return
			case <-time.After(watchRetryDelay):
				continue
			}
		}

		ws.drainWatchEvents(ctx, watcher, trigger, "validating")
		watcher.Stop()

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// watchMutatingWebhooks watches MutatingWebhookConfiguration objects owned
// by Orkestra and sends to trigger on DELETED or MODIFIED events.
// Reconnects automatically when the stream expires or drops.
func (ws *WebhookServer) watchMutatingWebhooks(ctx context.Context, trigger chan<- struct{}) {
	for {
		watcher, err := ws.kubeClient.AdmissionregistrationV1().
			MutatingWebhookConfigurations().
			Watch(ctx, metav1.ListOptions{
				LabelSelector: orkestraWebhookLabelSelector,
			})
		if err != nil {
			logger.Warn().Err(err).Msg("housekeeper watch (mutating): failed to start, retrying")
			select {
			case <-ctx.Done():
				return
			case <-time.After(watchRetryDelay):
				continue
			}
		}

		ws.drainWatchEvents(ctx, watcher, trigger, "mutating")
		watcher.Stop()

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// drainWatchEvents reads events from a watcher until the channel closes or ctx
// is cancelled. Only DELETED events signal the trigger channel.
//
// MODIFIED is intentionally ignored. Every reconcile calls Update on the
// webhook configuration, which always fires a MODIFIED event — reacting to it
// would immediately re-trigger the reconcile that just finished, creating a
// tight reconcile→update→MODIFIED→reconcile loop. Content drift from external
// modifications is caught within one safety-ticker interval instead.
func (ws *WebhookServer) drainWatchEvents(ctx context.Context, w watch.Interface, trigger chan<- struct{}, kind string) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.ResultChan():
			if !ok {
				// Channel closed: Watch stream expired or network error. Reconnect.
				return
			}
			if event.Type == watch.Deleted {
				logger.Warn().Str("kind", kind).Msg("housekeeper: configuration deleted — triggering reconcile")
				select {
				case trigger <- struct{}{}:
				default: // already pending
				}
			}
		}
	}
}

func (ws *WebhookServer) reconcileAdmissionWebhooks() {
	kat := ws.katalog

	if !ws.webhooksEnabled || ws.admissionRegistry == nil {
		cleanupOpts := WebhookCleanupOptions{}
		if kat.HasMutationRules() {
			cleanupOpts.mutating = true
		}
		if kat.HasValidationRules() {
			cleanupOpts.validating = true
		}
		if (cleanupOpts.mutating || cleanupOpts.validating) && ws.kubeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), lowTimeout)
			defer cancel()
			if err := UnregisterAdmissionWebhooks(ctx, ws.kubeClient, cleanupOpts); err != nil {
				logger.Error().Err(err).Msg("housekeeper: admission cleanup failed")
				if ws.webhookStats != nil {
					ws.webhookStats.RecordFailure()
				}
				if cleanupOpts.validating {
					metrics.RecordWebhookReconciliationFailure("validation")
				}
				if cleanupOpts.mutating {
					metrics.RecordWebhookReconciliationFailure("mutation")
				}
			}
		}
		return
	}

	if !kat.HasValidationRules() && !kat.HasMutationRules() {
		return
	}

	if ws.kubeClient != nil && ws.admissionRegistry != nil {
		ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
		defer cancel()
		if err := RegisterAdmissionWebhooks(ctx, ws.kubeClient, ws.admissionRegistry, ws.hookReg); err != nil {
			logger.Error().Err(err).Msg("housekeeper: admission registration failed")
			if ws.webhookStats != nil {
				ws.webhookStats.RecordFailure()
			}
			if kat.HasValidationRules() {
				metrics.RecordWebhookReconciliationFailure("validation")
			}
			if kat.HasMutationRules() {
				metrics.RecordWebhookReconciliationFailure("mutation")
			}
		}
	}
}

func (ws *WebhookServer) reconcileDeletionProtectionWebhook() {
	kat := ws.katalog

	enabled := kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil
	if !enabled {
		if ws.kubeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), lowTimeout)
			defer cancel()
			if err := cleanupValidatingWebhook(ctx, ws.kubeClient, deletionProtectionWebhookConfigName); err != nil {
				logger.Debug().Err(err).Msg("housekeeper: deletion protection cleanup skipped or failed")
				if ws.webhookStats != nil {
					ws.webhookStats.RecordFailure()
				}
				metrics.RecordWebhookReconciliationFailure("deletion-protection")
			}
		}
		return
	}

	if ws.kubeClient == nil {
		return
	}

	dpGVRs := kat.DeletionProtectionGVRs()
	if len(dpGVRs) == 0 {
		return
	}

	caBundle, err := readCABundle(ws.hookReg.TLSCertFile)
	if err != nil {
		logger.Error().Err(err).Msg("housekeeper: cannot read CA bundle for deletion protection")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("deletion-protection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
	defer cancel()

	if err := registerDeletionProtectionWebhook(ctx, ws.kubeClient, dpGVRs, caBundle, ws.hookReg); err != nil {
		logger.Error().Err(err).Msg("housekeeper: deletion protection registration failed")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("deletion-protection")
	}
}

func (ws *WebhookServer) reconcileNamespaceProtectionWebhook() {
	kat := ws.katalog

	enabled := kat.IsNamespaceProtectionEnabled() && len(kat.NamespaceProtectionGVRs()) > 0
	if !enabled {
		if ws.kubeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), lowTimeout)
			defer cancel()
			if err := cleanupValidatingWebhook(ctx, ws.kubeClient, namespaceProtectionWebhookConfigName); err != nil {
				logger.Debug().Err(err).Msg("housekeeper: namespace protection cleanup skipped or failed")
				if ws.webhookStats != nil {
					ws.webhookStats.RecordFailure()
				}
				metrics.RecordWebhookReconciliationFailure("namespace-protection")
			}
		}
		return
	}

	if ws.kubeClient == nil {
		return
	}

	npGVRs := kat.NamespaceProtectionGVRs()
	if len(npGVRs) == 0 {
		return
	}

	caBundle, err := readCABundle(ws.hookReg.TLSCertFile)
	if err != nil {
		logger.Error().Err(err).Msg("housekeeper: cannot read CA bundle for namespace protection")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("namespace-protection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
	defer cancel()

	svcName := kat.NamespaceProtectionServiceName()
	failurePolicy := kat.NamespaceProtectionFailurePolicy()
	if err := registerNamespaceProtectionWebhook(ctx, ws.kubeClient, npGVRs, caBundle, ws.hookReg, svcName, failurePolicy); err != nil {
		logger.Error().Err(err).Msg("housekeeper: namespace protection registration failed")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("namespace-protection")
	}
}

func (ws *WebhookServer) reconcileStrictModeWebhook() {
	kat := ws.katalog

	if !kat.IsStrictModeEnabled() {
		if ws.kubeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), lowTimeout)
			defer cancel()
			if err := cleanupValidatingWebhook(ctx, ws.kubeClient, strictModeProtectionWebhookConfigName); err != nil {
				logger.Debug().Err(err).Msg("housekeeper: strict mode cleanup skipped or failed")
				if ws.webhookStats != nil {
					ws.webhookStats.RecordFailure()
				}
				metrics.RecordWebhookReconciliationFailure("strict-mode-protection")
			}
		}
		return
	}

	if ws.kubeClient == nil {
		return
	}

	caBundle, err := readCABundle(ws.hookReg.TLSCertFile)
	if err != nil {
		logger.Error().Err(err).Msg("housekeeper: cannot read CA bundle for strict mode protection")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("strict-mode-protection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
	defer cancel()

	if err := registerStrictModeProtectionWebhook(ctx, ws.kubeClient, caBundle, ws.hookReg); err != nil {
		logger.Error().Err(err).Msg("housekeeper: strict mode protection registration failed")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("strict-mode-protection")
	}
}
