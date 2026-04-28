package generate

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	rbacv1 "k8s.io/api/rbac/v1"
)

// RenderBundle assembles a complete installation bundle:
// Namespace (once) → ServiceAccounts → ClusterRole → ClusterRoleBinding → ConfigMap.
// The Namespace appears exactly once at the top regardless of how many components
// are combined, so the output is safe to pipe directly into kubectl apply.
func RenderBundle(kfg *konfig.Konfig, rules []rbacv1.PolicyRule, inputFile, namespace string) (string, error) {
	nsBytes, err := renderNamespace(namespace)
	if err != nil {
		return "", fmt.Errorf("render namespace: %w", err)
	}
	rbacBytes, err := renderRBAC(kfg, rules, namespace)
	if err != nil {
		return "", fmt.Errorf("render rbac: %w", err)
	}
	cmBytes, err := renderConfigMapBytes(inputFile, namespace)
	if err != nil {
		return "", fmt.Errorf("render configmap: %w", err)
	}
	out := "---\n" + string(nsBytes) + "\n" + string(rbacBytes) + "\n---\n" + string(cmBytes)
	return out, nil
}
