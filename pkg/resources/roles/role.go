// pkg/resources/roles/role.go
package roles

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

// ResolvedRoleSpec is the fully resolved Role specification.
type ResolvedRoleSpec struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Rules     []rbacv1.PolicyRule

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string
}

// Create creates a Role if it does not already exist.
// Idempotent — skips if the Role already exists.
// Owner reference ensures cleanup when the CR is deleted.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedRoleSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("role.Create: invalid spec: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().RbacV1().Roles(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("role.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("role", spec.Name).
			Str("namespace", namespace).
			Msg("role already exists — skipping create")
		return nil
	}

	role := buildRole(owner, spec, namespace)

	_, err = kube.Clientset().RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("role.Create: creating role %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("role", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("role created")

	return nil
}

// Update applies the desired rules to an existing Role.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedRoleSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().RbacV1().Roles(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("role.Update: getting %q: %w", spec.Name, err)
	}

	if reflect.DeepEqual(existing.Rules, spec.Rules) {
		logger.Debug().
			Str("role", spec.Name).
			Str("namespace", namespace).
			Msg("role in sync — no update needed")
		return nil
	}

	existing.Rules = spec.Rules
	existing.Labels = spec.Labels

	_, err = kube.Clientset().RbacV1().Roles(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("role.Update: updating role %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("role", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("role updated")

	return nil
}

// Delete deletes the Role if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedRoleSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().RbacV1().Roles(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("role", spec.Name).
				Str("namespace", namespace).
				Msg("role already deleted — skipping")
			return nil
		}
		return fmt.Errorf("role.Delete: deleting role %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("role", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("role deleted")

	return nil
}

// DeleteIfOwned deletes the Role only if it is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedRoleSpec from a RoleTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.RoleTemplateSource, ownerName string) ResolvedRoleSpec {
	spec := ResolvedRoleSpec{
		Name:      src.Name,
		Namespace: src.Namespace,
		Labels:    make(map[string]string),
		Sleep:     src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-role"
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

func buildRole(owner domain.Object, spec ResolvedRoleSpec, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
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
		Rules: spec.Rules,
	}
}

func validateSpec(spec ResolvedRoleSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
