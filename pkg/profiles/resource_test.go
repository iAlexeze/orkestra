package profiles_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/profiles"
)

func TestResourceProfiles(t *testing.T) {
	tests := []struct {
		name       string
		profile    string
		expectErr  bool
		cpuRequest string
		memRequest string
		cpuLimit   string
		memLimit   string
	}{
		{"tiny", "tiny", false, "25m", "64Mi", "100m", "128Mi"},
		{"small", "small", false, "100m", "128Mi", "500m", "512Mi"},
		{"medium", "medium", false, "250m", "256Mi", "1", "1Gi"},
		{"large", "large", false, "500m", "512Mi", "2", "2Gi"},
		{"burst", "burst", false, "200m", "256Mi", "2", "2Gi"},
		{"steady", "steady", false, "300m", "256Mi", "600m", "512Mi"},
		{"compute-heavy", "compute-heavy", false, "1", "512Mi", "2", "1Gi"},
		{"memory-heavy", "memory-heavy", false, "250m", "1Gi", "500m", "2Gi"},
		{"unknown profile fails fast", "xlarge", true, "", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := profiles.ApplyResourceProfile(tt.profile)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := r.Requests["cpu"]; got != tt.cpuRequest {
				t.Errorf("requests.cpu: want %q got %q", tt.cpuRequest, got)
			}
			if got := r.Requests["memory"]; got != tt.memRequest {
				t.Errorf("requests.memory: want %q got %q", tt.memRequest, got)
			}
			if got := r.Limits["cpu"]; got != tt.cpuLimit {
				t.Errorf("limits.cpu: want %q got %q", tt.cpuLimit, got)
			}
			if got := r.Limits["memory"]; got != tt.memLimit {
				t.Errorf("limits.memory: want %q got %q", tt.memLimit, got)
			}
		})
	}
}

func TestIsValidResourceProfile(t *testing.T) {
	valid := []string{"tiny", "small", "medium", "large", "burst", "steady", "compute-heavy", "memory-heavy"}
	for _, name := range valid {
		if !profiles.IsValidResourceProfile(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	for _, name := range []string{"MEDIUM", "Tiny", "BURST"} {
		if !profiles.IsValidResourceProfile(name) {
			t.Errorf("expected case-insensitive %q to be valid", name)
		}
	}

	invalid := []string{"", "xlarge", "nano"}
	for _, name := range invalid {
		if profiles.IsValidResourceProfile(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
