//go:build integration

// tests/integration/komposer/sources_test.go
// Integration tests for Komposer source resolution and inline override behaviour.
package komposer_test

import (
	"os"
	"testing"

	"github.com/orkspace/orkestra/pkg/merger"
)

// writeKatalogFile creates a minimal Katalog YAML temp file with the given CRD names.
func writeKatalogFile(t *testing.T, name string, crdNames ...string) string {
	t.Helper()
	content := "apiVersion: orkestra.orkspace.io/v1\nkind: Katalog\nmetadata:\n  name: " + name + "\nspec:\n  crds:\n"
	for _, n := range crdNames {
		content += "    - name: " + n + "\n      enabled: true\n"
	}
	f, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// writeKomposerFile creates a Komposer YAML that sources the given Katalog file.
func writeKomposerFile(t *testing.T, name, sourcePath string, inlineCRDs ...string) string {
	t.Helper()
	content := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: " + name + "\nsources:\n  files:\n    - url: " + sourcePath + "\nspec:\n  crds:\n"
	for _, n := range inlineCRDs {
		content += "    - name: " + n + "\n      enabled: true\n"
	}
	if len(inlineCRDs) == 0 {
		content += "    []\n"
	}
	f, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestKomposer_InlineOverride_WinsOverSource(t *testing.T) {
	// Source Katalog has website enabled=true
	sourceKatalog := writeKatalogFile(t, "source", "website", "database")

	// Komposer sources that katalog but overrides website with enabled=false
	komposerContent := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: override-test\nsources:\n  files:\n    - url: " + sourceKatalog + "\nspec:\n  crds:\n    - name: website\n      enabled: false\n"
	f, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	f.WriteString(komposerContent)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	m := merger.New(f.Name())
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}

	// Total CRDs should still be 2 — inline replaces source, not appends
	if m.Count() != 2 {
		t.Errorf("expected 2 total CRDs, got %d", m.Count())
	}

	website, ok := m.Get("website")
	if !ok {
		t.Fatal("expected to find 'website'")
	}
	if website.IsEnabled() {
		t.Error("inline override should have disabled 'website'")
	}

	// database came from source and was not overridden
	db, ok := m.Get("database")
	if !ok {
		t.Fatal("expected to find 'database'")
	}
	if !db.IsEnabled() {
		t.Error("'database' should remain enabled (not overridden)")
	}
}

func TestKomposer_MultipleFileSources_AllMerged(t *testing.T) {
	srcA := writeKatalogFile(t, "source-a", "crd-a1", "crd-a2")
	srcB := writeKatalogFile(t, "source-b", "crd-b1")

	content := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: multi-source\nsources:\n  files:\n    - url: " + srcA + "\n    - url: " + srcB + "\nspec:\n  crds: []\n"
	f, _ := os.CreateTemp("", "*.yaml")
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	m := merger.New(f.Name())
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() failed: %v", err)
	}
	if m.Count() != 3 {
		t.Errorf("expected 3 CRDs from two sources, got %d", m.Count())
	}
}

func TestKomposer_CannotSourceAnotherKomposer(t *testing.T) {
	// A Komposer that references another Komposer as a source — should error
	innerKomposer := `apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: inner-komposer
sources:
  files: []
spec:
  crds:
    - name: inner
      enabled: true
`
	fInner, _ := os.CreateTemp("", "*.yaml")
	fInner.WriteString(innerKomposer)
	fInner.Close()
	t.Cleanup(func() { os.Remove(fInner.Name()) })

	outerContent := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: outer-komposer\nsources:\n  files:\n    - url: " + fInner.Name() + "\nspec:\n  crds: []\n"
	fOuter, _ := os.CreateTemp("", "*.yaml")
	fOuter.WriteString(outerContent)
	fOuter.Close()
	t.Cleanup(func() { os.Remove(fOuter.Name()) })

	m := merger.New(fOuter.Name())
	if err := m.Merge(); err == nil {
		t.Error("expected error: Komposer cannot source another Komposer")
	}
}

func TestKomposer_KatalogCannotDeclareSourcesBlock(t *testing.T) {
	// A Katalog with a sources block is a configuration error
	content := `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: bad-katalog
sources:
  files:
    - url: ./some-file.yaml
spec:
  crds:
    - name: website
      enabled: true
`
	f, _ := os.CreateTemp("", "*.yaml")
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	m := merger.New(f.Name())
	if err := m.Merge(); err == nil {
		t.Error("expected error: kind Katalog cannot declare sources block")
	}
}

func TestKomposer_DuplicateCRD_WithinSingleSource_ReturnsError(t *testing.T) {
	// Katalog with two CRDs of the same name — should error
	content := `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: dup-katalog
spec:
  crds:
    - name: website
      enabled: true
    - name: website
      enabled: false
`
	f, _ := os.CreateTemp("", "*.yaml")
	f.WriteString(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	m := merger.New(f.Name())
	if err := m.Merge(); err == nil {
		t.Error("expected error for duplicate CRD within a single Katalog")
	}
}
