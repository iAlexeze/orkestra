// pkg/merger/registry_v2_test.go
package merger_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ialexeze/orkestra/pkg/merger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ── RegistrySource.ResolvedURL ────────────────────────────────────────────────

func TestResolvedURL_AtShorthand(t *testing.T) {
	src := orktypes.RegistrySource{
		URL: "ghcr.io/konduktor-io/orkestra-registry/postgres@v14",
	}
	url, version := src.ResolvedURL()
	if url != "ghcr.io/konduktor-io/orkestra-registry/postgres" {
		t.Errorf("url: expected stripped @, got %q", url)
	}
	if version != "v14" {
		t.Errorf("version: expected v14, got %q", version)
	}
}

func TestResolvedURL_AtShorthand_GitURL(t *testing.T) {
	src := orktypes.RegistrySource{
		URL: "https://github.com/myorg/registry@main",
	}
	url, version := src.ResolvedURL()
	if url != "https://github.com/myorg/registry" {
		t.Errorf("url: expected %q, got %q", "https://github.com/myorg/registry", url)
	}
	if version != "main" {
		t.Errorf("version: expected main, got %q", version)
	}
}

func TestResolvedURL_ExplicitVersion(t *testing.T) {
	src := orktypes.RegistrySource{
		URL:     "ghcr.io/konduktor-io/orkestra-registry/postgres",
		Version: "v14.2.0",
	}
	url, version := src.ResolvedURL()
	if url != "ghcr.io/konduktor-io/orkestra-registry/postgres" {
		t.Errorf("url: expected unchanged, got %q", url)
	}
	if version != "v14.2.0" {
		t.Errorf("version: expected v14.2.0, got %q", version)
	}
}

func TestResolvedURL_AtShorthand_Takes_Priority(t *testing.T) {
	src := orktypes.RegistrySource{
		URL:     "ghcr.io/konduktor-io/orkestra-registry/postgres@v14",
		Version: "v15",
	}
	_, version := src.ResolvedURL()
	if version != "v14" {
		t.Errorf("@ shorthand should take priority, got version %q", version)
	}
}

func TestResolvedURL_DefaultVersion_OCI(t *testing.T) {
	src := orktypes.RegistrySource{
		URL: "ghcr.io/konduktor-io/orkestra-registry/postgres",
		OCI: true,
	}
	_, version := src.ResolvedURL()
	if version != "latest" {
		t.Errorf("OCI default version should be 'latest', got %q", version)
	}
}

func TestResolvedURL_DefaultVersion_Git(t *testing.T) {
	src := orktypes.RegistrySource{
		URL: "https://github.com/myorg/registry",
		OCI: false,
	}
	_, version := src.ResolvedURL()
	if version != "main" {
		t.Errorf("Git default version should be 'main', got %q", version)
	}
}

// ── RegistrySource.SourceFile ─────────────────────────────────────────────────

func TestSourceFile_Default_IsKatalog(t *testing.T) {
	src := orktypes.RegistrySource{URL: "https://github.com/myorg/r"}
	if src.SourceFile() != "katalog.yaml" {
		t.Errorf("default source file should be katalog.yaml, got %q", src.SourceFile())
	}
}

func TestSourceFile_UseKomposer_IsKomposer(t *testing.T) {
	src := orktypes.RegistrySource{UseKomposer: true}
	if src.SourceFile() != "komposer.yaml" {
		t.Errorf("UseKomposer=true should return komposer.yaml, got %q", src.SourceFile())
	}
}

// ── RequiredFiles ─────────────────────────────────────────────────────────────

func TestRequiredFiles_ContainsAll(t *testing.T) {
	expected := []string{"crd.yaml", "katalog.yaml", "komposer.yaml", "cr.yaml", "README.md"}
	if len(orktypes.RequiredFiles) != len(expected) {
		t.Errorf("expected %d required files, got %d", len(expected), len(orktypes.RequiredFiles))
	}
	for _, f := range expected {
		found := false
		for _, r := range orktypes.RequiredFiles {
			if r == f {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required file %q not in RequiredFiles", f)
		}
	}
}

// ── validatePatternStructure (pure filesystem check) ─────────────────────────

func TestValidatePatternStructure_AllPresent_NoError(t *testing.T) {
	dir := t.TempDir()
	for _, f := range orktypes.RequiredFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644); err != nil {
			t.Fatalf("writing %s: %v", f, err)
		}
	}
	if err := merger.ExportedValidatePatternStructure(dir, "test-url", "main"); err != nil {
		t.Errorf("all files present should pass: %v", err)
	}
}

func TestValidatePatternStructure_MissingFile_Error(t *testing.T) {
	dir := t.TempDir()
	for _, f := range orktypes.RequiredFiles {
		if f == "cr.yaml" {
			continue // leave cr.yaml missing
		}
		os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644)
	}
	err := merger.ExportedValidatePatternStructure(dir, "test-url", "main")
	if err == nil {
		t.Fatal("expected error for missing cr.yaml")
	}
	if !strings.Contains(err.Error(), "cr.yaml") {
		t.Errorf("error should mention cr.yaml: %q", err)
	}
}

func TestValidatePatternStructure_EmptyFile_Error(t *testing.T) {
	dir := t.TempDir()
	for _, f := range orktypes.RequiredFiles {
		content := []byte("content")
		if f == "README.md" {
			content = []byte{} // empty
		}
		os.WriteFile(filepath.Join(dir, f), content, 0644)
	}
	err := merger.ExportedValidatePatternStructure(dir, "test-url", "main")
	if err == nil {
		t.Fatal("expected error for empty README.md")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Errorf("error should mention README.md: %q", err)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should say 'empty': %q", err)
	}
}

func TestValidatePatternStructure_MultipleViolations_AllReported(t *testing.T) {
	dir := t.TempDir()
	// Write only one file — the rest are missing
	os.WriteFile(filepath.Join(dir, "crd.yaml"), []byte("content"), 0644)

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "main")
	if err == nil {
		t.Fatal("expected error")
	}
	// Should mention all four missing files
	for _, f := range []string{"katalog.yaml", "komposer.yaml", "cr.yaml", "README.md"} {
		if !strings.Contains(err.Error(), f) {
			t.Errorf("error should mention missing %q: %q", f, err)
		}
	}
}

// ── GitHub raw URL fetching (load via GitHub raw URL construction) ────────────
// These tests verify that the GitHub-specific raw file fetch path works
// by serving files at raw.githubusercontent.com-like paths on a test server.
// They use ExportedGitHubRawURL to construct the expected request paths.

var v2KatalogYAML = []byte(`apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: test-katalog
spec:
  crds:
    - name: myapp
      enabled: true
      apiTypes:
        group: test.orkestra.io
        version: v1alpha1
        kind: MyApp
        plural: myapps
      reconciler:
        default: true
`)

// v2PatternServer serves the 5 required pattern files at /<version>/<filename>.
// This simulates what the GitHub/GitLab raw URL fetch produces.
func v2PatternServer(t *testing.T, version string, overrides map[string][]byte) *httptest.Server {
	t.Helper()
	files := map[string][]byte{
		"/" + version + "/crd.yaml":      []byte("kind: CustomResourceDefinition"),
		"/" + version + "/katalog.yaml":  v2KatalogYAML,
		"/" + version + "/komposer.yaml": []byte("apiVersion: orkestra.konductor.io/v1Alpha\nkind: Komposer\nmetadata:\n  name: c\nspec:\n  crds:\n    - name: myapp\n      enabled: true\n      apiTypes:\n        group: test.orkestra.io\n        version: v1alpha1\n        kind: MyApp\n        plural: myapps\n"),
		"/" + version + "/cr.yaml":       []byte("kind: MyApp"),
		"/" + version + "/README.md":     []byte("# MyApp"),
	}
	for k, v := range overrides {
		files[k] = v
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
}

// NOTE: Tests that call ExportedLoadRegistrySourceV2 against a real GitHub/GitLab
// URL (or that require git clone / oras) belong in tests/integration/komposer/
// where the integration build tag signals their external dependencies.
// The tests below exercise only the URL-construction and structure-validation
// paths that are safe to run without network access.
