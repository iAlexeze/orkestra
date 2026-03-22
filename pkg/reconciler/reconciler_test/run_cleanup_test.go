// pkg/reconciler/run_cleanup_test.go
package reconciler_test

import (
	"context"
	"testing"

	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/reconciler"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

var podGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

func orphanedPod(name, namespace string) *unstructured.Unstructured {
	// No ownerReferences — the defining characteristic of an orphaned pod
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{},
			},
		},
	}
}

func ownedPod(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "ReplicaSet",
						"name":       "my-rs",
					},
				},
			},
		},
	}
}

func cleanupRule(field, message string, dryRun bool) orktypes.ValidationRule {
	return orktypes.ValidationRule{
		Field:    field,
		Operator: orktypes.ConditionExists,
		Message:  message,
		Action:   orktypes.ValidationActionCleanup,
		DryRun:   dryRun,
	}
}

// ── RunCleanupRules ───────────────────────────────────────────────────────────

func TestRunCleanupRules_NoRules(t *testing.T) {
	obj := orphanedPod("orphan-1", "default")
	result := reconciler.RunCleanupRules(obj, nil, "pod")
	if result.ShouldDelete {
		t.Error("no rules — should not delete")
	}
}

func TestRunCleanupRules_CleanupRule_Matches(t *testing.T) {
	// orphanedPod has no ownerReferences — notExists rule fires
	obj := orphanedPod("orphan-1", "default")
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "metadata.ownerReferences",
				Operator: orktypes.ConditionExists,
				Message:  "all pods must have an owner reference",
				Action:   orktypes.ValidationActionCleanup,
			},
		},
	}

	result := reconciler.RunCleanupRules(obj, cfg, "pod")

	if !result.ShouldDelete {
		t.Error("cleanup rule matched orphaned pod — ShouldDelete must be true")
	}
	if result.DryRunMatch {
		t.Error("non-dry-run rule should not set DryRunMatch")
	}
	if result.MatchedRule == nil {
		t.Error("MatchedRule should be set when a rule fires")
	}
}

func TestRunCleanupRules_CleanupRule_NoMatch(t *testing.T) {
	// ownedPod has ownerReferences — exists rule passes, cleanup does not fire
	obj := ownedPod("owned-1", "default")
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "metadata.ownerReferences",
				Operator: orktypes.ConditionExists,
				Message:  "all pods must have an owner reference",
				Action:   orktypes.ValidationActionCleanup,
			},
		},
	}

	result := reconciler.RunCleanupRules(obj, cfg, "pod")

	if result.ShouldDelete {
		t.Error("owned pod has owner reference — cleanup rule should NOT fire")
	}
}

func TestRunCleanupRules_DryRun_DoesNotDelete(t *testing.T) {
	obj := orphanedPod("orphan-dry", "default")
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			cleanupRule("metadata.ownerReferences", "orphaned pod", true), // dryRun: true
		},
	}

	result := reconciler.RunCleanupRules(obj, cfg, "pod")

	if result.ShouldDelete {
		t.Error("dry-run rule must not set ShouldDelete")
	}
	if !result.DryRunMatch {
		t.Error("dry-run rule that matched must set DryRunMatch")
	}
}

func TestRunCleanupRules_ShortCircuits_OnFirstMatch(t *testing.T) {
	// Two cleanup rules — only first should fire (cleanup short-circuits)
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
				// no ownerReferences, no labels
			},
		},
	}

	firstRule := orktypes.ValidationRule{
		Field:    "metadata.ownerReferences",
		Operator: orktypes.ConditionExists,
		Message:  "first cleanup rule",
		Action:   orktypes.ValidationActionCleanup,
	}
	secondRule := orktypes.ValidationRule{
		Field:    "metadata.labels.team",
		Operator: orktypes.ConditionExists,
		Message:  "second cleanup rule — should not fire",
		Action:   orktypes.ValidationActionCleanup,
	}

	cfg := &orktypes.ValidationConfig{Rules: []orktypes.ValidationRule{firstRule, secondRule}}
	result := reconciler.RunCleanupRules(obj, cfg, "pod")

	if result.MatchedRule == nil {
		t.Fatal("expected a matched rule")
	}
	if result.MatchedRule.Message != "first cleanup rule" {
		t.Errorf("expected first rule to match, got: %q", result.MatchedRule.Message)
	}
}

func TestRunCleanupRules_NonCleanupRules_Ignored(t *testing.T) {
	// deny and warn rules should be ignored by RunCleanupRules
	obj := orphanedPod("test", "default")
	cfg := &orktypes.ValidationConfig{
		Rules: []orktypes.ValidationRule{
			{
				Field:    "metadata.ownerReferences",
				Operator: orktypes.ConditionExists,
				Message:  "deny rule — not cleanup",
				Action:   orktypes.ValidationActionDeny,
			},
			{
				Field:    "metadata.labels.team",
				Operator: orktypes.ConditionExists,
				Message:  "warn rule — not cleanup",
				Action:   orktypes.ValidationActionWarn,
			},
		},
	}

	result := reconciler.RunCleanupRules(obj, cfg, "pod")

	if result.ShouldDelete || result.DryRunMatch {
		t.Error("deny and warn rules must not trigger cleanup")
	}
}

// ── ExecuteCleanup ────────────────────────────────────────────────────────────

func TestExecuteCleanup_DeletesResource(t *testing.T) {
	pod := orphanedPod("orphan-to-delete", "default")

	scheme := runtime.NewScheme()
	client := dynamicfake.NewSimpleDynamicClient(scheme, pod)

	gracePeriod := int64(0)
	rule := &orktypes.ValidationRule{
		Field:              "metadata.ownerReferences",
		Message:            "orphaned pod",
		Action:             orktypes.ValidationActionCleanup,
		GracePeriodSeconds: &gracePeriod,
	}

	err := reconciler.ExecuteCleanup(
		context.Background(),
		newFakeKubeclient(client),
		pod,
		podGVR,
		rule,
		"pod",
	)

	if err != nil {
		t.Fatalf("ExecuteCleanup returned unexpected error: %v", err)
	}

	// Verify the resource was deleted from the fake client
	list, err := client.Resource(podGVR).Namespace("default").List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		t.Fatalf("listing after delete: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 pods after cleanup, got %d", len(list.Items))
	}
}

func TestExecuteCleanup_GracePeriod(t *testing.T) {
	pod := orphanedPod("slow-pod", "default")

	scheme := runtime.NewScheme()
	client := dynamicfake.NewSimpleDynamicClient(scheme, pod)

	gracePeriod := int64(30)
	rule := &orktypes.ValidationRule{
		Field:              "metadata.ownerReferences",
		Message:            "orphaned pod",
		Action:             orktypes.ValidationActionCleanup,
		GracePeriodSeconds: &gracePeriod,
	}

	// Should not error — grace period is passed to the delete options
	err := reconciler.ExecuteCleanup(
		context.Background(),
		newFakeKubeclient(client),
		pod,
		podGVR,
		rule,
		"pod",
	)

	if err != nil {
		t.Fatalf("unexpected error with grace period: %v", err)
	}
}

// ── CleanupEventMessage ───────────────────────────────────────────────────────

func TestCleanupEventMessage_LiveDeletion(t *testing.T) {
	rule := &orktypes.ValidationRule{
		Field:   "metadata.ownerReferences",
		Message: "all pods must have an owner reference",
		Action:  orktypes.ValidationActionCleanup,
	}

	msg := reconciler.CleanupEventMessage(rule, "")

	if !containsStr(msg, "metadata.ownerReferences") {
		t.Errorf("message should mention field: %q", msg)
	}
	if !containsStr(msg, "all pods must have an owner reference") {
		t.Errorf("message should include user message: %q", msg)
	}
	if containsStr(msg, "dry-run") {
		t.Error("live deletion message should not mention dry-run")
	}
}

func TestCleanupEventMessage_DryRun(t *testing.T) {
	rule := &orktypes.ValidationRule{
		Field:   "metadata.ownerReferences",
		Message: "all pods must have an owner reference",
		Action:  orktypes.ValidationActionCleanup,
		DryRun:  true,
	}

	msg := reconciler.CleanupEventMessage(rule, "")

	if !containsStr(msg, "dry-run") {
		t.Errorf("dry-run message should mention dry-run: %q", msg)
	}
	if !containsStr(msg, "NOT deleted") {
		t.Errorf("dry-run message should clarify resource was not deleted: %q", msg)
	}
}

// ── ValidationAction helpers ──────────────────────────────────────────────────

func TestValidationAction_Cleanup(t *testing.T) {
	if !orktypes.ValidationActionCleanup.IsCleanup() {
		t.Error("cleanup action should report IsCleanup true")
	}
	if orktypes.ValidationActionCleanup.IsDeny() {
		t.Error("cleanup action should report IsDeny false")
	}
	if orktypes.ValidationActionCleanup.IsWarn() {
		t.Error("cleanup action should report IsWarn false")
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// newFakeKubeclient wraps a dynamic fake client in a minimal kubeclient shim
// for testing ExecuteCleanup. Adjust this to match your actual kubeclient API.
func newFakeKubeclient(client *dynamicfake.FakeDynamicClient) *kubeclient.Kubeclient {
	// In your test environment this would use kubeclient.NewFakeKubeclient(client)
	// or however your kubeclient package exposes test fakes.
	// This is a placeholder showing the intent.
	return nil // replace with your actual fake constructor
}
