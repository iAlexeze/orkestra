//go:build !runtime && !gateway

package cli

import "testing"

func TestParseFetchOverrides_Valid(t *testing.T) {
	got, err := parseFetchOverrides([]string{
		"services/a/intent.yaml=local/a.yaml",
		"services/b/intent.yaml=local/b.yaml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["services/a/intent.yaml"] != "local/a.yaml" || got["services/b/intent.yaml"] != "local/b.yaml" {
		t.Errorf("got %+v", got)
	}
}

func TestParseFetchOverrides_Empty(t *testing.T) {
	got, err := parseFetchOverrides(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %+v, want empty", got)
	}
}

func TestParseFetchOverrides_MissingEquals(t *testing.T) {
	_, err := parseFetchOverrides([]string{"no-equals-sign"})
	if err == nil {
		t.Fatal("expected an error for a malformed override")
	}
}

func TestParseFetchOverrides_EmptyPathSide(t *testing.T) {
	_, err := parseFetchOverrides([]string{"=file"})
	if err == nil {
		t.Fatal("expected an error when the path side is empty")
	}
}

func TestParseFetchOverrides_EmptyFileSide(t *testing.T) {
	_, err := parseFetchOverrides([]string{"path="})
	if err == nil {
		t.Fatal("expected an error when the local-file side is empty")
	}
}
