//go:build !runtime && !gateway

package cli

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_IdenticalFiles(t *testing.T) {
	content := []byte("line1\nline2\nline3\n")
	out := unifiedDiff("a.yaml", "b.yaml", content, content, false)
	// Non-verbose: identical files produce only headers, no +/- lines.
	if strings.Contains(out, "+") && !strings.Contains(out, "+++") {
		t.Error("identical files should produce no addition lines in non-verbose mode")
	}
	if strings.Contains(out, "-") && !strings.Contains(out, "---") {
		t.Error("identical files should produce no removal lines in non-verbose mode")
	}
}

func TestUnifiedDiff_IdenticalFilesVerbose(t *testing.T) {
	content := []byte("line1\nline2\n")
	out := unifiedDiff("a.yaml", "b.yaml", content, content, true)
	if !strings.Contains(out, " line1") {
		t.Error("verbose mode should show context lines with leading space")
	}
}

func TestUnifiedDiff_AddedLine(t *testing.T) {
	a := []byte("line1\n")
	b := []byte("line1\nline2\n")
	out := unifiedDiff("a.yaml", "b.yaml", a, b, false)
	if !strings.Contains(out, "+line2") {
		t.Errorf("expected '+line2' in diff output, got:\n%s", out)
	}
}

func TestUnifiedDiff_RemovedLine(t *testing.T) {
	a := []byte("line1\nline2\n")
	b := []byte("line1\n")
	out := unifiedDiff("a.yaml", "b.yaml", a, b, false)
	if !strings.Contains(out, "-line2") {
		t.Errorf("expected '-line2' in diff output, got:\n%s", out)
	}
}

func TestUnifiedDiff_ChangedLine(t *testing.T) {
	a := []byte("foo: bar\n")
	b := []byte("foo: baz\n")
	out := unifiedDiff("a.yaml", "b.yaml", a, b, false)
	if !strings.Contains(out, "-foo: bar") {
		t.Errorf("expected '-foo: bar', got:\n%s", out)
	}
	if !strings.Contains(out, "+foo: baz") {
		t.Errorf("expected '+foo: baz', got:\n%s", out)
	}
}

func TestUnifiedDiff_Headers(t *testing.T) {
	out := unifiedDiff("old.yaml", "new.yaml", []byte("a"), []byte("b"), false)
	if !strings.Contains(out, "--- old.yaml") {
		t.Errorf("expected '--- old.yaml' header, got:\n%s", out)
	}
	if !strings.Contains(out, "+++ new.yaml") {
		t.Errorf("expected '+++ new.yaml' header, got:\n%s", out)
	}
}

func TestUnifiedDiff_BothEmpty(t *testing.T) {
	out := unifiedDiff("a.yaml", "b.yaml", []byte(""), []byte(""), false)
	// Should not panic and should contain headers.
	if !strings.Contains(out, "---") {
		t.Errorf("expected headers even for empty files, got:\n%s", out)
	}
}

func TestUnifiedDiff_AEmpty(t *testing.T) {
	b := []byte("new line\n")
	out := unifiedDiff("a.yaml", "b.yaml", []byte(""), b, false)
	if !strings.Contains(out, "+new line") {
		t.Errorf("expected '+new line' for file added from empty, got:\n%s", out)
	}
}

func TestUnifiedDiff_BEmpty(t *testing.T) {
	a := []byte("old line\n")
	out := unifiedDiff("a.yaml", "b.yaml", a, []byte(""), false)
	if !strings.Contains(out, "-old line") {
		t.Errorf("expected '-old line' for file cleared to empty, got:\n%s", out)
	}
}

func TestUnifiedDiff_MultiLineChanges(t *testing.T) {
	a := []byte("a\nb\nc\n")
	b := []byte("a\nB\nc\n")
	out := unifiedDiff("x", "y", a, b, false)
	if !strings.Contains(out, "-b") {
		t.Errorf("expected '-b', got:\n%s", out)
	}
	if !strings.Contains(out, "+B") {
		t.Errorf("expected '+B', got:\n%s", out)
	}
	// Unchanged lines should not appear in non-verbose output.
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if l == " a" || l == " c" {
			t.Errorf("non-verbose mode should not show context line %q", l)
		}
	}
}
