// health/webhook_registration.go
package health

import (
	"context"
	"fmt"
	"os"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/logger"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ── Webhook configuration registration ────────────────────────────────────
//
// At startup, when ENABLE_WEBHOOKS=true, Orkestra creates or updates the
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
//
// Both are created with FailurePolicy: Ignore during initial rollout.
// Platform operators who want hard enforcement can change this to Fail.

const (
	validatingWebhookConfigName = "orkestra-validation"
	mutatingWebhookConfigName   = "orkestra-mutation"
)

// WebhookRegistrationOptions holds the configuration for webhook registration.
type WebhookRegistrationOptions struct {
	// ServiceName — the name of the Kubernetes Service fronting Orkestra.
	// The API server calls this service to reach /validate and /mutate.
	// Default: "orkestra"
	ServiceName string

	// ServiceNamespace — the namespace where the Orkestra Service lives.
	// Default: read from NAMESPACE environment variable.
	ServiceNamespace string

	// Port — the HTTPS port. Must match the conversion server port.
	// Default: 8443
	Port int32

	// FailurePolicy — what the API server does if Orkestra is unreachable.
	// admissionv1.Ignore (default): allow the operation and continue.
	// admissionv1.Fail: reject the operation if Orkestra cannot be reached.
	FailurePolicy admissionv1.FailurePolicyType

	// TLSCertFile — path to the TLS certificate file.
	// The certificate is read and used as the caBundle in the webhook config.
	// Default: read from TLS_CERT environment variable.
	TLSCertFile string
}

// DefaultWebhookRegistrationOptions returns sensible defaults.
func DefaultWebhookRegistrationOptions() WebhookRegistrationOptions {
	svc := os.Getenv("ORKESTRA_SERVICE_NAME")
	if svc == "" {
		svc = "orkestra"
	}
	return WebhookRegistrationOptions{
		ServiceName:      svc,
		ServiceNamespace: os.Getenv("NAMESPACE"),
		Port:             8443,
		FailurePolicy:    admissionv1.Ignore, // safe default — don't block on Orkestra outage
		TLSCertFile:      os.Getenv("TLS_CERT"),
	}
}

// RegisterWebhooks creates or updates the ValidatingWebhookConfiguration and
// MutatingWebhookConfiguration based on the current admission registry.
//
// Called from HealthServer.Start() when ENABLE_WEBHOOKS=true, after the
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
			Name: validatingWebhookConfigName,
			Labels: map[string]string{
				"managed-by": "orkestra",
			},
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
			Name: mutatingWebhookConfigName,
			Labels: map[string]string{
				"managed-by": "orkestra",
			},
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

// applyWebhookConfig creates or updates a ValidatingWebhookConfiguration.
// Uses server-side apply semantics: create if not exists, update if exists.
func applyWebhookConfig(ctx context.Context, client kubernetes.Interface, cfg *admissionv1.ValidatingWebhookConfiguration) error {
	existing, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, cfg.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, cfg, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	cfg.ResourceVersion = existing.ResourceVersion
	_, err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(ctx, cfg, metav1.UpdateOptions{})
	return err
}

// applyMutatingWebhookConfig creates or updates a MutatingWebhookConfiguration.
func applyMutatingWebhookConfig(ctx context.Context, client kubernetes.Interface, cfg *admissionv1.MutatingWebhookConfiguration) error {
	existing, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, cfg.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, cfg, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	cfg.ResourceVersion = existing.ResourceVersion
	_, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(ctx, cfg, metav1.UpdateOptions{})
	return err
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

func int32Ptr(i int32) *int32                                                   { return &i }
func matchPolicyPtr(p admissionv1.MatchPolicyType) *admissionv1.MatchPolicyType { return &p }
func reinvocationPolicyPtr(p admissionv1.ReinvocationPolicyType) *admissionv1.ReinvocationPolicyType {
	return &p
}

// admissionRegistryReader is the subset of AdmissionRegistry used by registration.
// Keeps the webhook_registration package from importing the full katalog package.
type admissionRegistryReader interface {
	ValidationGVRs() []katalog.GVREntry
	MutationGVRs() []katalog.GVREntry
}
