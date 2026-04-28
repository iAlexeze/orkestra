// pkg/note/random_test.go
package note

import (
	"regexp"
	"testing"
)

func TestRandomAlphanumeric(t *testing.T) {
	tests := []struct {
		name      string
		n         int
		wantLen   int
		wantEmpty bool
	}{
		{"positive length", 10, 10, false},
		{"zero length", 0, 0, true},
		{"negative length", -5, 0, true},
		{"large length", 100, 100, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomAlphanumeric(tt.n)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("randomAlphanumeric(%d) = %q, want empty", tt.n, got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("randomAlphanumeric(%d) length = %d, want %d", tt.n, len(got), tt.wantLen)
			}
			// check character set
			matched, _ := regexp.MatchString("^[a-zA-Z0-9]+$", got)
			if !matched {
				t.Errorf("randomAlphanumeric(%d) = %q, contains invalid characters", tt.n, got)
			}
		})
	}
	t.Run("uniqueness", func(t *testing.T) {
		s1 := randomAlphanumeric(32)
		s2 := randomAlphanumeric(32)
		if s1 == s2 {
			t.Errorf("randomAlphanumeric produced same string twice: %q", s1)
		}
	})
}

func TestRandomHex(t *testing.T) {
	tests := []struct {
		name      string
		n         int
		wantLen   int // 2 * n
		wantEmpty bool
	}{
		{"positive length", 16, 32, false},
		{"zero length", 0, 0, true},
		{"negative length", -3, 0, true},
		{"odd n produces even length", 7, 14, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomHex(tt.n)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("randomHex(%d) = %q, want empty", tt.n, got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("randomHex(%d) length = %d, want %d", tt.n, len(got), tt.wantLen)
			}
			matched, _ := regexp.MatchString("^[0-9a-f]+$", got)
			if !matched {
				t.Errorf("randomHex(%d) = %q, contains non-hex characters", tt.n, got)
			}
		})
	}
	t.Run("uniqueness", func(t *testing.T) {
		h1 := randomHex(20)
		h2 := randomHex(20)
		if h1 == h2 {
			t.Errorf("randomHex produced same string twice: %q", h1)
		}
	})
}

func TestRandomBase64(t *testing.T) {
	tests := []struct {
		name      string
		n         int
		wantLen   int // base64 length = 4 * ceil(n/3)
		wantEmpty bool
	}{
		{"positive length", 18, 24, false},
		{"zero length", 0, 0, true},
		{"negative length", -1, 0, true},
		{"n=1 gives length 4", 1, 4, false},
		{"n=2 gives length 4", 2, 4, false},
		{"n=3 gives length 4", 3, 4, false},
		{"n=4 gives length 8", 4, 8, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := randomBase64(tt.n)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("randomBase64(%d) = %q, want empty", tt.n, got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("randomBase64(%d) length = %d, want %d", tt.n, len(got), tt.wantLen)
			}
			// URL-safe base64 allows A-Z a-z 0-9 - _ and padding =
			matched, _ := regexp.MatchString("^[A-Za-z0-9_-]+=*$", got)
			if !matched {
				t.Errorf("randomBase64(%d) = %q, contains invalid characters", tt.n, got)
			}
		})
	}
	t.Run("uniqueness", func(t *testing.T) {
		b1 := randomBase64(24)
		b2 := randomBase64(24)
		if b1 == b2 {
			t.Errorf("randomBase64 produced same string twice: %q", b1)
		}
	})
}
