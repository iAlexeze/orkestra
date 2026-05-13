package utils

import (
	"strings"
	"testing"
)

func TestCenter_ShortLine(t *testing.T) {
	out := Center("hi")
	// Must be padded to ~30 chars width (centered in 60-col field)
	if !strings.Contains(out, "hi") {
		t.Error("Center must contain the original text")
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		// Leading spaces must exist for centering
		if !strings.HasPrefix(l, " ") {
			t.Errorf("centered line should have leading spaces, got %q", l)
		}
	}
}

func TestCenter_EmptyString(t *testing.T) {
	out := Center("")
	// Empty input: no lines, or only blank lines
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			t.Errorf("empty input should yield only blank lines, got %q", l)
		}
	}
}

func TestCenter_MultiLine(t *testing.T) {
	input := "line one\nline two"
	out := Center(input)
	if !strings.Contains(out, "line one") || !strings.Contains(out, "line two") {
		t.Error("Center must preserve all lines")
	}
}

func TestCenter_BlankLinePreserved(t *testing.T) {
	input := "a\n\nb"
	out := Center(input)
	// Blank line in the middle should appear as an empty line in output
	lines := strings.Split(out, "\n")
	hasBlank := false
	for _, l := range lines {
		if l == "" {
			hasBlank = true
			break
		}
	}
	if !hasBlank {
		t.Errorf("blank line in input must produce blank line in output; got %q", out)
	}
}
