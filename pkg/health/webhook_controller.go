// pkg/health/webhook_controller.go
package health

import (
	"context"
	"time"

	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
)

// webhookController continuously reconciles all admission webhook configurations
// declared by the Katalog. This is the central controller responsible for ensuring
// that Orkestra’s admission surface (validation, mutation, deletion protection)
// exists exactly when the user has declared it — and disappears when they have not.
//
// ───────────────────────────────────────────────────────────────────────────────
//  Declarative Webhook Lifecycle
// ───────────────────────────────────────────────────────────────────────────────
//
// Orkestra does not install static webhook manifests. Instead, the Katalog is the
// single source of truth:
//
//   security.admission.enabled: true
//       → create/update the validating and mutating webhook configurations
//
//   security.deletionProtection.enabled: true
//       → create/update the deletion‑protection webhook
//
//   (…and the inverse: when disabled, the controller removes them)
//
// This ensures the API server only calls Orkestra for capabilities the user has
// explicitly declared.
//
// ───────────────────────────────────────────────────────────────────────────────
//  Cleanup Semantics (Runtime That Leaves No Trace)
// ───────────────────────────────────────────────────────────────────────────────
//
// Kubernetes users are not accustomed to operators removing their own webhook
// configurations on shutdown. To avoid surprises, Orkestra makes this behavior
// explicit and opt‑in:
//
//   security.admission.cleanupOnShutdown: true
//   security.deletionProtection.cleanupOnShutdown: true
//
// When enabled, Orkestra removes its webhook configurations during shutdown,
// preserving the “runtime that leaves no trace” principle.
//
// When disabled (the default), webhook lifecycle is driven purely by the Katalog
// and reconciled here — not tied to pod lifecycle. This avoids race conditions
// during rollout where a new leader observes an existing webhook while the old
// leader is still shutting down.
//
// ───────────────────────────────────────────────────────────────────────────────
//  Controller Responsibility
// ───────────────────────────────────────────────────────────────────────────────
//
// This controller runs only on the elected leader and:
//
//   • Ensures required webhook configurations exist with the correct spec
//   • Removes webhook configurations when the Katalog disables them
//   • Periodically re‑checks for drift or accidental deletion
//   • Decouples webhook lifecycle from pod lifecycle
//
// This makes webhook management predictable, declarative, and aligned with the
// Katalog — while still allowing the runtime to vanish cleanly when configured.
//
// ───────────────────────────────────────────────────────────────────────────────
//
// The controller returns only fatal initialization errors. Reconciliation itself
// is continuous and non‑blocking.

// webhookController continuously reconciles all admission webhook configurations
// declared by the Katalog. See the header comment above for the full model.
//
// This method should be called once on leader startup. It returns only fatal
// initialization errors; reconciliation itself runs in a background goroutine.

const (
	highTimeout = 30 * time.Second
	lowTimeout  = 10 * time.Second
)

func (h *HealthServer) webhookController() error {
	// If we don't have a kube client or katalog, there's nothing to reconcile.
	if h.kubeClient == nil || h.katalog == nil {
		logger.Debug().Msg("webhook controller disabled: kube client or katalog not set")
		return nil
	}

	// If neither admission nor deletion protection is relevant, skip entirely.
	kat := h.katalog
	hasAdmission := h.hookKfg.WebhooksEnabled && h.admissionRegistry != nil &&
		(kat.HasValidationRules() || kat.HasMutationRules())
	hasDeletionProtection := kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil

	if !hasAdmission && !hasDeletionProtection {
		logger.Debug().Msg("webhook controller disabled: no admission or deletion protection declared")
		return nil
	}

	// Reconcile loop: best-effort, periodic, leader-only by construction of HealthServer usage.
	go func() {
		ticker := time.NewTicker(highTimeout)
		defer ticker.Stop()

		for {
			// If the server is no longer healthy, stop reconciling.
			if !h.healthy.Load() {
				logger.Debug().Msg("webhook controller stopping: health server not healthy")
				return
			}

			// UI stats heartbeat
			if h.webhookStats != nil {
				h.webhookStats.RecordReconciled()
			}

			// Prometheus metric: one reconciliation cycle
			metrics.RecordWebhookReconciled("controller")

			h.reconcileAdmissionWebhooks()
			h.reconcileDeletionProtectionWebhook()

			<-ticker.C
		}
	}()

	logger.Info().Msg("webhook controller started")
	return nil
}

// reconcileAdmissionWebhooks ensures validating/mutating webhook configurations
// match the current Katalog + admission registry state.
func (h *HealthServer) reconcileAdmissionWebhooks() {
	kat := h.katalog

	// If webhooks are not enabled at all, ensure they are removed.
	if !h.hookKfg.WebhooksEnabled || h.admissionRegistry == nil {
		// Best-effort cleanup when disabled via env/Katalog.
		cleanupOpts := WebhookCleanupOptions{}
		if kat.HasMutationRules() {
			cleanupOpts.mutating = true
		}
		if kat.HasValidationRules() {
			cleanupOpts.validating = true
		}
		if (cleanupOpts.mutating || cleanupOpts.validating) && h.kubeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), lowTimeout)
			defer cancel()
			if err := UnregisterWebhooks(ctx, h.kubeClient, cleanupOpts); err != nil {
				logger.Error().Err(err).Msg("webhook controller: admission webhook cleanup failed")

				// UI stats
				if h.webhookStats != nil {
					h.webhookStats.RecordFailure()
				}

				// Prometheus metric
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

	// If there are no rules, nothing to register.
	if !kat.HasValidationRules() && !kat.HasMutationRules() {
		return
	}

	// Register or refresh admission webhooks. RegisterWebhooks is expected to be idempotent.
	if h.kubeClient != nil && h.admissionRegistry != nil {
		ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
		defer cancel()
		if err := RegisterWebhooks(ctx, h.kubeClient, h.admissionRegistry, h.hookReg); err != nil {
			logger.Error().Err(err).
				Msg("webhook controller: admission webhook registration failed — admission interception may not work")

			// UI stats
			if h.webhookStats != nil {
				h.webhookStats.RecordFailure()
			}

			// Prometheus metric
			if kat.HasValidationRules() {
				metrics.RecordWebhookReconciliationFailure("validation")
			}

			if kat.HasMutationRules() {
				metrics.RecordWebhookReconciliationFailure("mutation")
			}
		}
	}
}

// reconcileDeletionProtectionWebhook ensures the deletion protection webhook
// matches the current Katalog state.
func (h *HealthServer) reconcileDeletionProtectionWebhook() {
	kat := h.katalog

	enabled := kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil
	if !enabled {
		// When deletion protection is disabled in the Katalog, ensure the webhook is removed.
		if h.kubeClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), lowTimeout)
			defer cancel()
			if err := cleanupValidatingWebhook(ctx, h.kubeClient, deletionProtectionWebhookConfigName); err != nil {
				// Best-effort: log and continue.
				logger.Debug().Err(err).Msg("webhook controller: deletion protection webhook cleanup skipped or failed")

				// UI stats
				if h.webhookStats != nil {
					h.webhookStats.RecordFailure()
				}

				// Prometheus metric
				metrics.RecordWebhookReconciliationFailure("deletion-protection")
			}
		}
		return
	}

	// Enabled: ensure the webhook exists with the correct spec.
	if h.kubeClient == nil {
		return
	}

	dpGVRs := kat.DeletionProtectionGVRs()
	if len(dpGVRs) == 0 {
		// Running outside cluster or no GVRs resolved — nothing to register.
		return
	}

	caBundle, err := readCABundle(h.hookReg.TLSCertFile)
	if err != nil {
		logger.Error().Err(err).Msg("webhook controller: cannot read CA bundle for deletion protection")

		// UI stats
		if h.webhookStats != nil {
			h.webhookStats.RecordFailure()
		}

		// Prometheus metric
		metrics.RecordWebhookReconciliationFailure("deletion-protection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), highTimeout)
	defer cancel()

	if err := registerDeletionProtectionWebhook(ctx, h.kubeClient, dpGVRs, caBundle, h.hookReg); err != nil {
		logger.Error().Err(err).
			Msg("webhook controller: deletion protection webhook registration failed — CRDs may not be protected")

		// UI stats
		if h.webhookStats != nil {
			h.webhookStats.RecordFailure()
		}

		// Prometheus metric
		metrics.RecordWebhookReconciliationFailure("deletion-protection")
	}
}
