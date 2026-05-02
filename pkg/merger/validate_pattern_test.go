// pkg/merger/validate_pattern_test.go
package merger

import (
	"os"
	"path/filepath"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func makeValidPatternDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range orktypes.RequiredFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644); err != nil {
			t.Fatalf("writing %s: %v", f, err)
		}
	}
	return dir
}

func TestValidatePatternStructure_Valid(t *testing.T) {
	dir := makeValidPatternDir(t)
	if err := validatePatternStructure(dir, "https://example.com/registry", "v1.0"); err != nil {
		t.Errorf("valid pattern must not error: %v", err)
	}
}

func TestValidatePatternStructure_MissingFile(t *testing.T) {
	dir := makeValidPatternDir(t)
	os.Remove(filepath.Join(dir, "crd.yaml"))
	err := validatePatternStructure(dir, "https://example.com/registry", "v1.0")
	if err == nil {
		t.Error("missing required file must return error")
	}
}

func TestValidatePatternStructure_EmptyFile(t *testing.T) {
	dir := makeValidPatternDir(t)
	// Overwrite one file with empty content
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	err := validatePatternStructure(dir, "https://example.com/registry", "v1.0")
	if err == nil {
		t.Error("empty required file must return error")
	}
}

func TestValidatePatternStructure_AllMissing(t *testing.T) {
	dir := t.TempDir() // empty dir
	err := validatePatternStructure(dir, "https://example.com/registry", "v1.0")
	if err == nil {
		t.Error("all files missing must return error")
	}
}
