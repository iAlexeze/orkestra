// webhook/registration.go — webhook configuration registration and cleanup.
//
// At startup, when admission webhooks or deletion/namespace protection are enabled,
// Orkestra creates or updates the corresponding ValidatingWebhookConfiguration and
// MutatingWebhookConfiguration objects that tell the API server to call Orkestra
// during admission.
//
// All registration functions are idempotent — safe to call on restart or from the
// reconciliation controller. Existing configurations are updated to match the
// current Katalog state.
package webhook

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	validatingWebhookConfigName          = "orkestra-validation"
	mutatingWebhookConfigName            = "orkestra-mutation"
	deletionProtectionWebhookConfigName  = "orkestra-deletion-protection"
	namespaceProtectionWebhookConfigName = "orkestra-namespace-protection"

	maxAttempts          = 5
	delayBetweenAttempts = 5 * time.Second
	gracePeriodSeconds   = int64(30)
)

// WebhookRegistrationOptions holds the configuration for webhook registration.
type WebhookRegistrationOptions struct {
	ServiceName              string
	ServiceNamespace         string
	Port                     int32
	FailurePolicy            admissionv1.FailurePolicyType
	TLSCertFile              string
	OrkestraResourceLabels   map[string]string
	OrkestraResourceSelector *metav1.LabelSelector
}

// WebhookCleanupOptions selects which webhook configurations to remove.
type WebhookCleanupOptions struct {
	mutating   bool
	validating bool
}

// admissionRegistryReader is the subset of katalog.AdmissionRegistry used here.
type admissionRegistryReader interface {
	ValidationGVRs() []katalog.GVREntry
	MutationGVRs() []katalog.GVREntry
}

// RegisterAdmissionWebhooks creates or updates the ValidatingWebhookConfiguration
// and MutatingWebhookConfiguration based on the current admission registry.
// Idempotent — safe to call on restart and from the webhook controller.
func RegisterAdmissionWebhooks(
	ctx context.Context,
	client kubernetes.Interface,
	registry admissionRegistryReader,
	opts WebhookRegistrationOptions,
) error {
	caBundle, err := readCABundle(opts.TLSCertFile)
	if err != nil {
		return fmt.Errorf("webhook registration: reading CA bundle: %w", err)
	}

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

// UnregisterAdmissionWebhooks removes the ValidatingWebhookConfiguration and/or
// MutatingWebhookConfiguration entries previously created from the admission registry.
// Called from Shutdown() when cleanupOnShutdown is enabled.
func UnregisterAdmissionWebhooks(
	ctx context.Context,
	client kubernetes.Interface,
	opts WebhookCleanupOptions,
) error {
	if opts.validating {
		if err := utils.RetryBackoff(func() error {
			return cleanupValidatingWebhook(ctx, client, validatingWebhookConfigName)
		}, utils.RetryOptions{
			Attempts: maxAttempts,
			Delay:    delayBetweenAttempts,
		}); err != nil {
			return fmt.Errorf("webhook cleanup: validating: %w", err)
		}
		logger.Info().Str("config", validatingWebhookConfigName).Msg("webhook: ValidatingWebhookConfiguration unregistered")
	}

	if opts.mutating {
		if err := utils.RetryBackoff(func() error {
			return cleanupMutatingWebhook(ctx, client, mutatingWebhookConfigName)
		}, utils.RetryOptions{
			Attempts: maxAttempts,
			Delay:    delayBetweenAttempts,
		}); err != nil {
			return fmt.Errorf("webhook cleanup: mutating: %w", err)
		}
		logger.Info().Str("config", mutatingWebhookConfigName).Msg("webhook: MutatingWebhookConfiguration unregistered")
	}

	return nil
}

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
			Labels: opts.OrkestraResourceLabels,
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name: "validate.orkestra.orkspace.io",
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
			},
		},
	}

	return applyWebhookConfig(ctx, client, config)
}

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
			Labels: opts.OrkestraResourceLabels,
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				Name: "mutate.orkestra.orkspace.io",
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
				ReinvocationPolicy:      reinvocationPolicyPtr(admissionv1.IfNeededReinvocationPolicy),
			},
		},
	}

	return applyMutatingWebhookConfig(ctx, client, config)
}

// registerDeletionProtectionWebhook creates or updates the ValidatingWebhookConfiguration
// for deletion protection. Two webhook entries are registered within the same configuration:
//
//  1. CRD protection — intercepts DELETE on customresourcedefinitions.
//  2. Orkestra resource protection — intercepts DELETE on deployments, services, ingresses,
//     and admission webhook configurations via ObjectSelector.
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

	var crdGVRs, orkestraGVRs []katalog.GVREntry
	for _, gvr := range gvrs {
		if gvr.Resource == "customresourcedefinitions" {
			crdGVRs = append(crdGVRs, gvr)
		} else {
			orkestraGVRs = append(orkestraGVRs, gvr)
		}
	}

	webhooks := make([]admissionv1.ValidatingWebhook, 0, 2)

	if len(crdGVRs) > 0 {
		webhooks = append(webhooks, admissionv1.ValidatingWebhook{
			Name: "protect.crds.orkestra.orkspace.io",
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

	if len(orkestraGVRs) > 0 {
		webhooks = append(webhooks, admissionv1.ValidatingWebhook{
			Name: "protect.resources.orkestra.orkspace.io",
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      opts.ServiceName,
					Namespace: opts.ServiceNamespace,
					Path:      &path,
					Port:      &port,
				},
				CABundle: caBundle,
			},
			ObjectSelector:          opts.OrkestraResourceSelector,
			Rules:                   buildDeletionProtectionRules(orkestraGVRs),
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
			Labels: opts.OrkestraResourceLabels,
		},
		Webhooks: webhooks,
	}

	return applyWebhookConfig(ctx, client, config)
}

// registerNamespaceProtectionWebhook creates or updates the ValidatingWebhookConfiguration
// for namespace protection. Intercepts CREATE and UPDATE on CRDs that declare
// allowedNamespaces or restrictedNamespaces.
func registerNamespaceProtectionWebhook(
	ctx context.Context,
	client kubernetes.Interface,
	gvrs []katalog.GVREntry,
	caBundle []byte,
	opts WebhookRegistrationOptions,
	svcName string,
	failurePolicy string,
) error {
	sideEffects := admissionv1.SideEffectClassNone
	path := "/namespace-protection"
	port := opts.Port
	fp := admissionv1FailurePolicyType(failurePolicy)

	config := &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceProtectionWebhookConfigName,
			Labels: opts.OrkestraResourceLabels,
		},
		Webhooks: []admissionv1.ValidatingWebhook{
			{
				Name: "namespace-protect.orkestra.orkspace.io",
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      svcName,
						Namespace: opts.ServiceNamespace,
						Path:      &path,
						Port:      &port,
					},
					CABundle: caBundle,
				},
				Rules:                   buildAdmissionRules(gvrs),
				FailurePolicy:           &fp,
				MatchPolicy:             matchPolicyPtr(admissionv1.Exact),
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				TimeoutSeconds:          int32Ptr(5),
			},
		},
	}

	return applyWebhookConfig(ctx, client, config)
}

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

func cleanupValidatingWebhook(ctx context.Context, client kubernetes.Interface, cfgName string) error {
	_, err := client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, cfgName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, cfgName,
		metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(gracePeriodSeconds)})
}

func cleanupMutatingWebhook(ctx context.Context, client kubernetes.Interface, cfgName string) error {
	_, err := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, cfgName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, cfgName,
		metav1.DeleteOptions{GracePeriodSeconds: int64Ptr(gracePeriodSeconds)})
}

// readCABundle reads the TLS certificate file and returns it as raw bytes.
func readCABundle(certFile string) ([]byte, error) {
	if certFile == "" {
		return nil, fmt.Errorf("TLS_CERT is required for webhook registration")
	}
	data, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", certFile, err)
	}
	return data, nil
}

// ── Pointer helpers ───────────────────────────────────────────────────────────

func int32Ptr(i int32) *int32                                                         { return &i }
func int64Ptr(i int64) *int64                                                         { return &i }
func matchPolicyPtr(p admissionv1.MatchPolicyType) *admissionv1.MatchPolicyType       { return &p }
func failurePolicyPtr(p admissionv1.FailurePolicyType) *admissionv1.FailurePolicyType { return &p }
func reinvocationPolicyPtr(p admissionv1.ReinvocationPolicyType) *admissionv1.ReinvocationPolicyType {
	return &p
}

// admissionv1FailurePolicyType converts a string failure policy to the typed form.
// Unrecognised values default to Ignore.
func admissionv1FailurePolicyType(policy string) admissionv1.FailurePolicyType {
	switch strings.ToLower(policy) {
	case "fail":
		return admissionv1.Fail
	default:
		return admissionv1.Ignore
	}
}
