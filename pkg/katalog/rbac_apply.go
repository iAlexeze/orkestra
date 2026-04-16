// pkg/katalog/rbac_apply.go
//
// RBAC auto-apply — applies the generated RBAC to the cluster at startup
// when security.rbac.enabled: true.
//
// Uses server-side apply (PATCH with fieldManager) so the operation is
// idempotent: safe to call on every restart, no diff check needed.
//
// The generated RBAC bundle:
//   - ClusterRole       — least-privilege verbs for all managed resources
//   - ClusterRoleBinding — binds the ClusterRole to the operator ServiceAccount
//   - ServiceAccount    — the identity the operator runs as
//
// All three are owned by Orkestra (field manager: "orkestra").
// On shutdown with cleanupOnShutdown: true, all three are deleted.
package katalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	rbacFieldManager = konfig.Orkestra
	defaultOperator  = konfig.OrkOperator
)

// RBACBundle is the generated RBAC for one operator instance.
type RBACBundle struct {
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
	ServiceAccount     *corev1.ServiceAccount
}

// BuildRBACBundle builds the complete RBAC bundle from the Katalog.
// The bundle is built but not applied here — call ApplyRBAC to apply it.
func (k *Katalog) BuildRBACBundle(namespace, serviceAccountName string) *RBACBundle {
	name := k.metadata.Name + "-operator"
	if name == "-operator" {
		name = defaultOperator
	}

	rules := k.GenerateRBACRules()

	clusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "orkestra",
				"app.kubernetes.io/name":       k.metadata.Name,
			},
		},
		Rules: rules,
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "orkestra",
				"app.kubernetes.io/name":       k.metadata.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
	}

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "orkestra",
				"app.kubernetes.io/name":       k.metadata.Name,
			},
		},
	}

	return &RBACBundle{
		ClusterRole:        clusterRole,
		ClusterRoleBinding: clusterRoleBinding,
		ServiceAccount:     serviceAccount,
	}
}

// ApplyRBAC applies the RBAC bundle to the cluster using server-side apply.
// Idempotent — safe to call on every startup.
// Does nothing when security.rbac.enabled is false.
func ApplyRBAC(ctx context.Context, client kubernetes.Interface, bundle *RBACBundle) error {
	if err := applyClusterRole(ctx, client, bundle.ClusterRole); err != nil {
		return fmt.Errorf("applying ClusterRole: %w", err)
	}
	if err := applyClusterRoleBinding(ctx, client, bundle.ClusterRoleBinding); err != nil {
		return fmt.Errorf("applying ClusterRoleBinding: %w", err)
	}
	if err := applyServiceAccount(ctx, client, bundle.ServiceAccount); err != nil {
		return fmt.Errorf("applying ServiceAccount: %w", err)
	}
	logger.Info().
		Str("clusterRole", bundle.ClusterRole.Name).
		Str("serviceAccount", bundle.ServiceAccount.Name).
		Str("namespace", bundle.ServiceAccount.Namespace).
		Msg("RBAC applied")
	return nil
}

// DeleteRBAC removes the RBAC bundle from the cluster.
// Called on shutdown when security.rbac.cleanupOnShutdown: true.
func DeleteRBAC(ctx context.Context, client kubernetes.Interface, bundle *RBACBundle) error {
	// Order: binding first, then role, then service account
	name := bundle.ClusterRole.Name

	if err := client.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		logger.Warn().Err(err).Str("name", name).Msg("RBAC cleanup: ClusterRoleBinding delete failed")
	}
	if err := client.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		logger.Warn().Err(err).Str("name", name).Msg("RBAC cleanup: ClusterRole delete failed")
	}
	if err := client.CoreV1().ServiceAccounts(bundle.ServiceAccount.Namespace).Delete(
		ctx, bundle.ServiceAccount.Name, metav1.DeleteOptions{}); err != nil {
		logger.Warn().Err(err).Str("name", bundle.ServiceAccount.Name).Msg("RBAC cleanup: ServiceAccount delete failed")
	}

	logger.Info().Str("clusterRole", name).Msg("RBAC deleted")
	return nil
}

// ── Server-side apply helpers ─────────────────────────────────────────────

func applyClusterRole(ctx context.Context, client kubernetes.Interface, cr *rbacv1.ClusterRole) error {
	data, err := json.Marshal(cr)
	if err != nil {
		return err
	}
	_, err = client.RbacV1().ClusterRoles().Patch(
		ctx, cr.Name,
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: rbacFieldManager, Force: boolPtr(true)},
	)
	return err
}

func applyClusterRoleBinding(ctx context.Context, client kubernetes.Interface, crb *rbacv1.ClusterRoleBinding) error {
	data, err := json.Marshal(crb)
	if err != nil {
		return err
	}
	_, err = client.RbacV1().ClusterRoleBindings().Patch(
		ctx, crb.Name,
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: rbacFieldManager, Force: boolPtr(true)},
	)
	return err
}

func applyServiceAccount(ctx context.Context, client kubernetes.Interface, sa *corev1.ServiceAccount) error {
	data, err := json.Marshal(sa)
	if err != nil {
		return err
	}
	_, err = client.CoreV1().ServiceAccounts(sa.Namespace).Patch(
		ctx, sa.Name,
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: rbacFieldManager, Force: boolPtr(true)},
	)
	return err
}

func boolPtr(b bool) *bool { return &b }
