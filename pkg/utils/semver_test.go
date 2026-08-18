package utils

import (
	"testing"
)

func TestSemverValid(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"1.2.3", true},
		{"v2.0.0", true},
		{"1.2.3-alpha.1", true},
		{"1.2.3+build", true},
		{"latest", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.v, func(t *testing.T) {
			if got := SemverValid(tt.v); got != tt.want {
				t.Errorf("SemverValid(%q) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestSemverMajorMinorPatch(t *testing.T) {
	major, err := SemverMajor("1.2.3")
	if err != nil || major != 1 {
		t.Errorf("SemverMajor: got %d, %v", major, err)
	}
	minor, err := SemverMinor("1.2.3")
	if err != nil || minor != 2 {
		t.Errorf("SemverMinor: got %d, %v", minor, err)
	}
	patch, err := SemverPatch("1.2.3")
	if err != nil || patch != 3 {
		t.Errorf("SemverPatch: got %d, %v", patch, err)
	}
	if _, err := SemverMajor("invalid"); err == nil {
		t.Error("SemverMajor: expected error for invalid input")
	}
}

func TestSemverCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.3.0", -1},
		{"1.2.3", "1.2.3", 0},
		{"2.0.0", "1.9.9", 1},
		{"v1.2.3", "1.2.3", 0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got, err := SemverCompare(tt.a, tt.b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("SemverCompare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
	if _, err := SemverCompare("invalid", "1.0.0"); err == nil {
		t.Error("SemverCompare: expected error for invalid input")
	}
}

func TestSemverCheck(t *testing.T) {
	tests := []struct {
		v, constraint string
		want          bool
	}{
		{"1.2.3", ">=1.0.0", true},
		{"1.2.3", ">=1.0.0,<2.0.0", true},
		{"2.1.0", "^1.0", false},
		{"1.2.3", "~1.2", true},
		{"1.31.0", ">=1.31", true},
		{"1.30.0", ">=1.31", false},
	}
	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			got, err := SemverCheck(tt.v, tt.constraint)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("SemverCheck(%q, %q) = %v, want %v", tt.v, tt.constraint, got, tt.want)
			}
		})
	}
	if _, err := SemverCheck("invalid", ">=1.0.0"); err == nil {
		t.Error("SemverCheck: expected error for invalid version")
	}
	if _, err := SemverCheck("1.0.0", "!!"); err == nil {
		t.Error("SemverCheck: expected error for invalid constraint")
	}
}

func TestSemverBump(t *testing.T) {
	tests := []struct {
		v, component, want string
	}{
		{"1.2.3", "patch", "1.2.4"},
		{"1.2.3", "minor", "1.3.0"},
		{"1.2.3", "major", "2.0.0"},
		{"v1.2.3", "patch", "1.2.4"},
		{"1.2.3-alpha", "patch", "1.2.4"},
	}
	for _, tt := range tests {
		t.Run(tt.v+"_"+tt.component, func(t *testing.T) {
			got, err := SemverBump(tt.v, tt.component)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("SemverBump(%q, %q) = %q, want %q", tt.v, tt.component, got, tt.want)
			}
		})
	}
	if _, err := SemverBump("invalid", "patch"); err == nil {
		t.Error("SemverBump: expected error for invalid version")
	}
	if _, err := SemverBump("1.2.3", "unknown"); err == nil {
		t.Error("SemverBump: expected error for unknown component")
	}
}
