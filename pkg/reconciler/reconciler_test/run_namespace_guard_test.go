// pkg/reconciler/reconciler_test/run_namespace_guard_test.go
package reconciler_test

import (
	"context"
	"testing"

	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func nsObj(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}

func TestNamespaceGuard_Allowed(t *testing.T) {
	restricted := orktypes.RestrictedNamespaces{"kube-system", "cert-manager"}
	obj := nsObj("my-website", "default")

	result := reconciler.CheckNamespace(context.Background(), obj, "default", restricted, "website")

	if !result.Allowed {
		t.Errorf("expected default to be allowed, got blocked")
	}
}

func TestNamespaceGuard_Blocked_Exact(t *testing.T) {
	restricted := orktypes.RestrictedNamespaces{"kube-system"}
	obj := nsObj("my-website", "default")

	result := reconciler.CheckNamespace(context.Background(), obj, "kube-system", restricted, "website")

	if result.Allowed {
		t.Error("expected kube-system to be blocked")
	}
	if result.Namespace != "kube-system" {
		t.Errorf("expected namespace kube-system, got %q", result.Namespace)
	}
	if result.Pattern != "kube-system" {
		t.Errorf("expected matching pattern kube-system, got %q", result.Pattern)
	}
}

func TestNamespaceGuard_Blocked_WildcardPrefix(t *testing.T) {
	restricted := orktypes.RestrictedNamespaces{"kube-*"}
	obj := nsObj("my-cr", "default")

	tests := []string{"kube-system", "kube-public", "kube-node-lease"}
	for _, ns := range tests {
		t.Run(ns, func(t *testing.T) {
			result := reconciler.CheckNamespace(context.Background(), obj, ns, restricted, "website")
			if result.Allowed {
				t.Errorf("expected %q to be blocked by kube-*", ns)
			}
			if result.Pattern != "kube-*" {
				t.Errorf("expected pattern kube-*, got %q", result.Pattern)
			}
		})
	}
}

func TestNamespaceGuard_Blocked_WildcardSuffix(t *testing.T) {
	restricted := orktypes.RestrictedNamespaces{"*-system"}
	obj := nsObj("my-cr", "default")

	tests := []string{"logging-system", "monitoring-system", "kube-system"}
	for _, ns := range tests {
		t.Run(ns, func(t *testing.T) {
			result := reconciler.CheckNamespace(context.Background(), obj, ns, restricted, "website")
			if result.Allowed {
				t.Errorf("expected %q to be blocked by *-system", ns)
			}
		})
	}
}

func TestNamespaceGuard_EmptyRestrictions_AlwaysAllowed(t *testing.T) {
	obj := nsObj("my-cr", "default")
	namespaces := []string{"kube-system", "cert-manager", "anything"}

	for _, ns := range namespaces {
		result := reconciler.CheckNamespace(context.Background(), obj, ns,
			orktypes.RestrictedNamespaces{}, "website")
		if !result.Allowed {
			t.Errorf("empty restrictions: expected %q to be allowed", ns)
		}
	}
}

func TestNamespaceGuard_EventMessage(t *testing.T) {
	result := &reconciler.NamespaceGuardResult{
		Allowed:   false,
		Namespace: "kube-system",
		Pattern:   "kube-*",
	}

	msg := result.EventMessage("ConfigMap", "my-config")

	if msg == "" {
		t.Error("expected non-empty event message")
	}

	// Should mention the resource kind, name, namespace, and pattern
	for _, substr := range []string{"ConfigMap", "my-config", "kube-system", "kube-*"} {
		if !containsStr(msg, substr) {
			t.Errorf("event message should contain %q: %q", substr, msg)
		}
	}
}
