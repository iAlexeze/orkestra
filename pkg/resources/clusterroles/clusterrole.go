// pkg/resources/clusterroles/clusterrole.go
package clusterroles

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
	"github.com/orkspace/orkestra/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedClusterRoleSpec is the fully resolved ClusterRole specification.
type ResolvedClusterRoleSpec struct {
	Name   string
	Labels map[string]string
	Rules  []rbacv1.PolicyRule
	Sleep  string
}

// Create creates a ClusterRole if it does not already exist.
// Idempotent — skips if the ClusterRole already exists.
// ClusterRoles are cluster-scoped; ownership is tracked via the orkestra.io/owner label.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedClusterRoleSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("clusterrole.Create: invalid spec: %w", err)
	}
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().RbacV1().ClusterRoles().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("clusterrole.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("clusterrole", spec.Name).
			Msg("clusterrole already exists — skipping create")
		return nil
	}

	cr := buildClusterRole(owner, spec)

	_, err = kube.Clientset().RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("clusterrole.Create: creating %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("clusterrole", spec.Name).
		Str("owner", owner.GetName()).
		Msg("clusterrole created")

	return nil
}

// Update applies the desired rules to an existing ClusterRole.
// If it does not exist, creates it.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedClusterRoleSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().RbacV1().ClusterRoles().Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("clusterrole.Update: getting %q: %w", spec.Name, err)
	}

	if reflect.DeepEqual(existing.Rules, spec.Rules) && reflect.DeepEqual(existing.Labels, spec.Labels) {
		logger.Debug().
			Str("clusterrole", spec.Name).
			Msg("clusterrole in sync — no update needed")
		return nil
	}

	existing.Rules = spec.Rules
	existing.Labels = spec.Labels

	_, err = kube.Clientset().RbacV1().ClusterRoles().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("clusterrole.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("clusterrole", spec.Name).
		Str("owner", owner.GetName()).
		Msg("clusterrole updated")

	return nil
}

// Delete deletes the ClusterRole if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedClusterRoleSpec) error {
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().RbacV1().ClusterRoles().Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("clusterrole.Delete: deleting %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("clusterrole", spec.Name).
		Str("owner", owner.GetName()).
		Msg("clusterrole deleted")

	return nil
}

// DeleteIfOwned deletes the ClusterRole only if it is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name string) error {

	existing, err := kube.Clientset().RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedClusterRoleSpec from a ClusterRoleTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.ClusterRoleTemplateSource, ownerName string) ResolvedClusterRoleSpec {
	spec := ResolvedClusterRoleSpec{
		Name:   src.Name,
		Labels: make(map[string]string),
		Sleep:  src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-cluster-role"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}
	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	for _, r := range src.Rules {
		spec.Rules = append(spec.Rules, rbacv1.PolicyRule{
			APIGroups:     r.APIGroups,
			Resources:     r.Resources,
			Verbs:         r.Verbs,
			ResourceNames: r.ResourceNames,
		})
	}

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildClusterRole(owner domain.Object, spec ResolvedClusterRoleSpec) *rbacv1.ClusterRole {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   spec.Name,
			Labels: spec.Labels,
		},
		Rules: spec.Rules,
	}
	if owner.GetNamespace() == "" {
		cr.OwnerReferences = []metav1.OwnerReference{
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
	return cr
}

func validateSpec(spec ResolvedClusterRoleSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
