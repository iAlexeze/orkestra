// pkg/note/kube_replica_test.go
package note

import (
	"testing"
)

func TestNoteDesiredReplicas(t *testing.T) {
	tests := []struct {
		name   string
		obj    interface{}
		wanted int
	}{
		{
			name: "spec.replicas = 3",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			wanted: 3,
		},
		{
			name: "spec.replicas missing (default 1)",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{},
			},
			wanted: 1,
		},
		{
			name: "spec.replicas = 0 (scale-to-zero)",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 0,
				},
			},
			wanted: 0,
		},
		{
			name:   "nil obj",
			obj:    nil,
			wanted: 1, // fallback
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteDesiredReplicas(tt.obj)
			if got != tt.wanted {
				t.Errorf("noteDesiredReplicas() = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func TestNoteReadyReplicas(t *testing.T) {
	tests := []struct {
		name   string
		obj    interface{}
		wanted int
	}{
		{
			name: "status.readyReplicas = 2",
			obj: map[string]interface{}{
				"status": map[string]interface{}{
					"readyReplicas": 2,
				},
			},
			wanted: 2,
		},
		{
			name: "status missing readyReplicas",
			obj: map[string]interface{}{
				"status": map[string]interface{}{},
			},
			wanted: 0,
		},
		{
			name:   "status missing",
			obj:    map[string]interface{}{},
			wanted: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteReadyReplicas(tt.obj)
			if got != tt.wanted {
				t.Errorf("noteReadyReplicas() = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func TestNoteAvailableReplicas(t *testing.T) {
	tests := []struct {
		name   string
		obj    interface{}
		wanted int
	}{
		{
			name: "status.availableReplicas = 2",
			obj: map[string]interface{}{
				"status": map[string]interface{}{
					"availableReplicas": 2,
				},
			},
			wanted: 2,
		},
		{
			name: "status missing availableReplicas",
			obj: map[string]interface{}{
				"status": map[string]interface{}{},
			},
			wanted: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteAvailableReplicas(tt.obj)
			if got != tt.wanted {
				t.Errorf("noteAvailableReplicas() = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func TestNoteUpdatedReplicas(t *testing.T) {
	tests := []struct {
		name   string
		obj    interface{}
		wanted int
	}{
		{
			name: "status.updatedReplicas = 3",
			obj: map[string]interface{}{
				"status": map[string]interface{}{
					"updatedReplicas": 3,
				},
			},
			wanted: 3,
		},
		{
			name: "status missing updatedReplicas",
			obj: map[string]interface{}{
				"status": map[string]interface{}{},
			},
			wanted: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteUpdatedReplicas(tt.obj)
			if got != tt.wanted {
				t.Errorf("noteUpdatedReplicas() = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func TestNoteReplicasReady(t *testing.T) {
	tests := []struct {
		name   string
		obj    interface{}
		wanted bool
	}{
		{
			name: "ready == desired -> true",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
				"status": map[string]interface{}{
					"readyReplicas": 3,
				},
			},
			wanted: true,
		},
		{
			name: "ready < desired -> false",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
				"status": map[string]interface{}{
					"readyReplicas": 2,
				},
			},
			wanted: false,
		},
		{
			name: "zero replicas and zero ready -> true (scale-to-zero)",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 0,
				},
				"status": map[string]interface{}{
					"readyReplicas": 0,
				},
			},
			wanted: true,
		},
		{
			name: "desired missing (default 1) but ready=0 -> false",
			obj: map[string]interface{}{
				"status": map[string]interface{}{
					"readyReplicas": 0,
				},
			},
			wanted: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteReplicasReady(tt.obj)
			if got != tt.wanted {
				t.Errorf("noteReplicasReady() = %v, want %v", got, tt.wanted)
			}
		})
	}
}
