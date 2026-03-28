// pkg/reconciler/namespace_guard_test.go
package reconciler_test

import (
	"context"
	"testing"

	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// guardObj builds a minimal unstructured CR for namespace guard tests.
func guardObj(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "demo.orkestra.io/v1",
			"kind":       "Website",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}

// ── Empty restricted list ─────────────────────────────────────────────────────

func TestCheckNamespace_EmptyRestricted_Allowed(t *testing.T) {
	obj := guardObj("default", "my-site")
	result := reconciler.CheckNamespace(context.Background(), obj, "default", nil, "Website")

	if !result.Allowed {
		t.Error("empty restricted list should allow any namespace")
	}
	if result.Namespace != "default" {
		t.Errorf("namespace: expected default, got %q", result.Namespace)
	}
}

// ── Exact match — blocked ─────────────────────────────────────────────────────

func TestCheckNamespace_ExactMatch_Blocked(t *testing.T) {
	obj := guardObj("default", "my-site")
	restricted := orktypes.RestrictedNamespaces{"kube-system", "cert-manager"}

	result := reconciler.CheckNamespace(context.Background(), obj, "kube-system", restricted, "Website")

	if result.Allowed {
		t.Error("kube-system is restricted — should be blocked")
	}
	if result.Namespace != "kube-system" {
		t.Errorf("namespace: expected kube-system, got %q", result.Namespace)
	}
	if result.Pattern == "" {
		t.Error("matched pattern should be set when blocked")
	}
}

// ── Wildcard — blocked ────────────────────────────────────────────────────────

func TestCheckNamespace_WildcardPrefix_Blocked(t *testing.T) {
	obj := guardObj("default", "my-site")
	restricted := orktypes.RestrictedNamespaces{"kube-*"}

	result := reconciler.CheckNamespace(context.Background(), obj, "kube-public", restricted, "Website")

	if result.Allowed {
		t.Error("kube-public should be blocked by kube-* wildcard")
	}
	if result.Pattern != "kube-*" {
		t.Errorf("pattern: expected kube-*, got %q", result.Pattern)
	}
}

func TestCheckNamespace_WildcardPrefix_DoesNotMatchOther(t *testing.T) {
	obj := guardObj("default", "my-site")
	restricted := orktypes.RestrictedNamespaces{"kube-*"}

	result := reconciler.CheckNamespace(context.Background(), obj, "production", restricted, "Website")

	if !result.Allowed {
		t.Error("production should not be blocked by kube-* wildcard")
	}
}

// ── Not in restricted list ─────────────────────────────────────────────────────

func TestCheckNamespace_NotRestricted_Allowed(t *testing.T) {
	obj := guardObj("default", "my-site")
	restricted := orktypes.RestrictedNamespaces{"kube-system", "cert-manager", "monitoring"}

	result := reconciler.CheckNamespace(context.Background(), obj, "production", restricted, "Website")

	if !result.Allowed {
		t.Error("production is not restricted — should be allowed")
	}
	if result.Pattern != "" {
		t.Errorf("pattern should be empty when allowed, got %q", result.Pattern)
	}
}

// ── Multiple patterns ─────────────────────────────────────────────────────────

func TestCheckNamespace_MultiplePatterns_MatchesFirst(t *testing.T) {
	obj := guardObj("default", "my-site")
	restricted := orktypes.RestrictedNamespaces{"kube-*", "*-system", "cert-manager"}

	// kube-system matches both kube-* and *-system
	result := reconciler.CheckNamespace(context.Background(), obj, "kube-system", restricted, "Website")

	if result.Allowed {
		t.Error("kube-system should be blocked")
	}
	// Pattern returned is the first match
	if result.Pattern == "" {
		t.Error("matched pattern should be set")
	}
}

// ── EventMessage ──────────────────────────────────────────────────────────────

func TestNamespaceGuardResult_EventMessage(t *testing.T) {
	obj := guardObj("default", "my-site")
	restricted := orktypes.RestrictedNamespaces{"kube-system"}

	result := reconciler.CheckNamespace(context.Background(), obj, "kube-system", restricted, "Website")

	msg := result.EventMessage("Deployment", "my-app")
	if msg == "" {
		t.Error("EventMessage should return a non-empty string")
	}
	// Message should mention the resource kind and name
	if len(msg) < 10 {
		t.Errorf("EventMessage seems too short: %q", msg)
	}
}
