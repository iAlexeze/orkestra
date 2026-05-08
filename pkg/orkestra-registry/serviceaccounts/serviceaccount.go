// pkg/orkestra-registry/serviceaccounts/serviceaccount.go
package serviceaccounts

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/orkestra-registry/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedServiceAccountSpec is the fully resolved ServiceAccount specification.
type ResolvedServiceAccountSpec struct {
	// Name — ServiceAccount name. Required.
	Name string

	// Namespace — target namespace. Required.
	Namespace string

	// Labels — applied to ServiceAccount metadata.
	Labels map[string]string
}

// Create creates a ServiceAccount if it does not already exist.
// Idempotent — skips if the ServiceAccount exists.
// Owner reference set so ServiceAccount is garbage collected when CR is deleted.
//
// ServiceAccounts have no meaningful spec fields that can drift after creation.
// There is no Update function — Create is called from both onCreate and
// onReconcile paths. If it exists, it stays. If it was deleted, it is recreated.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedServiceAccountSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("serviceaccount.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)

	_, err := kube.Clientset().CoreV1().ServiceAccounts(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("serviceaccount.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("serviceaccount", spec.Name).
			Str("namespace", namespace).
			Msg("serviceaccount already exists — skipping create")
		return nil
	}

	sa := buildServiceAccount(owner, spec, namespace)

	_, err = kube.Clientset().CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("serviceaccount.Create: creating serviceaccount %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("serviceaccount", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("serviceaccount created")

	return nil
}

// Delete deletes the ServiceAccount if it exists.
// For most cases owner references handle cleanup automatically —
// only use this when explicit cleanup control is needed.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedServiceAccountSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)

	err := kube.Clientset().CoreV1().ServiceAccounts(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("serviceaccount", spec.Name).
				Str("namespace", namespace).
				Msg("serviceaccount already deleted — skipping")
			return nil
		}
		return fmt.Errorf("serviceaccount.Delete: deleting serviceaccount %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("serviceaccount", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("serviceaccount deleted")

	return nil
}

// DeleteIfOwned deletes the ServiceAccount if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().CoreV1().ServiceAccounts(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Only delete if we own it
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().CoreV1().ServiceAccounts(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedServiceAccountSpec from a ServiceAccountTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.ServiceAccountTemplateSource, ownerName string) ResolvedServiceAccountSpec {
	spec := ResolvedServiceAccountSpec{
		Name:      src.Name,
		Namespace: src.Namespace,
		Labels:    make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-sa"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	spec.Labels[labels.Managed] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildServiceAccount(owner domain.Object, spec ResolvedServiceAccountSpec, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
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
	}
}

func validateSpec(spec ResolvedServiceAccountSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
