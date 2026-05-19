// pkg/note/kube_pods_test.go
package note

import (
	"testing"
)

// enrichedDeployment builds a minimal enriched resource map as children.go
// would produce it — with _pods embedded.
func enrichedDeployment(pods []map[string]interface{}) map[string]interface{} {
	raw := make([]interface{}, len(pods))
	for i, p := range pods {
		raw[i] = p
	}
	return map[string]interface{}{
		"metadata": map[string]interface{}{"name": "web"},
		"_pods":    raw,
	}
}

func pod(name, ip, phase string, ready bool, restartCount int64) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"ip":           ip,
		"phase":        phase,
		"ready":        ready,
		"node":         "node-1",
		"restartCount": restartCount,
	}
}

func TestNotePodNames(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want string
	}{
		{
			name: "two pods",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "10.0.0.1", "Running", true, 0),
				pod("web-def", "10.0.0.2", "Running", true, 0),
			}),
			want: "web-abc, web-def",
		},
		{
			name: "single pod",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "10.0.0.1", "Running", true, 0),
			}),
			want: "web-abc",
		},
		{
			name: "no pods (enrichment empty)",
			obj:  enrichedDeployment(nil),
			want: "",
		},
		{
			name: "_pods key absent",
			obj:  map[string]interface{}{"metadata": map[string]interface{}{"name": "web"}},
			want: "",
		},
		{
			name: "nil obj",
			obj:  nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notePodNames(tt.obj)
			if got != tt.want {
				t.Errorf("notePodNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNotePodIPs(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want string
	}{
		{
			name: "two pods with IPs",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "10.0.0.1", "Running", true, 0),
				pod("web-def", "10.0.0.2", "Running", true, 0),
			}),
			want: "10.0.0.1, 10.0.0.2",
		},
		{
			name: "pod with empty IP (not yet assigned)",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "", "Pending", false, 0),
			}),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notePodIPs(tt.obj)
			if got != tt.want {
				t.Errorf("notePodIPs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNotePodCount(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want int
	}{
		{
			name: "three pods",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-0", "10.0.0.1", "Running", true, 0),
				pod("web-1", "10.0.0.2", "Running", true, 0),
				pod("web-2", "10.0.0.3", "Running", false, 0),
			}),
			want: 3,
		},
		{
			name: "no pods",
			obj:  enrichedDeployment(nil),
			want: 0,
		},
		{
			name: "nil obj",
			obj:  nil,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notePodCount(tt.obj)
			if got != tt.want {
				t.Errorf("notePodCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNoteReadyPodCount(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want int
	}{
		{
			name: "two of three ready",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-0", "10.0.0.1", "Running", true, 0),
				pod("web-1", "10.0.0.2", "Running", true, 0),
				pod("web-2", "10.0.0.3", "Pending", false, 0),
			}),
			want: 2,
		},
		{
			name: "none ready",
			obj:  enrichedDeployment(nil),
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteReadyPodCount(tt.obj)
			if got != tt.want {
				t.Errorf("noteReadyPodCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNoteHasCrashingPod(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want bool
	}{
		{
			name: "pod with 3 restarts → crashing",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "10.0.0.1", "Running", false, 3),
			}),
			want: true,
		},
		{
			name: "pod with 2 restarts → not crashing (threshold is > 2)",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "10.0.0.1", "Running", true, 2),
			}),
			want: false,
		},
		{
			name: "all pods healthy",
			obj: enrichedDeployment([]map[string]interface{}{
				pod("web-abc", "10.0.0.1", "Running", true, 0),
				pod("web-def", "10.0.0.2", "Running", true, 1),
			}),
			want: false,
		},
		{
			name: "no pods",
			obj:  enrichedDeployment(nil),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noteHasCrashingPod(tt.obj)
			if got != tt.want {
				t.Errorf("noteHasCrashingPod() = %v, want %v", got, tt.want)
			}
		})
	}
}
