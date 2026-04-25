//go:build integration

// tests/integration/komposer/merge_test.go
// Integration tests for the Merger: loading Katalog and Komposer files from disk,
// resolving sources, merging CRD lists, and enforcing deduplication rules.
package komposer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orkspace/orkestra/pkg/merger"
)

const katalogYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: test-katalog
spec:
  crds:
    - name: website
      enabled: true
    - name: database
      enabled: true
`

const katalogDisabledYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: test-disabled-katalog
spec:
  crds:
    - name: website
      enabled: false
    - name: cache
      enabled: true
`

const komposerWithFileSourceYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: test-komposer
sources:
  files:
    - url: %s
spec:
  crds: []
`

const komposerWithInlineOverrideYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: override-komposer
sources:
  files:
    - url: %s
spec:
  crds:
    - name: website
      enabled: false
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "katalog-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestMerger_SingleKatalog_LoadsAllCRDs(t *testing.T) {
	path := writeTemp(t, katalogYAML)
	m := merger.New(path)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	if m.Count() != 2 {
		t.Errorf("expected 2 CRDs, got %d", m.Count())
	}
}

func TestMerger_DisabledCRD_ExcludedFromEnabled(t *testing.T) {
	path := writeTemp(t, katalogDisabledYAML)
	m := merger.New(path)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	if m.Count() != 2 {
		t.Errorf("All() should return 2 (including disabled), got %d", m.Count())
	}
	if m.EnabledCount() != 1 {
		t.Errorf("Enabled() should return 1, got %d", m.EnabledCount())
	}
}

func TestMerger_GetCRD_ByName(t *testing.T) {
	path := writeTemp(t, katalogYAML)
	m := merger.New(path)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	entry, ok := m.Get("website")
	if !ok {
		t.Fatal("expected to find 'website'")
	}
	if entry.Name != "website" {
		t.Errorf("expected name 'website', got %q", entry.Name)
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Error("should not find 'nonexistent'")
	}
}

func TestMerger_TwoKatalogFiles_Merged(t *testing.T) {
	a := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: katalog-a
spec:
  crds:
    - name: alpha
      enabled: true
`)
	b := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: katalog-b
spec:
  crds:
    - name: beta
      enabled: true
`)
	m := merger.New(a, b)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	if m.Count() != 2 {
		t.Errorf("expected 2 CRDs from two katalog files, got %d", m.Count())
	}
}

func TestMerger_DuplicateCRD_AcrossEntryPoints_ReturnsError(t *testing.T) {
	a := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: katalog-a
spec:
  crds:
    - name: website
      enabled: true
`)
	b := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: katalog-b
spec:
  crds:
    - name: website
      enabled: true
`)
	m := merger.New(a, b)
	if err := m.Merge(); err == nil {
		t.Error("expected error for duplicate CRD 'website' across entry points")
	}
}

func TestMerger_KomposerFileSource_LoadsKatalog(t *testing.T) {
	katalogPath := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: source-katalog
spec:
  crds:
    - name: sourced-crd
      enabled: true
`)
	komposerContent := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: test-komposer\nsources:\n  files:\n    - url: " + katalogPath + "\nspec:\n  crds: []\n"
	komposerPath := writeTemp(t, komposerContent)

	m := merger.New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	if m.Count() != 1 {
		t.Errorf("expected 1 CRD from sourced Katalog, got %d", m.Count())
	}
	if _, ok := m.Get("sourced-crd"); !ok {
		t.Error("expected to find 'sourced-crd'")
	}
}

func TestMerger_Add_AppendsEntryPoints(t *testing.T) {
	a := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: katalog-a
spec:
  crds:
    - name: alpha
      enabled: true
`)
	b := writeTemp(t, `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: katalog-b
spec:
  crds:
    - name: beta
      enabled: true
`)
	m := merger.New(a).Add(b)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	if m.Count() != 2 {
		t.Errorf("expected 2 CRDs after Add(), got %d", m.Count())
	}
}

func TestMerger_NonExistentFile_ReturnsError(t *testing.T) {
	m := merger.New(filepath.Join(os.TempDir(), "does-not-exist-xyz.yaml"))
	if err := m.Merge(); err == nil {
		t.Error("expected error loading non-existent file")
	}
}
