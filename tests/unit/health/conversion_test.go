// health/conversion_logic_test.go
package health_test

import (
	"encoding/json"
	"testing"

	"github.com/ialexeze/orkestra/pkg/health"
	"github.com/ialexeze/orkestra/pkg/katalog"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// buildWebsiteV1Alpha1 returns a minimal v1alpha1 Website object
// as it would arrive in a ConversionReview from Kubernetes.
func buildWebsiteV1Alpha1() map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "demo.orkestra.io/v1alpha1",
		"kind":       "Website",
		"metadata": map[string]interface{}{
			"name":      "my-blog",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"image":    "myorg/nginx:1.25",
			"replicas": int64(2),
			"port":     "80",
			"theme":    "dark",
		},
	}
}

// buildWebsiteV1 returns a minimal v1 Website object.
func buildWebsiteV1() map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "demo.orkestra.io/v1",
		"kind":       "Website",
		"metadata": map[string]interface{}{
			"name":      "my-blog",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"image":    "myorg/nginx:1.25",
			"replicas": int64(2),
			"port":     "80",
			"seo": map[string]interface{}{
				"enabled": false,
			},
		},
	}
}

// websiteRules returns the ConversionRules for the Website CRD.
func websiteRules() *orktypes.ConversionRules {
	return &orktypes.ConversionRules{
		Kind:           "Website",
		StorageVersion: "v1",
		Paths: []orktypes.ConversionPath{
			{
				From: "v1alpha1",
				To:   "v1",
				Spec: map[string]interface{}{
					"image":    "{{ .spec.image }}",
					"replicas": "{{ .spec.replicas }}",
					"port":     "{{ .spec.port }}",
					"seo": map[string]interface{}{
						"enabled": false,
					},
				},
			},
			{
				From: "v1",
				To:   "v1alpha1",
				Spec: map[string]interface{}{
					"image":    "{{ .spec.image }}",
					"replicas": "{{ .spec.replicas }}",
					"port":     "{{ .spec.port }}",
					"theme":    "default",
				},
			},
		},
	}
}

// ── Up-conversion: v1alpha1 → v1 ─────────────────────────────────────────────

func TestApplyConversion_UpConversion_V1alpha1ToV1(t *testing.T) {
	obj := buildWebsiteV1Alpha1()
	rules := websiteRules()

	// targetAPIVersion is the full string Kubernetes sends in DesiredAPIVersion
	result, err := health.ExportedApplyConversionfunc(obj, rules, "demo.orkestra.io/v1")
	if err != nil {
		t.Fatalf("up-conversion failed: %v", err)
	}

	// apiVersion must be updated to the full target apiVersion
	if result["apiVersion"] != "demo.orkestra.io/v1" {
		t.Errorf("apiVersion: expected %q, got %q", "demo.orkestra.io/v1", result["apiVersion"])
	}

	spec, ok := result["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec is not a map")
	}

	// Fields preserved from source
	if spec["image"] != "myorg/nginx:1.25" {
		t.Errorf("image: expected myorg/nginx:1.25, got %v", spec["image"])
	}

	// seo.enabled must be present (default from conversion path)
	seo, ok := spec["seo"].(map[string]interface{})
	if !ok {
		t.Fatal("spec.seo is missing or not a map")
	}
	if seo["enabled"] != false {
		t.Errorf("seo.enabled: expected false, got %v", seo["enabled"])
	}

	// theme must NOT be present in v1
	if _, exists := spec["theme"]; exists {
		t.Error("spec.theme should not exist in v1 output")
	}
}

// ── Down-conversion: v1 → v1alpha1 ───────────────────────────────────────────

func TestApplyConversion_DownConversion_V1ToV1alpha1(t *testing.T) {
	obj := buildWebsiteV1()
	rules := websiteRules()

	// Kubernetes sends the full apiVersion string as DesiredAPIVersion
	result, err := health.ExportedApplyConversionfunc(obj, rules, "demo.orkestra.io/v1alpha1")
	if err != nil {
		t.Fatalf("down-conversion failed: %v", err)
	}

	if result["apiVersion"] != "demo.orkestra.io/v1alpha1" {
		t.Errorf("apiVersion: expected %q, got %q", "demo.orkestra.io/v1alpha1", result["apiVersion"])
	}

	spec, ok := result["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec is not a map")
	}

	// theme must be present in v1alpha1 (default from conversion path)
	if spec["theme"] != "default" {
		t.Errorf("theme: expected %q, got %v", "default", spec["theme"])
	}

	// seo must NOT be present in v1alpha1
	if _, exists := spec["seo"]; exists {
		t.Error("spec.seo should not exist in v1alpha1 output")
	}

	// Fields preserved from source
	if spec["image"] != "myorg/nginx:1.25" {
		t.Errorf("image: expected myorg/nginx:1.25, got %v", spec["image"])
	}
}

// ── No-op: same version ───────────────────────────────────────────────────────

func TestApplyConversion_SameVersion_NoOp(t *testing.T) {
	obj := buildWebsiteV1()
	rules := websiteRules()

	// Requesting the same version as the source — should be a no-op
	result, err := health.ExportedApplyConversionfunc(obj, rules, "demo.orkestra.io/v1")
	if err != nil {
		t.Fatalf("no-op conversion failed: %v", err)
	}

	if result["apiVersion"] != "demo.orkestra.io/v1" {
		t.Errorf("apiVersion changed during no-op: got %q", result["apiVersion"])
	}
}

// ── Missing path ──────────────────────────────────────────────────────────────

func TestApplyConversion_MissingPath_Error(t *testing.T) {
	obj := buildWebsiteV1Alpha1()
	rules := &orktypes.ConversionRules{
		Kind:           "Website",
		StorageVersion: "v1",
		Paths: []orktypes.ConversionPath{
			// Only up-conversion declared — down-conversion is missing
			{From: "v1alpha1", To: "v1", Spec: map[string]interface{}{"image": "{{ .spec.image }}"}},
		},
	}

	_, err := health.ExportedApplyConversionfunc(obj, rules, "demo.orkestra.io/v1beta1")
	if err == nil {
		t.Error("expected error for missing conversion path")
	}
}

// ── Full apiVersion string parsing ────────────────────────────────────────────

func TestApplyConversion_BareVersionTarget(t *testing.T) {
	// Core group resources use bare version strings (no group prefix)
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata":   map[string]interface{}{"name": "test"},
		"spec":       map[string]interface{}{"schedulerName": "default"},
	}

	rules := &orktypes.ConversionRules{
		Kind:           "Pod",
		StorageVersion: "v1",
		Paths: []orktypes.ConversionPath{
			{
				From: "v1alpha1",
				To:   "v1",
				Spec: map[string]interface{}{"schedulerName": "{{ .spec.schedulerName }}"},
			},
		},
	}

	// Same version — no-op, should not error even with bare version string
	result, err := health.ExportedApplyConversionfunc(obj, rules, "v1")
	if err != nil {
		t.Fatalf("bare version no-op failed: %v", err)
	}
	if result["apiVersion"] != "v1" {
		t.Errorf("expected apiVersion=v1, got %q", result["apiVersion"])
	}
}

// ── ConversionRules.FindPath ──────────────────────────────────────────────────

func TestFindPath_ExactMatch(t *testing.T) {
	rules := websiteRules()

	p := rules.FindPath("v1alpha1", "v1")
	if p == nil {
		t.Fatal("expected path for v1alpha1 → v1")
	}
	if p.From != "v1alpha1" || p.To != "v1" {
		t.Errorf("wrong path returned: from=%q to=%q", p.From, p.To)
	}
}

func TestFindPath_ReverseDirection(t *testing.T) {
	rules := websiteRules()

	p := rules.FindPath("v1", "v1alpha1")
	if p == nil {
		t.Fatal("expected path for v1 → v1alpha1")
	}
	if p.From != "v1" || p.To != "v1alpha1" {
		t.Errorf("wrong path returned: from=%q to=%q", p.From, p.To)
	}
}

func TestFindPath_NotFound(t *testing.T) {
	rules := websiteRules()

	p := rules.FindPath("v1beta1", "v1")
	if p != nil {
		t.Error("expected nil for missing path")
	}
}

// ── ConversionReview round-trip ───────────────────────────────────────────────

func TestConversionReview_RoundTrip(t *testing.T) {
	// Simulate the full ConversionReview flow as Kubernetes would call it.
	// This tests the handler logic end-to-end without a live HTTP server.

	obj := buildWebsiteV1Alpha1()
	objJSON, _ := json.Marshal(obj)

	review := ConversionReview{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "ConversionReview",
		Request: &ConversionReviewRequest{
			UID:               "test-uid-123",
			DesiredAPIVersion: "demo.orkestra.io/v1", // full apiVersion string
			Objects:           []json.RawMessage{objJSON},
		},
	}

	rules := websiteRules()
	registry := katalog.NewInMemoryRegistryForTest()
	registry.RegisterConversionRules(rules)

	response := health.ProcessConversionReviewForTest(review, registry)

	if response.Response.Result.Status != "Success" {
		t.Errorf("expected Success, got %q: %s",
			response.Response.Result.Status,
			response.Response.Result.Message)
	}

	if len(response.Response.ConvertedObjects) != 1 {
		t.Fatalf("expected 1 converted object, got %d", len(response.Response.ConvertedObjects))
	}

	var converted map[string]interface{}
	if err := json.Unmarshal(response.Response.ConvertedObjects[0], &converted); err != nil {
		t.Fatalf("unmarshalling converted object: %v", err)
	}

	if converted["apiVersion"] != "demo.orkestra.io/v1" {
		t.Errorf("converted apiVersion: expected demo.orkestra.io/v1, got %q",
			converted["apiVersion"])
	}
}

// ── Type aliases for test access ─────────────────────────────────────────────
// These match the types in health/conversion.go
type ConversionReview = health.ConversionReview
type ConversionReviewRequest = health.ConversionReviewRequest
