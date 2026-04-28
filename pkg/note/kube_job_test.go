// pkg/note/kube_job_test.go
package note

import (
	"testing"
)

func TestNoteJobSucceeded(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want bool
	}{
		{"nil", nil, false},
		{"no status", map[string]interface{}{}, false},
		{"status without succeeded", map[string]interface{}{
			"status": map[string]interface{}{},
		}, false},
		{"succeeded = 0", map[string]interface{}{
			"status": map[string]interface{}{
				"succeeded": int64(0),
			},
		}, false},
		{"succeeded = 1", map[string]interface{}{
			"status": map[string]interface{}{
				"succeeded": int64(1),
			},
		}, true},
		{"succeeded = 2", map[string]interface{}{
			"status": map[string]interface{}{
				"succeeded": int64(2),
			},
		}, true},
		{"succeeded as float64", map[string]interface{}{
			"status": map[string]interface{}{
				"succeeded": float64(1),
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteJobSucceeded(tt.obj); got != tt.want {
				t.Errorf("noteJobSucceeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteJobFailed(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want bool
	}{
		{"nil", nil, false},
		{"no status", map[string]interface{}{}, false},
		{"status without failed", map[string]interface{}{
			"status": map[string]interface{}{},
		}, false},
		{"failed = 0", map[string]interface{}{
			"status": map[string]interface{}{
				"failed": int64(0),
			},
		}, false},
		{"failed = 1", map[string]interface{}{
			"status": map[string]interface{}{
				"failed": int64(1),
			},
		}, true},
		{"failed = 2", map[string]interface{}{
			"status": map[string]interface{}{
				"failed": int64(2),
			},
		}, true},
		{"failed as float64", map[string]interface{}{
			"status": map[string]interface{}{
				"failed": float64(1),
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteJobFailed(tt.obj); got != tt.want {
				t.Errorf("noteJobFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteJobActive(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want bool
	}{
		{"nil", nil, false},
		{"no status", map[string]interface{}{}, false},
		{"status without active", map[string]interface{}{
			"status": map[string]interface{}{},
		}, false},
		{"active = 0", map[string]interface{}{
			"status": map[string]interface{}{
				"active": int64(0),
			},
		}, false},
		{"active = 1", map[string]interface{}{
			"status": map[string]interface{}{
				"active": int64(1),
			},
		}, true},
		{"active = 2", map[string]interface{}{
			"status": map[string]interface{}{
				"active": int64(2),
			},
		}, true},
		{"active as float64", map[string]interface{}{
			"status": map[string]interface{}{
				"active": float64(1),
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteJobActive(tt.obj); got != tt.want {
				t.Errorf("noteJobActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
