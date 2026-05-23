// pkg/orkestra-registry/rolebindings/rolebinding.go
package rolebindings

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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedRoleBindingSpec is the fully resolved RoleBinding specification.
type ResolvedRoleBindingSpec struct {
	Name      string
	Namespace string
	Labels    map[string]string
	RoleRef   rbacv1.RoleRef
	Subjects  []rbacv1.Subject

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string
}

// Create creates a RoleBinding if it does not already exist.
// Idempotent — skips if the RoleBinding already exists.
// Owner reference ensures cleanup when the CR is deleted.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedRoleBindingSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("rolebinding.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().RbacV1().RoleBindings(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("rolebinding.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("rolebinding", spec.Name).
			Str("namespace", namespace).
			Msg("rolebinding already exists — skipping create")
		return nil
	}

	rb := buildRoleBinding(owner, spec, namespace)

	_, err = kube.Clientset().RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("rolebinding.Create: creating rolebinding %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("rolebinding", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("rolebinding created")

	return nil
}

// Update applies the desired subjects and roleRef to an existing RoleBinding.
// RoleRef is immutable in Kubernetes — if it changed the binding is deleted and recreated.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedRoleBindingSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().RbacV1().RoleBindings(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("rolebinding.Update: getting %q: %w", spec.Name, err)
	}

	// roleRef is immutable — recreate if it changed
	if existing.RoleRef.Name != spec.RoleRef.Name || existing.RoleRef.Kind != spec.RoleRef.Kind {
		if delErr := kube.Clientset().RbacV1().RoleBindings(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{}); delErr != nil && !errors.IsNotFound(delErr) {
			return fmt.Errorf("rolebinding.Update: deleting stale binding %q: %w", spec.Name, delErr)
		}
		return Create(ctx, kube, owner, spec)
	}

	existing.Subjects = spec.Subjects
	existing.Labels = spec.Labels

	_, err = kube.Clientset().RbacV1().RoleBindings(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("rolebinding.Update: updating rolebinding %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("rolebinding", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("rolebinding updated")

	return nil
}

// Delete deletes the RoleBinding if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedRoleBindingSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().RbacV1().RoleBindings(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("rolebinding", spec.Name).
				Str("namespace", namespace).
				Msg("rolebinding already deleted — skipping")
			return nil
		}
		return fmt.Errorf("rolebinding.Delete: deleting rolebinding %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("rolebinding", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("rolebinding deleted")

	return nil
}

// DeleteIfOwned deletes the RoleBinding only if it is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedRoleBindingSpec from a RoleBindingTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.RoleBindingTemplateSource, ownerName string) ResolvedRoleBindingSpec {
	spec := ResolvedRoleBindingSpec{
		Name:      src.Name,
		Namespace: src.Namespace,
		Labels:    make(map[string]string),
		Sleep:     src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-rolebinding"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	kind := src.RoleRef.Kind
	if kind == "" {
		kind = "Role"
	}
	spec.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     kind,
		Name:     src.RoleRef.Name,
	}

	for _, s := range src.Subjects {
		spec.Subjects = append(spec.Subjects, rbacv1.Subject{
			Kind:      s.Kind,
			Name:      s.Name,
			Namespace: s.Namespace,
		})
	}

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildRoleBinding(owner domain.Object, spec ResolvedRoleBindingSpec, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
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
		RoleRef:  spec.RoleRef,
		Subjects: spec.Subjects,
	}
}

func validateSpec(spec ResolvedRoleBindingSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
