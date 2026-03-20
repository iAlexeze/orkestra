// pkg/orkestra-registry/secrets/secret.go
package secrets

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedSecretSpec is the fully resolved Secret specification.
type ResolvedSecretSpec struct {
	// Name — Secret name. Required.
	Name string

	// Namespace — target namespace. Required.
	Namespace string

	// StringData — static key-value entries.
	// Values are plain strings — template expressions are not supported here.
	StringData map[string]string

	// Data
	Data map[string][]byte `yaml:"data" validate:"omitempty"`

	// Type — Kubernetes Secret type.
	// Default: "Opaque"
	Type string `yaml:"type" validate:"omitempty"`

	// FromSecret — name of a source Secret to copy from.
	// When set, copies all keys from the source into the target.
	// Individual Data entries merge on top — override specific keys if needed.
	FromSecret string

	// FromNamespace — namespace where FromSecret lives.
	// Default: same namespace as the CR.
	FromNamespace string

	// Labels — applied to Secret metadata.
	Labels map[string]string
}

// Create creates a Secret in the target namespace if it does not already exist.
// When FromSecret is set, copies data from the source Secret automatically.
// Idempotent — skips creation if the Secret already exists.
// Owner reference is set so the Secret is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedSecretSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("secret.Create: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	_, err := kube.Clientset().CoreV1().Secrets(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("secret.Create: checking existence of %q in %q: %w", spec.Name, namespace, err)
	}
	if err == nil {
		logger.Debug().
			Str("secret", spec.Name).
			Str("namespace", namespace).
			Msg("secret already exists — skipping create")
		return nil
	}

	// If copying from another Secret, fetch its data first
	data, stringData, err := resolveData(ctx, kube, spec, owner)
	if err != nil {
		return fmt.Errorf("secret.Create: resolving data for %q: %w", spec.Name, err)
	}

	secret := buildSecret(owner, spec, namespace, data, stringData)

	_, err = kube.Clientset().CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("secret.Create: creating secret %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("secret", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("secret created")

	return nil
}

// Update reconciles an existing Secret to match the resolved spec.
// When FromSecret is set, re-syncs data from the source Secret on every reconcile.
// If the Secret does not exist, creates it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedSecretSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("secret.Update: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	existing, err := kube.Clientset().CoreV1().Secrets(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("secret", spec.Name).
				Str("namespace", namespace).
				Msg("secret not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("secret.Update: getting secret %q: %w", spec.Name, err)
	}

	// Re-resolve data — picks up changes in the source Secret
	data, stringData, err := resolveData(ctx, kube, spec, owner)
	if err != nil {
		return fmt.Errorf("secret.Update: resolving data for %q: %w", spec.Name, err)
	}

	// Check if data has changed
	if secretDataEqual(existing.Data, data) && stringDataEqual(existing.StringData, stringData) {
		logger.Debug().
			Str("secret", spec.Name).
			Str("namespace", namespace).
			Msg("secret in sync — no update needed")
		return nil
	}

	updated := existing.DeepCopy()
	updated.Data = data
	updated.StringData = stringData

	_, err = kube.Clientset().CoreV1().Secrets(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("secret.Update: updating secret %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("secret", spec.Name).
		Str("namespace", namespace).
		Msg("secret updated")

	return nil
}

// Delete deletes the Secret if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedSecretSpec) error {
	namespace := resolveNamespace(owner, spec)

	err := kube.Clientset().CoreV1().Secrets(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("secret", spec.Name).
				Str("namespace", namespace).
				Msg("secret already deleted — skipping")
			return nil
		}
		return fmt.Errorf("secret.Delete: deleting secret %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("secret", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("secret deleted")

	return nil
}

// CopyToNamespaces copies a Secret to multiple target namespaces.
// Each copy is owned by the CR — garbage collected when the CR is deleted.
// Idempotent — skips namespaces where the Secret already exists.
// This is the power pattern: one declaration copies to N namespaces automatically.
//
// Example Katalog declaration:
//
//	onCreate:
//	  secrets:
//	    - name: db-credentials
//	      fromSecret: master-db-creds
//	      fromNamespace: platform
//	      toNamespaces:
//	        - "{{ .metadata.namespace }}"
//	        - monitoring
//	        - staging
func CopyToNamespaces(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	owner domain.Object,
	spec ResolvedSecretSpec,
	toNamespaces []string,
) error {
	// Fetch source Secret once — reuse data for all target namespaces
	sourceData, sourceStringData, err := resolveData(ctx, kube, spec, owner)
	if err != nil {
		return fmt.Errorf("secret.CopyToNamespaces: reading source %q: %w", spec.FromSecret, err)
	}

	for _, ns := range toNamespaces {
		if ns == "" {
			continue
		}

		nsSpec := spec
		nsSpec.Namespace = ns

		_, err := kube.Clientset().CoreV1().Secrets(ns).Get(ctx, spec.Name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("secret.CopyToNamespaces: checking %q in %q: %w", spec.Name, ns, err)
		}
		if err == nil {
			logger.Debug().
				Str("secret", spec.Name).
				Str("namespace", ns).
				Msg("secret already exists in namespace — skipping")
			continue
		}

		secret := buildSecret(owner, nsSpec, ns, sourceData, sourceStringData)

		_, err = kube.Clientset().CoreV1().Secrets(ns).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("secret.CopyToNamespaces: creating %q in %q: %w", spec.Name, ns, err)
		}

		logger.Info().
			Str("secret", spec.Name).
			Str("namespace", ns).
			Str("owner", owner.GetName()).
			Msg("secret copied to namespace")
	}

	return nil
}

// Resolve builds a ResolvedSecretSpec from a SecretTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.SecretTemplateSource, ownerName string) ResolvedSecretSpec {
	spec := ResolvedSecretSpec{
		Name:          src.Name,
		Namespace:     src.Namespace,
		FromSecret:    src.FromSecret,
		FromNamespace: src.FromNamespace,
		Type:          src.Type,
		StringData:    src.Data, // declared as strings in YAML
		Labels:        make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-secret"
	}
	if spec.Type == "" {
		spec.Type = "Opaque"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	// System labels
	spec.Labels["managed-by"] = "orkestra"
	spec.Labels["orkestra-owner"] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// resolveData returns the Secret data to write.
// When FromSecret is set, fetches data from the source Secret.
// Otherwise uses the static data declared in the spec.
func resolveData(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	spec ResolvedSecretSpec,
	owner domain.Object,
) (map[string][]byte, map[string]string, error) {
	if spec.FromSecret == "" {
		return spec.Data, spec.StringData, nil
	}

	// Determine source namespace
	fromNS := spec.FromNamespace
	if fromNS == "" {
		fromNS = owner.GetNamespace()
	}
	if fromNS == "" {
		fromNS = "default"
	}

	source, err := kube.Clientset().CoreV1().Secrets(fromNS).Get(ctx, spec.FromSecret, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("reading source secret %q from %q: %w", spec.FromSecret, fromNS, err)
	}

	logger.Debug().
		Str("source", spec.FromSecret).
		Str("sourceNamespace", fromNS).
		Msg("copying secret data from source")

	return source.Data, nil, nil
}

func buildSecret(owner domain.Object, spec ResolvedSecretSpec, namespace string, data map[string][]byte, stringData map[string]string) *corev1.Secret {
	secretType := corev1.SecretTypeOpaque
	switch spec.Type {
	case "kubernetes.io/tls":
		secretType = corev1.SecretTypeTLS
	case "kubernetes.io/dockerconfigjson":
		secretType = corev1.SecretTypeDockerConfigJson
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
			Labels:    spec.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         owner.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:               owner.GetObjectKind().GroupVersionKind().Kind,
					Name:               owner.GetName(),
					UID:                owner.GetUID(),
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Type:       secretType,
		Data:       data,
		StringData: stringData,
	}
}

func validateSpec(spec ResolvedSecretSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	if spec.FromSecret == "" && len(spec.Data) == 0 && len(spec.StringData) == 0 {
		return fmt.Errorf("one of fromSecret or data must be declared")
	}
	return nil
}

func resolveNamespace(owner domain.Object, spec ResolvedSecretSpec) string {
	if spec.Namespace != "" {
		return spec.Namespace
	}
	if owner.GetNamespace() != "" {
		return owner.GetNamespace()
	}
	return "default"
}

func secretDataEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || string(va) != string(vb) {
			return false
		}
	}
	return true
}

func stringDataEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		if b[k] != va {
			return false
		}
	}
	return true
}

func boolPtr(b bool) *bool { return &b }
