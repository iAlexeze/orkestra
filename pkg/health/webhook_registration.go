// health/webhook_registration.go
package health

import (
	"context"
	"fmt"
	"os"
	"time"

	"strings"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ── Webhook configuration registration ────────────────────────────────────
//
// At startup, when ENABLE_ADMISSION_WEBHOOK=true, Orkestra creates or updates the
// ValidatingWebhookConfiguration and MutatingWebhookConfiguration objects
// that tell the API server to call Orkestra during admission.
//
// The configurations are built from the admission registry — only CRDs with
// declared validation or mutation rules are included. This ensures the API
// server only calls Orkestra for resources it actually has rules for.
//
// The CA bundle is read from the TLS_CERT file (same certificate used by
// the HTTPS server). The API server uses this to trust Orkestra's TLS endpoint.
//
// Configuration names:
//   orkestra-validation   → ValidatingWebhookConfiguration
//   orkestra-mutation     → MutatingWebhookConfiguration
//   orkestra-delete-protection → ValidatingWebhookConfiguration
//
// Both are created with FailurePolicy: Ignore during initial rollout.
// Platform operators who want hard enforcement can change this to Fail.

const (
	validatingWebhookConfigName         = "orkestra-validation"
	mutatingWebhookConfigName           = "orkestra-mutation"
	deletionProtectionWebhookConfigName = "orkestra-delete-protection"

	// Webhook Cleanup setup
	// MaxAttempts — number of retries for cleanup operations.
	maxAttempts = 5

	// Delay — time between retry attempts.
	delayBetweenAttempts = 5 * time.Second

	// The duration in seconds before the webhook should be deleted.
	gracePeriodSeconds = int64(30)
)

// orkestraResourceLabels defines the labels used to identify Orkestra-managed
// resources for deletion protection.
var orkestraResourceLabels = map[string]string{
	"app.kubernetes.io/name":      "orkestra",
	"app.kubernetes.io/component": "orkestra-internal",
}

// Label selector shared by all Orkestra-managed Kubernetes resources.
// Narrows the webhook to only the operator's own deployment, service, ingress,
// and admission webhook configurations (validation + mutation).
var orkestraResourceSelector = &metav1.LabelSelector{
	MatchLabels: orkestraResourceLabels,
}

// WebhookRegistrationOptions holds the configuration for webhook registration.
type WebhookRegistrationOptions struct {
	// ServiceName — the name of the Kubernetes Service fronting Orkestra.
	// The API server calls this service to reach /validate and /mutate.
	// Default: "orkestra"
	ServiceName string

	// ServiceNamespace — the namespace where the Orkestra Service lives.
	// Default: read from ORKESTRA_NAMESPACE environment variable.
	ServiceNamespace string

	// Port — the HTTPS port. Must match the conversion server port.
	// Default: 8443
	Port int32

	// FailurePolicy — what the API server does if Orkestra is unreachable.
	// admissionv1.Ignore (default): allow the operation and continue.
	// admissionv1.Fail: reject the operation if Orkestra cannot be reached.
	// Configurable from WEBHOOK_FAILURE_POLICY environment variable.
	FailurePolicy admissionv1.FailurePolicyType

	// TLSCertFile — path to the TLS certificate file.
	// The certificate is read and used as the caBundle in the webhook config.
	// Default: read from TLS_CERT environment variable.
	TLSCertFile string
}

// RegisterWebhooks creates or updates the ValidatingWebhookConfiguration and
// MutatingWebhookConfiguration based on the current admission registry.
//
// Called from HealthServer.Start() when ENABLE_ADMISSION_WEBHOOK=true, after the
// Katalog is fully loaded and the admission registry is populated.
//
// The function is idempotent — safe to call on restart. Existing configurations
// are updated to match the current Katalog state. CRDs removed from the Katalog
// are removed from the webhook configuration.
func RegisterWebhooks(
	ctx context.Context,
	client kubernetes.Interface,
	registry admissionRegistryReader,
	opts WebhookRegistrationOptions,
) error {
	// Read the CA bundle from the TLS certificate file
	caBundle, err := readCABundle(opts.TLSCertFile)
	if err != nil {
		return fmt.Errorf("webhook registration: reading CA bundle: %w", err)
	}

	// Register ValidatingWebhookConfiguration if there are validation rules
	valGVRs := registry.ValidationGVRs()
	if len(valGVRs) > 0 {
		if err := registerValidatingWebhook(ctx, client, valGVRs, caBundle, opts); err != nil {
			return fmt.Errorf("webhook registration: validating: %w", err)
		}
		logger.Info().
			Int("crds", len(valGVRs)).
			Str("config", validatingWebhookConfigName).
			Msg("webhook: ValidatingWebhookConfiguration registered")
	}

	// Register MutatingWebhookConfiguration if there are mutation rules
	mutGVRs := registry.MutationGVRs()
	if len(mutGVRs) > 0 {
		if err := registerMutatingWebhook(ctx, client, mutGVRs, caBundle, opts); err != nil {
			return fmt.Errorf("webhook registration: mutating: %w", err)
		}
		logger.Info().
			Int("crds", len(mutGVRs)).
			Str("config", mutatingWebhookConfigName).
			Msg("webhook: MutatingWebhookConfiguration registered")
	}

	return nil
}

// UnregisterWebhooks removes the ValidatingWebhookConfiguration and
// MutatingWebhookConfiguration entries that were previously created from the
// admission registry.
//
// Called from HealthServer.Shutdown() when ENABLE_ADMISSION_WEBHOOK=true, after the
// runtime begins shutting down and the admission registry is no longer needed.
//
// The function is destructive — only call during shutdown. Any
// webhook configurations created by Orkestra are cleaned up.
type WebhookCleanupOptions struct {
	mutating   bool
	validating bool
}

func UnregisterWebhooks(
	ctx context.Context,
	client kubernetes.Interface,
	opts WebhookCleanupOptions,
) error {

	// Cleanup ValidatingWebhookConfiguration with retry
	if opts.validating {
		if err := utils.RetryBackoff(func() error {
			return cleanupValidatingWebhook(ctx, client, validatingWebhookConfigName)
		}, utils.RetryOptions{
			Attempts: maxAttempts,
			Delay:    delayBetweenAttempts,
		},
		); err != nil {
			return fmt.Errorf("webhook cleanup: validating: %w", err)
		}
		logger.Info().
			Str("config", validatingWebhookConfigName).
			Msg("webhook: ValidatingWebhookConfiguration unregistered")
	}

	// Cleanup MutatingWebhookConfiguration with retry
	if opts.mutating {
		if err := utils.RetryBackoff(func() error {
			return cleanupMutatingWebhook(ctx, client, mutatingWebhookConfigName)
		}, utils.RetryOptions{
			Attempts: maxAttempts,
			Delay:    delayBetweenAttempts,
		}); err != nil {
			return fmt.Errorf("webhook cleanup: mutating: %w", err)
		}

		logger.Info().
			Str("config", mutatingWebhookConfigName).
			Msg("webhook: MutatingWebhookConfiguration unregistered")

	}
	return nil
}

// registerValidatingWebhook creates or updates the ValidatingWebhookConfiguration.
func registerValidatingWebhook(
	ctx context.Context,
	client kubernetes.Interface,
	gvrs []katalog.GVREntry,
	caBundle []byte,
	opts WebhookRegistrationOptions,
) error {
	sideEffects := admissionv1.SideEffectClassNone
	path := "/validate"
	port := opts.Port

	config := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   validatingWebhookConfigName,
			Labels: orkestraResourceLabels, // Add orkestra labels for deletion protection
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				// One webhook per CRD in the registry.
				// The webhook name must be a fully-qualified domain name.
				Name: "validate.orkestra.konductor.io",
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      opts.ServiceName,
						Namespace: opts.ServiceNamespace,
						Path:      &path,
						Port:      &port,
					},
					CABundle: caBundle,
				},
				// Build admission rules from the registered GVRs
				Rules: buildAdmissionRules(gvrs),
				// FailurePolicy: if Orkestra is unreachable, use the configured policy
				FailurePolicy: &opts.FailurePolicy,
				// Match policy: apply to the exact GVR, not equivalent resources
				MatchPolicy: matchPolicyPtr(admissionv1.Exact),
				// AdmissionReviewVersions: which versions we support
				AdmissionReviewVersions: []string{"v1"},
				// SideEffects: validation has no side effects
				SideEffects: &sideEffects,
				// TimeoutSeconds: 5 seconds is the Kubernetes default — sufficient
				// for in-memory rule evaluation
				TimeoutSeconds: int32Ptr(5),
			},
		},
	}

	return applyWebhookConfig(ctx, client, config)
}

// registerMutatingWebhook creates or updates the MutatingWebhookConfiguration.
func registerMutatingWebhook(
	ctx context.Context,
	client kubernetes.Interface,
	gvrs []katalog.GVREntry,
	caBundle []byte,
	opts WebhookRegistrationOptions,
) error {
	sideEffects := admissionv1.SideEffectClassNone
	path := "/mutate"
	port := opts.Port

	config := &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   mutatingWebhookConfigName,
			Labels: orkestraResourceLabels, // Add orkestra labels for deletion protection
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				Name: "mutate.orkestra.konductor.io",
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      opts.ServiceName,
						Namespace: opts.ServiceNamespace,
						Path:      &path,
						Port:      &port,
					},
					CABundle: caBundle,
				},
				Rules:                   buildAdmissionRules(gvrs),
				FailurePolicy:           &opts.FailurePolicy,
				MatchPolicy:             matchPolicyPtr(admissionv1.Exact),
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				TimeoutSeconds:          int32Ptr(5),
				// ReinvocationPolicy: IfNeeded — re-invoke if another webhook
				// modifies the object after ours. Ensures our defaults are applied
				// even if another webhook changes the object first.
				ReinvocationPolicy: reinvocationPolicyPtr(admissionv1.IfNeededReinvocationPolicy),
			},
		},
	}

	return applyMutatingWebhookConfig(ctx, client, config)
}

// registerDeletionProtectionWebhook creates or updates the ValidatingWebhookConfiguration for deletion protection.
//
// Two webhook entries are registered within the same configuration:
//
//  1. CRD protection — intercepts DELETE on customresourcedefinitions.
//     No ObjectSelector: the handler narrows to managed CRDs via ProtectedCRDNames().
//
//  2. Orkestra resource protection — intercepts DELETE on deployments, services, ingresses,
//     and now Orkestra’s own admission webhooks (validating + mutating).
//     ObjectSelector narrows to Orkestra-owned resources only, so the webhook never
//     intercepts unrelated cluster resources.
//
//     This ensures:
//     • The Orkestra operator cannot be deleted while deletion protection is enabled
//     • The Orkestra Service and Ingress cannot be deleted
//     • The ValidatingWebhookConfiguration and MutatingWebhookConfiguration used for
//     admission (validation + mutation) cannot be deleted
//
//     Protecting the admission webhooks themselves closes the loop: deletion protection
//     guarantees that the admission surface cannot be disabled by removing the webhook
//     configuration. This makes the system self-protecting and fully declarative.
func registerDeletionProtectionWebhook(
	ctx context.Context,
	client kubernetes.Interface,
	gvrs []katalog.GVREntry,
	caBundle []byte,
	opts WebhookRegistrationOptions,
) error {
	sideEffects := admissionv1.SideEffectClassNone
	path := "/deletion-protection"
	port := opts.Port

	// Split GVRs into two groups:
	//   crdGVRs      — customresourcedefinitions; no ObjectSelector, handler filters by name
	//   orkestraGVRs — deployment, service, ingress, validatingwebhookconfigurations,
	//                   mutatingwebhookconfigurations; ObjectSelector narrows to Orkestra resources only
	//
	// The admission webhook GVRs (validating + mutating) are included here so that
	// deletion protection also shields Orkestra’s own admission webhooks from deletion.
	var crdGVRs, orkestraGVRs []katalog.GVREntry
	for _, gvr := range gvrs {
		if gvr.Resource == "customresourcedefinitions" {
			crdGVRs = append(crdGVRs, gvr)
		} else {
			// Includes deployments, services, ingresses, validatingwebhookconfigurations,
			// mutatingwebhookconfigurations — all protected via ObjectSelector.
			orkestraGVRs = append(orkestraGVRs, gvr)
		}
	}

	webhooks := make([]admissionv1.ValidatingWebhook, 0, 3)

	// Webhook 1: CRD deletions — handler filters by ProtectedCRDNames()
	if len(crdGVRs) > 0 {
		webhooks = append(webhooks, admissionv1.ValidatingWebhook{
			Name: "protect.crds.orkestra.konductor.io",
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      opts.ServiceName,
					Namespace: opts.ServiceNamespace,
					Path:      &path,
					Port:      &port,
				},
				CABundle: caBundle,
			},
			Rules:                   buildDeletionProtectionRules(crdGVRs),
			FailurePolicy:           failurePolicyPtr(admissionv1.Fail),
			MatchPolicy:             matchPolicyPtr(admissionv1.Exact),
			AdmissionReviewVersions: []string{"v1"},
			SideEffects:             &sideEffects,
			TimeoutSeconds:          int32Ptr(5),
		})
	}

	// Webhook 2: Orkestra resource deletions (deployment, service, ingress, validatingwebhookconfigurations,
	// mutatingwebhookconfigurations).
	//
	// ObjectSelector ensures only resources carrying the Orkestra labels are intercepted.
	// This includes the admission webhooks themselves, making them deletion-protected.
	//
	// Handler always blocks — if the webhook fired, the ObjectSelector already confirmed ownership.
	if len(orkestraGVRs) > 0 {
		webhooks = append(webhooks, admissionv1.ValidatingWebhook{
			Name: "protect.resources.orkestra.konductor.io",
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      opts.ServiceName,
					Namespace: opts.ServiceNamespace,
					Path:      &path,
					Port:      &port,
				},
				CABundle: caBundle,
			},
			ObjectSelector:          orkestraResourceSelector,
			Rules:                   buildDeletionProtectionRules(orkestraGVRs),
			FailurePolicy:           failurePolicyPtr(admissionv1.Fail),
			MatchPolicy:             matchPolicyPtr(admissionv1.Exact),
			AdmissionReviewVersions: []string{"v1"},
			SideEffects:             &sideEffects,
			TimeoutSeconds:          int32Ptr(5),
		})
	}

	// Webhook 3: Self-protection - block deletion of the deletion-protection webhook itself.
	// This closes the bootstrap gap: the webhook must protect itself before it can be deleted.
	if true {
		scopeCluster := admissionv1.ClusterScope

		webhooks = append(webhooks, admissionv1.ValidatingWebhook{
			Name: "protect.self.orkestra.konductor.io",
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      opts.ServiceName,
					Namespace: opts.ServiceNamespace,
					Path:      &path,
					Port:      &port,
				},
				CABundle: caBundle,
			},
			ObjectSelector: orkestraResourceSelector,
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: []admissionv1.OperationType{
						admissionv1.Delete,
					},
					Rule: admissionv1.Rule{
						APIGroups:   []string{"admissionregistration.k8s.io"},
						APIVersions: []string{"v1"},
						Resources:   []string{"validatingwebhookconfigurations"},
						Scope:       &scopeCluster,
					},
				},
			},
			FailurePolicy:           failurePolicyPtr(admissionv1.Fail),
			MatchPolicy:             matchPolicyPtr(admissionv1.Exact),
			AdmissionReviewVersions: []string{"v1"},
			SideEffects:             &sideEffects,
			TimeoutSeconds:          int32Ptr(5),
		})
	}

	config := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deletionProtectionWebhookConfigName,
			Labels: orkestraResourceLabels, // Add orkestra labels for deletion protection
		},
		Webhooks: webhooks,
	}

	return applyWebhookConfig(ctx, client, config)
}

// buildAdmissionRules converts GVREntry slices to Kubernetes RuleWithOperations.
// Each GVREntry becomes one rule covering the specific group/version/resource.
func buildAdmissionRules(gvrs []katalog.GVREntry) []admissionv1.RuleWithOperations {
	rules := make([]admissionv1.RuleWithOperations, 0, len(gvrs))
	for _, gvr := range gvrs {
		ops := make([]admissionv1.OperationType, 0, len(gvr.Operations))
		for _, op := range gvr.Operations {
			ops = append(ops, admissionv1.OperationType(op))
		}
		rules = append(rules, admissionv1.RuleWithOperations{
			Operations: ops,
			Rule: admissionv1.Rule{
				APIGroups:   []string{gvr.Group},
				APIVersions: []string{gvr.Version},
				Resources:   []string{gvr.Resource},
			},
		})
	}
	return rules
}

// buildDeletionProtectionRules mirrors buildAdmissionRules but for deletion protection.
func buildDeletionProtectionRules(gvrs []katalog.GVREntry) []admissionv1.RuleWithOperations {
	rules := make([]admissionv1.RuleWithOperations, 0, len(gvrs))
	for _, gvr := range gvrs {
		ops := make([]admissionv1.OperationType, 0, len(gvr.Operations))
		for _, op := range gvr.Operations {
			ops = append(ops, admissionv1.OperationType(op))
		}
		rules = append(rules, admissionv1.RuleWithOperations{
			Operations: ops,
			Rule: admissionv1.Rule{
				APIGroups:   []string{gvr.Group},
				APIVersions: []string{gvr.Version},
				Resources:   []string{gvr.Resource},
			},
		})
	}
	return rules
}

// applyWebhookConfig creates or updates a ValidatingWebhookConfiguration.
// Uses server-side apply semantics: create if not exists, update if exists.
func applyWebhookConfig(ctx context.Context, client kubernetes.Interface, cfg *admissionv1.ValidatingWebhookConfiguration) error {
	existing, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, cfg.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create if not exists
		_, err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, cfg, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	// Update if existing
	cfg.ResourceVersion = existing.ResourceVersion
	_, err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(ctx, cfg, metav1.UpdateOptions{})
	return err
}

// applyMutatingWebhookConfig creates or updates a MutatingWebhookConfiguration.
func applyMutatingWebhookConfig(ctx context.Context, client kubernetes.Interface, cfg *admissionv1.MutatingWebhookConfiguration) error {
	existing, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, cfg.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create if not exists
		_, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, cfg, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	// Update if existing
	cfg.ResourceVersion = existing.ResourceVersion
	_, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(ctx, cfg, metav1.UpdateOptions{})
	return err
}

// cleanupValidatingWebhook deletes the ValidatingWebhookConfiguration.
func cleanupMutatingWebhook(ctx context.Context, client kubernetes.Interface, cfgName string) error {
	_, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, cfgName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Delete if exisiting
	return client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, cfgName,
		metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(gracePeriodSeconds)})
}

// cleanupMutatingWebhook deletes the MutatingWebhookConfiguration.
func cleanupValidatingWebhook(ctx context.Context, client kubernetes.Interface, cfgName string) error {
	_, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, cfgName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Delete if existing
	return client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, cfgName,
		metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(gracePeriodSeconds)})
}

// readCABundle reads the TLS certificate file and returns it as raw bytes.
// The API server requires this to trust Orkestra's TLS endpoint.
func readCABundle(certFile string) ([]byte, error) {
	if certFile == "" {
		return nil, fmt.Errorf("TLS_CERT is required for webhook registration — " +
			"set TLS_CERT to the path of the serving certificate")
	}
	data, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", certFile, err)
	}
	return data, nil
}

// ── Pointer helpers ───────────────────────────────────────────────────────

func int32Ptr(i int32) *int32                                                         { return &i }
func int64Ptr(i int64) *int64                                                         { return &i }
func matchPolicyPtr(p admissionv1.MatchPolicyType) *admissionv1.MatchPolicyType       { return &p }
func failurePolicyPtr(p admissionv1.FailurePolicyType) *admissionv1.FailurePolicyType { return &p }
func reinvocationPolicyPtr(p admissionv1.ReinvocationPolicyType) *admissionv1.ReinvocationPolicyType {
	return &p
}

// admissionv1FailurePolicyType converts a string failure policy ("Fail" or "Ignore")
// to the typed admissionv1.FailurePolicyType. Unrecognised values default to Ignore.
func admissionv1FailurePolicyType(policy string) admissionv1.FailurePolicyType {
	switch strings.ToLower(policy) {
	case "fail":
		return admissionv1.Fail
	default:
		return admissionv1.Ignore
	}
}

// admissionRegistryReader is the subset of AdmissionRegistry used by registration.
// Keeps the webhook_registration package from importing the full katalog package.
type admissionRegistryReader interface {
	ValidationGVRs() []katalog.GVREntry
	MutationGVRs() []katalog.GVREntry
}
