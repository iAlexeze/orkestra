// pkg/merger/validate_pattern_test.go
package merger

import (
	"os"
	"path/filepath"
	"testing"
)

func makeKatalogPatternDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "katalog.yaml"), []byte("kind: Katalog\n"), 0644); err != nil {
		t.Fatalf("writing katalog.yaml: %v", err)
	}
	return dir
}

func TestValidatePatternStructure_KatalogOnly_Valid(t *testing.T) {
	dir := makeKatalogPatternDir(t)
	if err := validatePatternStructure(dir, "https://example.com/registry", "v1.0"); err != nil {
		t.Errorf("katalog.yaml alone must pass: %v", err)
	}
}

func TestValidatePatternStructure_MotifOnly_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "motif.yaml"), []byte("kind: Motif\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validatePatternStructure(dir, "https://example.com/registry", "v1.0"); err != nil {
		t.Errorf("motif.yaml alone must pass: %v", err)
	}
}

func TestValidatePatternStructure_MissingPrimaryFile(t *testing.T) {
	dir := t.TempDir()
	// Only a crd.yaml — no katalog.yaml or motif.yaml
	os.WriteFile(filepath.Join(dir, "crd.yaml"), []byte("content"), 0644)
	if err := validatePatternStructure(dir, "https://example.com/registry", "v1.0"); err == nil {
		t.Error("missing primary file must return error")
	}
}

func TestValidatePatternStructure_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := validatePatternStructure(dir, "https://example.com/registry", "v1.0"); err == nil {
		t.Error("empty dir must return error")
	}
}
