package profiles_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/profiles"
)

func TestProbeProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		ok      bool
		want    profiles.ProbeTimings
	}{
		{
			name:    "fast",
			profile: "fast",
			ok:      true,
			want:    profiles.ProbeTimings{InitialDelaySeconds: 5, PeriodSeconds: 10, FailureThreshold: 2, SuccessThreshold: 1, TimeoutSeconds: 5},
		},
		{
			name:    "standard",
			profile: "standard",
			ok:      true,
			want:    profiles.ProbeTimings{InitialDelaySeconds: 15, PeriodSeconds: 20, FailureThreshold: 3, SuccessThreshold: 1, TimeoutSeconds: 10},
		},
		{
			name:    "patient",
			profile: "patient",
			ok:      true,
			want:    profiles.ProbeTimings{InitialDelaySeconds: 30, PeriodSeconds: 30, FailureThreshold: 5, SuccessThreshold: 1, TimeoutSeconds: 10},
		},
		{
			name:    "slow-start",
			profile: "slow-start",
			ok:      true,
			want:    profiles.ProbeTimings{InitialDelaySeconds: 0, PeriodSeconds: 10, FailureThreshold: 30, SuccessThreshold: 1, TimeoutSeconds: 10},
		},
		{
			name:    "unknown returns false",
			profile: "turbo",
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := profiles.ApplyProbeProfile(tt.profile)
			if ok != tt.ok {
				t.Fatalf("ok: want %v got %v", tt.ok, ok)
			}
			if ok && got != tt.want {
				t.Errorf("timings: want %+v got %+v", tt.want, got)
			}
		})
	}
}

func TestDefaultProbeTimingsMatchStandard(t *testing.T) {
	standard, ok := profiles.ApplyProbeProfile("standard")
	if !ok {
		t.Fatal("standard profile not found")
	}
	if profiles.DefaultProbeTimings != standard {
		t.Errorf("DefaultProbeTimings does not match standard profile: default=%+v standard=%+v",
			profiles.DefaultProbeTimings, standard)
	}
}

func TestIsValidProbeProfile(t *testing.T) {
	valid := []string{"fast", "standard", "patient", "slow-start"}
	for _, name := range valid {
		if !profiles.IsValidProbeProfile(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{"", "turbo", "FAST"}
	for _, name := range invalid {
		if profiles.IsValidProbeProfile(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
