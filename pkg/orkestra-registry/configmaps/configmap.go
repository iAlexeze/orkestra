// pkg/orkestra-registry/configmaps/configmap.go
package configmaps

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedConfigMapSpec is the fully resolved ConfigMap specification.
type ResolvedConfigMapSpec struct {
	// Name — ConfigMap name. Required.
	Name string

	// Namespace — target namespace. Required.
	Namespace string

	// Data — key-value configuration entries.
	Data map[string]string

	// FromConfigMap — name of a source ConfigMap to copy from.
	// When set, copies all keys from the source into the target.
	// Individual Data entries merge on top — override specific keys if needed.
	FromConfigMap string

	// FromNamespace — namespace where FromConfigMap lives.
	// Default: same namespace as the CR.
	FromNamespace string

	// Labels — applied to ConfigMap metadata.
	Labels map[string]string
}

// Create creates a ConfigMap if it does not already exist.
// Idempotent — skips if ConfigMap exists.
// Owner reference set for cascade deletion.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedConfigMapSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("configmap.Create: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	_, err := kube.Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("configmap.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("configmap", spec.Name).
			Str("namespace", namespace).
			Msg("configmap already exists — skipping create")
		return nil
	}

	data, err := resolveData(ctx, kube, spec, owner)
	if err != nil {
		return fmt.Errorf("configmap.Create: resolving data: %w", err)
	}

	cm := buildConfigMap(owner, spec, namespace, data)

	_, err = kube.Clientset().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("configmap.Create: creating %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("configmap", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("configmap created")

	return nil
}

// Update reconciles an existing ConfigMap to match the resolved spec.
// Re-syncs data from FromConfigMap on every reconcile if set.
// If ConfigMap does not exist, creates it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedConfigMapSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("configmap.Update: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	existing, err := kube.Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("configmap", spec.Name).
				Str("namespace", namespace).
				Msg("configmap not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("configmap.Update: getting %q: %w", spec.Name, err)
	}

	data, err := resolveData(ctx, kube, spec, owner)
	if err != nil {
		return fmt.Errorf("configmap.Update: resolving data: %w", err)
	}

	if reflect.DeepEqual(existing.Data, data) {
		logger.Debug().
			Str("configmap", spec.Name).
			Str("namespace", namespace).
			Msg("configmap in sync — no update needed")
		return nil
	}

	updated := existing.DeepCopy()
	updated.Data = data

	_, err = kube.Clientset().CoreV1().ConfigMaps(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("configmap.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("configmap", spec.Name).
		Str("namespace", namespace).
		Msg("configmap updated")

	return nil
}

// Delete deletes the ConfigMap if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedConfigMapSpec) error {
	namespace := resolveNamespace(owner, spec)

	err := kube.Clientset().CoreV1().ConfigMaps(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("configmap", spec.Name).
				Str("namespace", namespace).
				Msg("configmap already deleted — skipping")
			return nil
		}
		return fmt.Errorf("configmap.Delete: deleting %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("configmap", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("configmap deleted")

	return nil
}

// CopyToNamespaces copies a ConfigMap to multiple target namespaces.
// Reads the source once and creates copies in each namespace.
// Idempotent — skips namespaces where the ConfigMap already exists.
//
// Example Katalog declaration:
//
//	onCreate:
//	  configMaps:
//	    - name: app-config
//	      fromConfigMap: base-app-config
//	      fromNamespace: platform
//	      toNamespaces:
//	        - "{{ .metadata.namespace }}"
//	        - staging
//	        - production
func CopyToNamespaces(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	owner domain.Object,
	spec ResolvedConfigMapSpec,
	toNamespaces []string,
) error {
	// Fetch source once — reuse for all target namespaces
	data, err := resolveData(ctx, kube, spec, owner)
	if err != nil {
		return fmt.Errorf("configmap.CopyToNamespaces: reading source: %w", err)
	}

	for _, ns := range toNamespaces {
		if ns == "" {
			continue
		}

		_, err := kube.Clientset().CoreV1().ConfigMaps(ns).Get(ctx, spec.Name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("configmap.CopyToNamespaces: checking %q in %q: %w", spec.Name, ns, err)
		}
		if err == nil {
			logger.Debug().
				Str("configmap", spec.Name).
				Str("namespace", ns).
				Msg("configmap already exists in namespace — skipping")
			continue
		}

		nsSpec := spec
		nsSpec.Namespace = ns
		cm := buildConfigMap(owner, nsSpec, ns, data)

		_, err = kube.Clientset().CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("configmap.CopyToNamespaces: creating %q in %q: %w", spec.Name, ns, err)
		}

		logger.Info().
			Str("configmap", spec.Name).
			Str("namespace", ns).
			Str("owner", owner.GetName()).
			Msg("configmap copied to namespace")
	}

	return nil
}

// Resolve builds a ResolvedConfigMapSpec from a ConfigMapTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.ConfigMapTemplateSource, ownerName string) ResolvedConfigMapSpec {
	spec := ResolvedConfigMapSpec{
		Name:          src.Name,
		Namespace:     src.Namespace,
		Data:          src.Data,
		FromConfigMap: src.FromConfigMap,
		FromNamespace: src.FromNamespace,
		Labels:        make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-config"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	spec.Labels[konfig.LabelManaged] = konfig.LabelManagedValue
	spec.Labels[konfig.LabelOrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// resolveData merges FromConfigMap source data with declared Data.
// Declared Data keys win over source keys — allows targeted overrides.
func resolveData(
	ctx context.Context,
	kube *kubeclient.Kubeclient,
	spec ResolvedConfigMapSpec,
	owner domain.Object,
) (map[string]string, error) {
	merged := make(map[string]string)

	// Base: copy from source ConfigMap if declared
	if spec.FromConfigMap != "" {
		fromNS := spec.FromNamespace
		if fromNS == "" {
			fromNS = owner.GetNamespace()
		}
		if fromNS == "" {
			fromNS = "default"
		}

		source, err := kube.Clientset().CoreV1().ConfigMaps(fromNS).Get(ctx, spec.FromConfigMap, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("reading source configmap %q from %q: %w", spec.FromConfigMap, fromNS, err)
		}

		for k, v := range source.Data {
			merged[k] = v
		}

		logger.Debug().
			Str("source", spec.FromConfigMap).
			Str("sourceNamespace", fromNS).
			Int("keys", len(source.Data)).
			Msg("copied configmap data from source")
	}

	// Override: declared Data keys win over source keys
	for k, v := range spec.Data {
		merged[k] = v
	}

	return merged, nil
}

func buildConfigMap(owner domain.Object, spec ResolvedConfigMapSpec, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
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
					Controller:         utils.BoolPtr(true),
					BlockOwnerDeletion: utils.BoolPtr(true),
				},
			},
		},
		Data: data,
	}
}

func validateSpec(spec ResolvedConfigMapSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func resolveNamespace(owner domain.Object, spec ResolvedConfigMapSpec) string {
	if spec.Namespace != "" {
		return spec.Namespace
	}
	if owner.GetNamespace() != "" {
		return owner.GetNamespace()
	}
	return "default"
}
