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
	// @ in URL takes priority over explicit Version field
	src := orktypes.RegistrySource{
		URL:     "ghcr.io/konduktor-io/orkestra-registry/postgres@v14",
		Version: "v15", // should be ignored
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
		// no version
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
		// no version
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
		t.Errorf("default SourceFile should be katalog.yaml, got %q", src.SourceFile())
	}
}

func TestSourceFile_UseKomposer_IsKomposer(t *testing.T) {
	src := orktypes.RegistrySource{
		URL:         "https://github.com/myorg/r",
		UseKomposer: true,
	}
	if src.SourceFile() != "komposer.yaml" {
		t.Errorf("useKomposer:true should return komposer.yaml, got %q", src.SourceFile())
	}
}

// ── validatePatternStructure ──────────────────────────────────────────────────

func TestValidatePatternStructure_AllFiles_Passes(t *testing.T) {
	dir := t.TempDir()

	// Write all five required files with content
	for _, f := range orktypes.RequiredFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644); err != nil {
			t.Fatalf("writing %s: %v", f, err)
		}
	}

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "v1")
	if err != nil {
		t.Errorf("all files present and non-empty — expected no error, got: %v", err)
	}
}

func TestValidatePatternStructure_MissingFile_Fails(t *testing.T) {
	dir := t.TempDir()

	// Write all except README.md
	for _, f := range orktypes.RequiredFiles {
		if f == "README.md" {
			continue
		}
		os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644)
	}

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "v1")
	if err == nil {
		t.Error("missing README.md — expected error")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Errorf("error should mention README.md: %q", err)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error should say 'missing': %q", err)
	}
}

func TestValidatePatternStructure_EmptyFile_Fails(t *testing.T) {
	dir := t.TempDir()

	for _, f := range orktypes.RequiredFiles {
		content := []byte("content")
		if f == "cr.yaml" {
			content = []byte{} // empty
		}
		os.WriteFile(filepath.Join(dir, f), content, 0644)
	}

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "v1")
	if err == nil {
		t.Error("empty cr.yaml — expected error")
	}
	if !strings.Contains(err.Error(), "cr.yaml") {
		t.Errorf("error should mention cr.yaml: %q", err)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should say 'empty': %q", err)
	}
}

func TestValidatePatternStructure_MultipleViolations_AllReported(t *testing.T) {
	dir := t.TempDir()

	// Only write katalog.yaml — everything else missing
	os.WriteFile(filepath.Join(dir, "katalog.yaml"), []byte("content"), 0644)

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "v1")
	if err == nil {
		t.Fatal("expected error for multiple missing files")
	}

	// All 4 missing files should be mentioned
	for _, f := range []string{"crd.yaml", "komposer.yaml", "cr.yaml", "README.md"} {
		if !strings.Contains(err.Error(), f) {
			t.Errorf("error should mention %q: %v", f, err)
		}
	}
}

func TestValidatePatternStructure_IncludesGuidance(t *testing.T) {
	// Error message should include the list of required files so users
	// know what to add without having to look at documentation
	dir := t.TempDir()
	// empty dir — all files missing

	err := merger.ExportedValidatePatternStructure(dir, "test-url", "v1")
	if err == nil {
		t.Fatal("expected error")
	}

	for _, required := range []string{
		"crd.yaml", "katalog.yaml", "komposer.yaml", "cr.yaml", "README.md",
	} {
		if !strings.Contains(err.Error(), required) {
			t.Errorf("error guidance should mention required file %q", required)
		}
	}
}

// ── loadRegistrySource via HTTP test server ────────────────────────────────────

var catalogYAML = []byte(`
apiVersion: orkestra.konductor.io/v1Alpha
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

var komposerYAML = []byte(`
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: test-komposer
spec:
  crds:
    - name: myapp
      enabled: true
      apiTypes:
        group: test.orkestra.io
        version: v1alpha1
        kind: MyApp
        plural: myapps
`)

// completePatternServer serves all 5 required files for a GitHub-like raw endpoint.
func completePatternServer(t *testing.T, overrides map[string][]byte) *httptest.Server {
	t.Helper()
	files := map[string][]byte{
		"/main/crd.yaml":      []byte("kind: CustomResourceDefinition"),
		"/main/katalog.yaml":  catalogYAML,
		"/main/komposer.yaml": komposerYAML,
		"/main/cr.yaml":       []byte("kind: MyApp"),
		"/main/README.md":     []byte("# MyApp Operator\nDocumentation here."),
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

func TestLoadRegistrySource_Git_LoadsKatalog(t *testing.T) {
	srv := completePatternServer(t, nil)
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		URL:         srv.URL, // will be treated as GitHub-like
		Version:     "main",
		OCI:         false,
		UseKomposer: false, // load katalog.yaml
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
	if crds[0].Name != "myapp" {
		t.Errorf("expected CRD name=myapp, got %q", crds[0].Name)
	}
}

func TestLoadRegistrySource_Git_UseKomposer(t *testing.T) {
	srv := completePatternServer(t, nil)
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		URL:         srv.URL,
		Version:     "main",
		UseKomposer: true, // load komposer.yaml
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error with useKomposer: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD from komposer, got %d", len(crds))
	}
}

func TestLoadRegistrySource_AtShorthand_VersionParsed(t *testing.T) {
	srv := completePatternServer(t, map[string][]byte{
		"/v1.2.0/crd.yaml":      []byte("kind: CustomResourceDefinition"),
		"/v1.2.0/katalog.yaml":  catalogYAML,
		"/v1.2.0/komposer.yaml": komposerYAML,
		"/v1.2.0/cr.yaml":       []byte("kind: MyApp"),
		"/v1.2.0/README.md":     []byte("# MyApp"),
	})
	defer srv.Close()

	m := merger.New()

	src := orktypes.RegistrySource{
		URL: srv.URL + "@v1.2.0", // @ shorthand
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("@ shorthand should parse version correctly: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
}

func TestLoadRegistrySource_MissingFile_FailsFast(t *testing.T) {
	// Serve all files EXCEPT cr.yaml — validation should catch it
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "cr.yaml") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer srv.Close()

	m := merger.New()

	src := orktypes.RegistrySource{
		URL:     srv.URL,
		Version: "main",
	}

	_, err := merger.ExportedLoadRegistrySource(m, src)
	if err == nil {
		t.Error("expected error for missing cr.yaml")
	}
	if !strings.Contains(err.Error(), "cr.yaml") {
		t.Errorf("error should mention cr.yaml: %q", err)
	}
}

func TestLoadRegistrySource_EmptyFile_FailsFast(t *testing.T) {
	srv := completePatternServer(t, map[string][]byte{
		"/main/README.md": {}, // empty README
	})
	defer srv.Close()

	m := merger.New()

	src := orktypes.RegistrySource{
		URL:     srv.URL,
		Version: "main",
	}

	_, err := merger.ExportedLoadRegistrySource(m, src)
	if err == nil {
		t.Error("expected error for empty README.md")
	}
	if !strings.Contains(err.Error(), "README.md") {
		t.Errorf("error should mention README.md: %q", err)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should say 'empty': %q", err)
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
