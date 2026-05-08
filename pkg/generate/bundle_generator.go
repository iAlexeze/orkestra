package generate

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RenderBundle assembles a complete installation bundle:
// Namespace (once) → ServiceAccounts → ClusterRole → ClusterRoleBinding → ConfigMap.
// The Namespace appears exactly once at the top regardless of how many components
// are combined, so the output is safe to pipe directly into kubectl apply.
func RenderBundle(
	rules []rbacv1.PolicyRule,
	inputFile, namespace, workloadNamespace string,
) (string, error) {

	nsBytes, err := renderNamespace(namespace)
	if err != nil {
		return "", fmt.Errorf("render namespace: %w", err)
	}

	workloadNsBytes, err := renderNamespace(workloadNamespace)
	if err != nil {
		return "", fmt.Errorf("render task namespace: %w", err)
	}

	rbacBytes, err := renderRBAC(rules, namespace)
	if err != nil {
		return "", fmt.Errorf("render rbac: %w", err)
	}

	cmBytes, err := renderConfigMapBytes(inputFile, namespace)
	if err != nil {
		return "", fmt.Errorf("render configmap: %w", err)
	}

	// Build YAML docs with proper separators
	var docs []string

	docs = append(docs, string(nsBytes))

	if workloadNamespace != "" {
		docs = append(docs, string(workloadNsBytes))
	}

	// renderRBAC prefixes each object with its own ---; strip the leading one
	// so the assembler loop below adds it uniformly alongside the other docs.
	docs = append(docs, strings.TrimPrefix(string(rbacBytes), "---\n"))
	docs = append(docs, string(cmBytes))

	// Join all docs: each gets a leading --- from here.
	out := ""
	for i, d := range docs {
		out += "---\n" + d + "\n"
		if i < len(docs)-1 {
			out += "\n"
		}
	}

	return out, nil
}
