// pkg/reconciler/run_namespace_guard.go
package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// NamespaceGuardResult holds the outcome of a namespace restriction check.
type NamespaceGuardResult struct {
	// Allowed — true when the namespace is not restricted
	Allowed bool

	// Namespace — the namespace that was checked
	Namespace string

	// Pattern — the restriction pattern that matched (empty when Allowed)
	Pattern string
}

// CheckNamespace reports whether a target namespace is permitted for
// child resource creation. Returns a NamespaceGuardResult.
//
// Called from the registry before every Create operation:
//   - runDeployments calls this before orkdeploy.Create
//   - runSecrets calls this before orksecret.Create
//   - etc.
//
// Also called directly from runTemplateReconcile before dispatching to
// the registry — if the CR's own namespace is restricted, skip entirely.
func CheckNamespace(
	ctx context.Context,
	obj domain.Object,
	targetNamespace string,
	restricted orktypes.RestrictedNamespaces,
	crdName string,
) *NamespaceGuardResult {
	if len(restricted) == 0 {
		return &NamespaceGuardResult{Allowed: true, Namespace: targetNamespace}
	}

	if restricted.IsRestricted(targetNamespace) {
		logger.FromContext(ctx).Warn().
			Str("crd", crdName).
			Str("cr", fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())).
			Str("targetNamespace", targetNamespace).
			Msg("namespace guard: child resource creation skipped — namespace is restricted")

		return &NamespaceGuardResult{
			Allowed:   false,
			Namespace: targetNamespace,
			Pattern:   matchedPattern(targetNamespace, restricted),
		}
	}

	return &NamespaceGuardResult{Allowed: true, Namespace: targetNamespace}
}

// EventMessage returns the Kubernetes event message for a blocked namespace.
func (r *NamespaceGuardResult) EventMessage(resourceKind, resourceName string) string {
	return fmt.Sprintf(
		"Skipped creating %s %q in namespace %q — namespace is restricted (matched pattern: %q)",
		resourceKind, resourceName, r.Namespace, r.Pattern,
	)
}

// matchedPattern returns the first pattern in the restriction list that
// matches the given namespace. Used for event and log messages.
func matchedPattern(namespace string, restricted orktypes.RestrictedNamespaces) string {
	for _, pattern := range restricted {
		if matchesPattern(namespace, string(pattern)) {
			return pattern
		}
	}
	return ""
}

// matchesPattern reports whether a namespace matches a restriction pattern.
// Supports exact matches and suffix wildcards (e.g. "kube-*").
func matchesPattern(namespace, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(namespace, prefix)
	}
	return namespace == pattern
}

// resolveTargetNamespace resolves the target namespace for a child resource.
// For namespaced child resources, the target is the CR's namespace by default.
// Template expressions in the namespace field override this.
//
// If the resolved namespace is empty, falls back to the CR's namespace.
func resolveTargetNamespace(crNamespace, templateNamespace string) string {
	if templateNamespace != "" {
		return templateNamespace
	}
	return crNamespace
}
