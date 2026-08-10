package intake

import (
	"reflect"
	"testing"
)

func TestMatchedWatchFiles_EmptyWatchMatchesEverything(t *testing.T) {
	changed := []string{"services/foo/intent.yaml", "README.md"}
	got := MatchedWatchFiles(nil, changed)
	if !reflect.DeepEqual(got, changed) {
		t.Errorf("got %v, want %v", got, changed)
	}
}

func TestMatchedWatchFiles_FiltersByPattern(t *testing.T) {
	watch := []string{"services/*/intent.yaml"}
	changed := []string{"services/foo/intent.yaml", "services/foo/README.md", "docs/intro.md"}
	got := MatchedWatchFiles(watch, changed)
	want := []string{"services/foo/intent.yaml"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMatchedWatchFiles_MultiplePatternsUnion(t *testing.T) {
	watch := []string{"services/*/intent.yaml", "*.json"}
	changed := []string{"services/foo/intent.yaml", "top-level.json", "unrelated.go"}
	got := MatchedWatchFiles(watch, changed)
	want := []string{"services/foo/intent.yaml", "top-level.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMatchedWatchFiles_InvalidPatternSkipped(t *testing.T) {
	watch := []string{"["} // malformed glob
	changed := []string{"anything.yaml"}
	got := MatchedWatchFiles(watch, changed)
	if len(got) != 0 {
		t.Errorf("got %v, want no matches for an invalid pattern", got)
	}
}

func TestMatchedWatchFiles_NoMatches(t *testing.T) {
	watch := []string{"services/*/intent.yaml"}
	changed := []string{"docs/intro.md"}
	got := MatchedWatchFiles(watch, changed)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestCollectChangedFiles_DedupesPreservingOrder(t *testing.T) {
	got := CollectChangedFiles(
		[]string{"a.yaml", "b.yaml"},
		[]string{"b.yaml", "c.yaml"},
	)
	want := []string{"a.yaml", "b.yaml", "c.yaml"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCollectChangedFiles_Empty(t *testing.T) {
	got := CollectChangedFiles()
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}
