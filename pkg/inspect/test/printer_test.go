// pkg/inspect/printer_test.go
package inspect_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ialexeze/orkestra/pkg/inspect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHumanAge(t *testing.T) {
	tests := []struct {
		name     string
		age      time.Duration
		expected string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 2 * 24 * time.Hour, "2d"},
		{"weeks", 14 * 24 * time.Hour, "2w"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := metav1.Time{Time: time.Now().Add(-tt.age)}
			got := inspect.HumanAge(ts)
			if got != tt.expected {
				t.Errorf("HumanAge(%v): expected %q, got %q", tt.age, tt.expected, got)
			}
		})
	}
}

func TestHumanAge_Zero(t *testing.T) {
	got := inspect.HumanAge(metav1.Time{})
	if got != "<unknown>" {
		t.Errorf("expected <unknown> for zero time, got %q", got)
	}
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer

	header := []string{"NAME", "STATUS", "AGE"}
	rows := [][]string{
		{"my-website", "Ready", "5m"},
		{"my-blog", "Pending", "2d"},
	}

	inspect.PrintTable(&buf, header, rows)

	output := buf.String()

	// Header should appear
	if !strings.Contains(output, "NAME") {
		t.Error("expected NAME in output")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("expected STATUS in output")
	}

	// Data rows should appear
	if !strings.Contains(output, "my-website") {
		t.Error("expected my-website in output")
	}
	if !strings.Contains(output, "my-blog") {
		t.Error("expected my-blog in output")
	}
	if !strings.Contains(output, "Ready") {
		t.Error("expected Ready in output")
	}
}

// func TestPrintTable_Empty(t *testing.T) {
// 	var buf bytes.Buffer
// 	inspect.PrintTable(&buf, []string{"NAME", "STATUS"}, [][]string{})
// 	// Should not panic and should produce only the header
// 	output := buf.String()
// 	if !strings.Contains(output, "NAME") {
// 		t.Error("expected header even for empty table")
// 	}
// }

// func TestExtractStatus_Phase(t *testing.T) {
// 	// Build a fake unstructured object with status.phase
// 	obj := buildUnstructured(map[string]interface{}{
// 		"status": map[string]interface{}{
// 			"phase": "Ready",
// 		},
// 	})

// 	got := inspect.ExtractStatus(obj)
// 	if got != "Ready" {
// 		t.Errorf("expected Ready, got %q", got)
// 	}
// }

// func TestExtractStatus_State(t *testing.T) {
// 	obj := buildUnstructured(map[string]interface{}{
// 		"status": map[string]interface{}{
// 			"state": "Active",
// 		},
// 	})

// 	got := inspect.ExtractStatus(obj)
// 	if got != "Active" {
// 		t.Errorf("expected Active, got %q", got)
// 	}
// }

// func TestExtractStatus_Conditions(t *testing.T) {
// 	obj := buildUnstructured(map[string]interface{}{
// 		"status": map[string]interface{}{
// 			"conditions": []interface{}{
// 				map[string]interface{}{
// 					"type":   "Ready",
// 					"status": "True",
// 				},
// 			},
// 		},
// 	})

// 	got := inspect.ExtractStatus(obj)
// 	if got != "Ready=True" {
// 		t.Errorf("expected Ready=True, got %q", got)
// 	}
// }

// func TestExtractStatus_NoStatus(t *testing.T) {
// 	obj := buildUnstructured(map[string]interface{}{
// 		"spec": map[string]interface{}{
// 			"image": "nginx:1.25",
// 		},
// 	})

// 	got := inspect.ExtractStatus(obj)
// 	if got != "Unknown" {
// 		t.Errorf("expected Unknown, got %q", got)
// 	}
// }

func TestHealthIcon(t *testing.T) {
	tests := []struct {
		status   string
		hasGreen bool
		hasRed   bool
	}{
		{"ready", true, false},
		{"running", true, false},
		{"healthy", true, false},
		{"error", false, true},
		{"failed", false, true},
		{"degraded", false, true},
		{"unknown", false, false}, // gray — neither green nor red
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := inspect.HealthIcon(tt.status)
			hasGreen := strings.Contains(icon, "\033[32m")
			hasRed := strings.Contains(icon, "\033[31m")

			if tt.hasGreen && !hasGreen {
				t.Errorf("status %q: expected green icon", tt.status)
			}
			if tt.hasRed && !hasRed {
				t.Errorf("status %q: expected red icon", tt.status)
			}
		})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// buildUnstructured creates a minimal unstructured object for testing.
func buildUnstructured(extra map[string]interface{}) *unstructuredObj {
	obj := map[string]interface{}{
		"apiVersion": "demo.orkestra.io/v1alpha1",
		"kind":       "Website",
		"metadata": map[string]interface{}{
			"name":      "test-website",
			"namespace": "default",
		},
	}
	for k, v := range extra {
		obj[k] = v
	}
	return &unstructuredObj{Object: obj}
}

// unstructuredObj is a minimal stand-in for *unstructured.Unstructured in tests.
type unstructuredObj struct {
	Object map[string]interface{}
}
