// pkg/merger/registry_test.go
package merger_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orkspace/orkestra/pkg/merger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ── RegistryRef ───────────────────────────────────────────────────────────────

func TestRegistryRef_Ref_SHA_Priority(t *testing.T) {
	ref := orktypes.RegistryRef{Branch: "main", Version: "v1.0.0", SHA: "abc123"}
	if ref.Ref() != "abc123" {
		t.Errorf("SHA should take priority, got %q", ref.Ref())
	}
}

func TestRegistryRef_Ref_Version_Priority(t *testing.T) {
	ref := orktypes.RegistryRef{Branch: "main", Version: "v1.0.0"}
	if ref.Ref() != "v1.0.0" {
		t.Errorf("Version should take priority over Branch, got %q", ref.Ref())
	}
}

func TestRegistryRef_Ref_Branch(t *testing.T) {
	ref := orktypes.RegistryRef{Branch: "develop"}
	if ref.Ref() != "develop" {
		t.Errorf("expected develop, got %q", ref.Ref())
	}
}

func TestRegistryRef_Ref_Default(t *testing.T) {
	ref := orktypes.RegistryRef{}
	if ref.Ref() != "main" {
		t.Errorf("default ref should be main, got %q", ref.Ref())
	}
}

func TestRegistryRef_IsDefault(t *testing.T) {
	if !(orktypes.RegistryRef{}).IsDefault() {
		t.Error("empty ref should report IsDefault true")
	}
	if (orktypes.RegistryRef{Branch: "main"}).IsDefault() {
		t.Error("ref with branch set should not be default")
	}
	if (orktypes.RegistryRef{SHA: "abc"}).IsDefault() {
		t.Error("ref with SHA set should not be default")
	}
}

// ── RegistrySource.ResolvedURL ────────────────────────────────────────────────

func TestResolvedURL_AtShorthand(t *testing.T) {
	src := orktypes.RegistrySource{URL: "ghcr.io/orkspace/orkestra-registry/postgres@v14"}
	u, version := src.ResolvedURL()
	if u != "ghcr.io/orkspace/orkestra-registry/postgres" {
		t.Errorf("url: expected stripped @, got %q", u)
	}
	if version != "v14" {
		t.Errorf("version: expected v14, got %q", version)
	}
}

func TestResolvedURL_AtShorthand_GitURL(t *testing.T) {
	src := orktypes.RegistrySource{URL: "https://github.com/myorg/registry@main"}
	u, version := src.ResolvedURL()
	if u != "https://github.com/myorg/registry" {
		t.Errorf("url: expected %q, got %q", "https://github.com/myorg/registry", u)
	}
	if version != "main" {
		t.Errorf("version: expected main, got %q", version)
	}
}

func TestResolvedURL_ExplicitVersion(t *testing.T) {
	src := orktypes.RegistrySource{URL: "ghcr.io/orkspace/orkestra-registry/postgres", Version: "v14.2.0"}
	u, version := src.ResolvedURL()
	if u != "ghcr.io/orkspace/orkestra-registry/postgres" {
		t.Errorf("url: expected unchanged, got %q", u)
	}
	if version != "v14.2.0" {
		t.Errorf("version: expected v14.2.0, got %q", version)
	}
}

func TestResolvedURL_AtShorthand_Takes_Priority(t *testing.T) {
	src := orktypes.RegistrySource{URL: "ghcr.io/orkspace/orkestra-registry/postgres@v14", Version: "v15"}
	_, version := src.ResolvedURL()
	if version != "v14" {
		t.Errorf("@ shorthand should take priority, got version %q", version)
	}
}

func TestResolvedURL_DefaultVersion_OCI(t *testing.T) {
	src := orktypes.RegistrySource{URL: "ghcr.io/orkspace/orkestra-registry/postgres", OCI: true}
	_, version := src.ResolvedURL()
	if version != "latest" {
		t.Errorf("OCI default version should be 'latest', got %q", version)
	}
}

func TestResolvedURL_DefaultVersion_Git(t *testing.T) {
	src := orktypes.RegistrySource{URL: "https://github.com/myorg/registry", OCI: false}
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

// ── validatePatternStructure ──────────────────────────────────────────────────

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
			continue
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
			content = []byte{}
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
	os.WriteFile(filepath.Join(dir, "crd.yaml"), []byte("content"), 0644)

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "main")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, f := range []string{"katalog.yaml", "komposer.yaml", "cr.yaml", "README.md"} {
		if !strings.Contains(err.Error(), f) {
			t.Errorf("error should mention missing %q: %q", f, err)
		}
	}
}

// ── Deprecated catalog-map registry protocol ──────────────────────────────────

func TestLoadRegistrySource_NoRegistryURL_Error(t *testing.T) {
	os.Unsetenv("ORK_REGISTRY")

	m := merger.New()

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	_, err := merger.ExportedLoadRegistrySource(m, src)
	if err == nil {
		t.Error("expected error when no registry URL is configured")
	}
	if !strings.Contains(err.Error(), "ORK_REGISTRY") {
		t.Errorf("error should mention ORK_REGISTRY, got: %q", err.Error())
	}
}

func TestLoadRegistrySource_EmptyKatalogMap_NoError(t *testing.T) {
	m := merger.New()
	m.SetRegistryURL("https://github.com/example/registry")

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("empty katalog map should not error: %v", err)
	}
	if len(crds) != 0 {
		t.Errorf("expected 0 CRDs, got %d", len(crds))
	}
}

// ── GitHub raw URL construction ───────────────────────────────────────────────

func TestGitHubRawURL_Construction(t *testing.T) {
	got := merger.ExportedGitHubRawURL("https://github.com/myorg/myrepo", "main", "katalog.yaml")
	want := "https://raw.githubusercontent.com/myorg/myrepo/main/katalog.yaml"
	if got != want {
		t.Errorf("githubRawURL: got %q, want %q", got, want)
	}
}

// patternServer serves the 5 required pattern files at /<version>/<filename>.
func patternServer(t *testing.T, version string, overrides map[string][]byte) *httptest.Server {
	t.Helper()
	files := map[string][]byte{
		"/" + version + "/crd.yaml":      []byte("kind: CustomResourceDefinition"),
		"/" + version + "/katalog.yaml":  testKatalogYAML,
		"/" + version + "/komposer.yaml": []byte("apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: c\nspec:\n  crds:\n    - name: myapp\n      enabled: true\n      apiTypes:\n        group: test.orkestra.io\n        version: v1alpha1\n        kind: MyApp\n        plural: myapps\n"),
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

var testKatalogYAML = []byte(`apiVersion: orkestra.orkspace.io/v1
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
      operatorBox:
        default: true
`)

// NOTE: Tests that require network access (git clone, OCI pull, real GitHub/GitLab)
// belong in tests/integration/komposer/ behind the integration build tag.
