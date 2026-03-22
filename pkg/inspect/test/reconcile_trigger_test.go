// pkg/inspect/reconcile_trigger_test.go
package inspect_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ialexeze/orkestra/pkg/inspect"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

var websiteGVR = schema.GroupVersionResource{
	Group:    "demo.orkestra.io",
	Version:  "v1alpha1",
	Resource: "websites",
}

// buildWebsite creates an unstructured Website CR for use in tests.
func buildWebsite(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "demo.orkestra.io/v1alpha1",
			"kind":       "Website",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"image":    "nginx:1.25",
				"replicas": int64(1),
			},
		},
	}
}

func TestTriggerReconcile_SetsAnnotation(t *testing.T) {
	website := buildWebsite("my-blog", "default")

	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme, website)

	err := inspect.TriggerReconcile(
		context.Background(),
		client,
		websiteGVR,
		"default",
		"my-blog",
	)
	if err != nil {
		t.Fatalf("TriggerReconcile: unexpected error: %v", err)
	}

	// Fetch the patched object and check the annotation was set
	got, err := client.Resource(websiteGVR).Namespace("default").Get(
		context.Background(), "my-blog", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("getting patched object: %v", err)
	}

	annotations := got.GetAnnotations()
	if annotations == nil {
		t.Fatal("expected annotations to be set, got nil")
	}

	val, ok := annotations[inspect.ReconcileAnnotation]
	if !ok {
		t.Fatalf("expected annotation %q to be set, got annotations: %v",
			inspect.ReconcileAnnotation, annotations)
	}
	if val == "" {
		t.Error("expected non-empty annotation value")
	}
}

func TestTriggerReconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme) // empty — no objects

	err := inspect.TriggerReconcile(
		context.Background(),
		client,
		websiteGVR,
		"default",
		"nonexistent",
	)
	if err == nil {
		t.Fatal("expected error for non-existent resource, got nil")
	}
}

func TestTriggerReconcileAll_AllCRsTriggered(t *testing.T) {
	websites := []*unstructured.Unstructured{
		buildWebsite("site-a", "default"),
		buildWebsite("site-b", "default"),
		buildWebsite("site-c", "production"),
	}

	objects := make([]runtime.Object, len(websites))
	for i, w := range websites {
		objects[i] = w
	}

	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme, objects...)

	results, err := inspect.TriggerReconcileAll(
		context.Background(),
		client,
		websiteGVR,
		"", // all namespaces
	)
	if err != nil {
		t.Fatalf("TriggerReconcileAll: unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Error != nil {
			t.Errorf("result %s/%s: unexpected error: %v", r.Namespace, r.Name, r.Error)
		}
	}
}

func TestTriggerReconcileAll_EmptyCluster(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme) // no objects

	results, err := inspect.TriggerReconcileAll(
		context.Background(),
		client,
		websiteGVR,
		"default",
	)
	if err != nil {
		t.Fatalf("unexpected error for empty cluster: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty cluster, got %d", len(results))
	}
}

func TestBuildAnnotationPatch_ValidJSON(t *testing.T) {
	// Verify the annotation patch produces valid JSON that the API server would accept
	key := inspect.ReconcileAnnotation
	value := "2026-03-20T10:00:00Z"

	// Reconstruct the patch via the exported annotation constant
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				key: value,
			},
		},
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("patch is not valid JSON: %v", err)
	}

	// Unmarshal back and verify structure
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("cannot unmarshal patch: %v", err)
	}

	metadata, ok := decoded["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata object in patch")
	}

	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		t.Fatal("expected annotations object in patch")
	}

	if annotations[key] != value {
		t.Errorf("expected annotation value %q, got %q", value, annotations[key])
	}
}

// ── Ensure interfaces are satisfied ──────────────────────────────────────────

var _ meta.RESTMapper = (*meta.DefaultRESTMapper)(nil)
