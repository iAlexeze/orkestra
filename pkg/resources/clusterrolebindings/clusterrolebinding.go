// pkg/resources/clusterrolebindings/clusterrolebinding.go
package clusterrolebindings

import (
	"context"
	"fmt"
	"reflect"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/resources/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedClusterRoleBindingSpec is the fully resolved ClusterRoleBinding specification.
type ResolvedClusterRoleBindingSpec struct {
	Name     string
	Labels   map[string]string
	RoleRef  rbacv1.RoleRef
	Subjects []rbacv1.Subject
	Sleep    string
}

// Create creates a ClusterRoleBinding if it does not already exist.
// Idempotent — skips if the ClusterRoleBinding already exists.
// ClusterRoleBindings are cluster-scoped; ownership is tracked via the orkestra.io/owner label.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedClusterRoleBindingSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("clusterrolebinding.Create: invalid spec: %w", err)
	}
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().RbacV1().ClusterRoleBindings().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("clusterrolebinding.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("clusterrolebinding", spec.Name).
			Msg("clusterrolebinding already exists — skipping create")
		return nil
	}

	crb := buildClusterRoleBinding(spec)

	_, err = kube.Clientset().RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("clusterrolebinding.Create: creating %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("clusterrolebinding", spec.Name).
		Str("owner", owner.GetName()).
		Msg("clusterrolebinding created")

	return nil
}

// Update applies the desired subjects to an existing ClusterRoleBinding.
// RoleRef is immutable in Kubernetes — if it changed the binding is deleted and recreated.
// If it does not exist, creates it.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedClusterRoleBindingSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().RbacV1().ClusterRoleBindings().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("clusterrolebinding.Update: getting %q: %w", spec.Name, err)
	}

	// roleRef is immutable — recreate if it changed
	if existing.RoleRef.Name != spec.RoleRef.Name || existing.RoleRef.Kind != spec.RoleRef.Kind {
		if delErr := kube.Clientset().RbacV1().ClusterRoleBindings().Delete(ctx, spec.Name, metav1.DeleteOptions{}); delErr != nil && !errors.IsNotFound(delErr) {
			return fmt.Errorf("clusterrolebinding.Update: deleting stale binding %q: %w", spec.Name, delErr)
		}
		return Create(ctx, kube, owner, spec)
	}

	if reflect.DeepEqual(existing.Subjects, spec.Subjects) && reflect.DeepEqual(existing.Labels, spec.Labels) {
		logger.Debug().
			Str("clusterrolebinding", spec.Name).
			Msg("clusterrolebinding in sync — no update needed")
		return nil
	}

	existing.Subjects = spec.Subjects
	existing.Labels = spec.Labels

	_, err = kube.Clientset().RbacV1().ClusterRoleBindings().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("clusterrolebinding.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("clusterrolebinding", spec.Name).
		Str("owner", owner.GetName()).
		Msg("clusterrolebinding updated")

	return nil
}

// Delete deletes the ClusterRoleBinding if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedClusterRoleBindingSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().RbacV1().ClusterRoleBindings().Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("clusterrolebinding.Delete: deleting %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("clusterrolebinding", spec.Name).
		Str("owner", owner.GetName()).
		Msg("clusterrolebinding deleted")

	return nil
}

// DeleteIfOwned deletes the ClusterRoleBinding only if it is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name string) error {

	existing, err := kube.Clientset().RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedClusterRoleBindingSpec from a ClusterRoleBindingTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.ClusterRoleBindingTemplateSource, ownerName string) ResolvedClusterRoleBindingSpec {
	spec := ResolvedClusterRoleBindingSpec{
		Name:   src.Name,
		Labels: make(map[string]string),
		Sleep:  src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-crb"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	kind := src.RoleRef.Kind
	if kind == "" {
		kind = "ClusterRole"
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

func buildClusterRoleBinding(spec ResolvedClusterRoleBindingSpec) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   spec.Name,
			Labels: spec.Labels,
			// No OwnerReference: ClusterRoleBindings are cluster-scoped; a namespace-scoped CR
			// cannot own a cluster-scoped resource. Ownership is tracked via the
			// OrkestraOwner label; cleanup is performed explicitly via DeleteIfOwned.
		},
		RoleRef:  spec.RoleRef,
		Subjects: spec.Subjects,
	}
}

func validateSpec(spec ResolvedClusterRoleBindingSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
