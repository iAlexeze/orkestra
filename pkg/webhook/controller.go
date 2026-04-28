// webhook/controller.go — continuous webhook configuration reconciliation.
//
// The webhook controller runs in the background and ensures that Orkestra's
// admission surface (validation, mutation, deletion protection, namespace protection)
// exists exactly when the user has declared it — and disappears when they have not.
//
// Declarative Webhook Lifecycle
//
//	security.admission.enabled: true
//	    → create/update the validating and mutating webhook configurations
//
//	security.deletionProtection.enabled: true
//	    → create/update the deletion-protection webhook
//
//	(inverse: when disabled, the controller removes the corresponding webhook)
//
// # Cleanup Semantics
//
// Webhook configurations are not removed on pod restart by default. Cleanup is
// opt-in via security.admission.cleanupOnShutdown and
// security.deletionProtection.cleanupOnShutdown. This avoids race conditions
// during rollout where a new leader sees an existing webhook while the old leader
// is still shutting down.
//
// The controller returns only fatal initialization errors. Reconciliation itself
// is continuous and non-blocking.
package webhook

import (
	"context"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/metrics"
)

const (
	highTimeout = 30 * time.Second
	lowTimeout  = 10 * time.Second
)

// webhookController continuously reconciles all admission webhook configurations
// declared by the Katalog. Runs in a background goroutine until ctx is cancelled.
// Called once on leader startup. Returns only fatal initialization errors.
func (ws *WebhookServer) webhookController(ctx context.Context) error {
	if ws.kubeClient == nil || ws.katalog == nil {
		logger.Debug().Msg("webhook controller disabled: kube client or katalog not set")
		return nil
	}

	kat := ws.katalog
	hasAdmission := kat.HasValidationRules() || kat.HasMutationRules()
	hasDeletionProtection := kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil
	hasNamespaceProtection := kat.IsNamespaceProtectionEnabled() && len(kat.NamespaceProtectionGVRs()) > 0

	if !hasAdmission && !hasDeletionProtection && !hasNamespaceProtection {
		logger.Debug().Msg("webhook controller disabled: no admission or protection declared")
		return nil
	}

	go func() {
		ticker := time.NewTicker(kat.WebhookControllerSyncInterval())
		defer ticker.Stop()

		for {
			if ws.webhookStats != nil {
				ws.webhookStats.RecordReconciled()
			}
			metrics.RecordWebhookReconciled("controller")

			ws.reconcileAdmissionWebhooks()
			ws.reconcileDeletionProtectionWebhook()
			ws.reconcileNamespaceProtectionWebhook()

			select {
			case <-ctx.Done():
				logger.Debug().Msg("webhook controller stopping")
				return
			case <-ticker.C:
			}
		}
	}()

	logger.Info().Msg("webhook controller started")
	return nil
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
				logger.Error().Err(err).Msg("webhook controller: admission cleanup failed")
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
			logger.Error().Err(err).Msg("webhook controller: admission registration failed")
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
				logger.Debug().Err(err).Msg("webhook controller: deletion protection cleanup skipped or failed")
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
		logger.Error().Err(err).Msg("webhook controller: cannot read CA bundle for deletion protection")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("deletion-protection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
	defer cancel()

	if err := registerDeletionProtectionWebhook(ctx, ws.kubeClient, dpGVRs, caBundle, ws.hookReg); err != nil {
		logger.Error().Err(err).Msg("webhook controller: deletion protection registration failed")
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
				logger.Debug().Err(err).Msg("webhook controller: namespace protection cleanup skipped or failed")
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
		logger.Error().Err(err).Msg("webhook controller: cannot read CA bundle for namespace protection")
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
		logger.Error().Err(err).Msg("webhook controller: namespace protection registration failed")
		if ws.webhookStats != nil {
			ws.webhookStats.RecordFailure()
		}
		metrics.RecordWebhookReconciliationFailure("namespace-protection")
	}
}
