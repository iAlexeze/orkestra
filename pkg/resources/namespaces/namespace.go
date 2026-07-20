// pkg/resources/namespace/namespace.go
package namespaces

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/resources/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedNamespaceSpec is the fully resolved Namespace specification.
type ResolvedNamespaceSpec struct {
	// Name — name. Required.
	Name string

	// Labels — applied to Namespace metadata.
	Labels map[string]string

	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage.
	// optional
	Finalizers []string

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string
}

// Create creates a Namespace if it does not already exist.
// Idempotent — skips if the Namespace exists.
// Owner reference set so Namespace is garbage collected when CR is deleted.
//
// Namespaces have no meaningful spec fields that can drift after creation.
// There is no Update function — Create is called from both onCreate and
// onReconcile paths. If it exists, it stays. If it was deleted, it is recreated.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedNamespaceSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("namespace.Create: invalid spec: %w", err)
	}

	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().CoreV1().Namespaces().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("namespace.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("namespace", spec.Name).
			Msg("namespace already exists — skipping create")
		return nil
	}

	sa := buildNamespace(owner, spec)

	_, err = kube.Clientset().CoreV1().Namespaces().Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("namespace.Create: creating namespace %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("namespace", spec.Name).
		Str("owner", owner.GetName()).
		Msg("namespace created")

	return nil
}

// Update reconciles an existing Namespace to match the resolved spec.
// Handles drift — if replicas or image have changed, patches the Namespace.
// If the Namespace does not exist, creates it.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedNamespaceSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("namespace.Update: invalid spec: %w", err)
	}

	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().CoreV1().Namespaces().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("namespace", spec.Name).
				Msg("namespace not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("namespace.Update: getting namespace %q: %w", spec.Name, err)
	}

	// Check for drift — replicas and image are the reconcilable fields
	drifted := false
	updated := existing.DeepCopy()

	if !common.LabelsEqual(existing.Labels, spec.Labels) {
		updated.Labels = spec.Labels
		drifted = true
	}

	if !drifted {
		logger.Debug().
			Str("namespace", spec.Name).
			Msg("namespace in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().CoreV1().Namespaces().Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("namespace.Update: updating namespace %q: %w", spec.Name, err)
	}

	return nil
}

// Delete deletes the Namespace if it exists.
// For most cases owner references handle cleanup automatically —
// only use this when explicit cleanup control is needed.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedNamespaceSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().CoreV1().Namespaces().Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("namespace", spec.Name).
				Msg("namespace already deleted — skipping")
			return nil
		}
		return fmt.Errorf("namespace.Delete: deleting namespace %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("namespace", spec.Name).
		Str("owner", owner.GetName()).
		Msg("namespace deleted")

	return nil
}

// DeleteIfOwned deletes the Namespace if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name string) error {

	existing, err := kube.Clientset().CoreV1().Namespaces().
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
	return kube.Clientset().CoreV1().Namespaces().
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedNamespaceSpec from a NamespaceTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.NamespaceTemplateSource, ownerName string) ResolvedNamespaceSpec {
	spec := ResolvedNamespaceSpec{
		Name:       src.Name,
		Labels:     make(map[string]string),
		Finalizers: src.Finalizers,
		Sleep:      src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "ns"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildNamespace(owner domain.Object, spec ResolvedNamespaceSpec) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   spec.Name,
			Labels: spec.Labels,
		},
	}
	// Cluster-scoped owners can hold owner references on cluster-scoped resources.
	// Namespace-scoped owners cannot — GC would treat the namespace as orphaned and
	// delete it immediately. Fall back to label-based tracking in that case.
	if owner.GetNamespace() == "" {
		ns.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion:         owner.GetObjectKind().GroupVersionKind().GroupVersion().String(),
				Kind:               owner.GetObjectKind().GroupVersionKind().Kind,
				Name:               owner.GetName(),
				UID:                owner.GetUID(),
				Controller:         utils.BoolPtr(true),
				BlockOwnerDeletion: utils.BoolPtr(true),
			},
		}
	}
	return ns
}

func validateSpec(spec ResolvedNamespaceSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
