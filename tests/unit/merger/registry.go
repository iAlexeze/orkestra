// pkg/merger/registry_test.go
package merger_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ialexeze/orkestra/pkg/merger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
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

// ── URL construction ──────────────────────────────────────────────────────────

func TestGitHubRawURL(t *testing.T) {
	tests := []struct {
		repoURL  string
		ref      string
		filePath string
		expected string
	}{
		{
			"https://github.com/konduktor-io/orkestra-registry",
			"main",
			"registry/katalogs/website/katalog.yaml",
			"https://raw.githubusercontent.com/konduktor-io/orkestra-registry/main/registry/katalogs/website/katalog.yaml",
		},
		{
			// With .git suffix — should be stripped
			"https://github.com/myorg/my-registry.git",
			"v1.2.0",
			"registry/katalogs/platform/katalog.yaml",
			"https://raw.githubusercontent.com/myorg/my-registry/v1.2.0/registry/katalogs/platform/katalog.yaml",
		},
		{
			// With trailing slash
			"https://github.com/myorg/my-registry/",
			"abc123",
			"registry/katalogs/app/katalog.yaml",
			"https://raw.githubusercontent.com/myorg/my-registry/abc123/registry/katalogs/app/katalog.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			got := merger.ExportedGitHubRawURL(tt.repoURL, tt.ref, tt.filePath)
			if got != tt.expected {
				t.Errorf("\nexpected: %q\n     got: %q", tt.expected, got)
			}
		})
	}
}

func TestGitLabRawURL(t *testing.T) {
	tests := []struct {
		repoURL  string
		ref      string
		filePath string
		expected string
	}{
		{
			"https://gitlab.com/myorg/orkestra-registry",
			"main",
			"registry/katalogs/website/katalog.yaml",
			"https://gitlab.com/myorg/orkestra-registry/-/raw/main/registry/katalogs/website/katalog.yaml",
		},
		{
			"https://gitlab.com/myorg/registry.git",
			"v2.0.0",
			"registry/katalogs/db/katalog.yaml",
			"https://gitlab.com/myorg/registry/-/raw/v2.0.0/registry/katalogs/db/katalog.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			got := merger.ExportedGitLabRawURL(tt.repoURL, tt.ref, tt.filePath)
			if got != tt.expected {
				t.Errorf("\nexpected: %q\n     got: %q", tt.expected, got)
			}
		})
	}
}

// ── loadRegistrySource via HTTP test server ────────────────────────────────────

var websiteKatalogYAML = []byte(`
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-katalog
spec:
  crds:
    - name: website
      enabled: true
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      reconciler:
        default: true
`)

var platformKatalogYAML = []byte(`
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: platform-namespace-katalog
spec:
  crds:
    - name: platformnamespace
      enabled: true
      apiTypes:
        group: platform.orkestra.io
        version: v1alpha1
        kind: PlatformNamespace
        plural: platformnamespaces
      reconciler:
        default: true
`)

// registryTestServer returns an httptest.Server that serves registry files.
// It simulates the raw file serving that GitHub/GitLab would provide.
func registryTestServer(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
}

func TestLoadRegistrySource_SingleKatalog(t *testing.T) {
	// Serve the website katalog at the expected path
	srv := registryTestServer(t, map[string][]byte{
		"/main/registry/katalogs/website/katalog.yaml": websiteKatalogYAML,
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
	if crds[0].Name != "website" {
		t.Errorf("expected CRD name=website, got %q", crds[0].Name)
	}
}

func TestLoadRegistrySource_MultipleKatalogs(t *testing.T) {
	srv := registryTestServer(t, map[string][]byte{
		"/main/registry/katalogs/website/katalog.yaml":           websiteKatalogYAML,
		"/main/registry/katalogs/platformnamespace/katalog.yaml": platformKatalogYAML,
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website":           {Branch: "main"},
			"platformnamespace": {Branch: "main"},
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(crds) != 2 {
		t.Errorf("expected 2 CRDs, got %d: %v", len(crds), crds)
	}
}

func TestLoadRegistrySource_VersionRef(t *testing.T) {
	// Version tag as ref — same URL structure, different ref segment
	srv := registryTestServer(t, map[string][]byte{
		"/v1.2.0/registry/katalogs/website/katalog.yaml": websiteKatalogYAML,
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Version: "v1.2.0"},
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
}

func TestLoadRegistrySource_SHARef(t *testing.T) {
	srv := registryTestServer(t, map[string][]byte{
		"/abc123def456/registry/katalogs/website/katalog.yaml": websiteKatalogYAML,
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {SHA: "abc123def456"},
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
}

func TestLoadRegistrySource_DefaultRef_UsesMain(t *testing.T) {
	// Empty RegistryRef — should default to "main"
	srv := registryTestServer(t, map[string][]byte{
		"/main/registry/katalogs/website/katalog.yaml": websiteKatalogYAML,
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {}, // empty — defaults to main
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
}

func TestLoadRegistrySource_KatalogNotFound_Error(t *testing.T) {
	srv := registryTestServer(t, map[string][]byte{
		// website katalog missing
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	_, err := merger.ExportedLoadRegistrySource(m, src)
	if err == nil {
		t.Error("expected error for missing katalog")
	}
}

func TestLoadRegistrySource_NoRegistryURL_Error(t *testing.T) {
	// No registry URL anywhere — should produce clear error
	os.Unsetenv("ORK_REGISTRY")

	m := merger.New()
	// No SetRegistryURL called

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	_, err := merger.ExportedLoadRegistrySource(m, src)
	if err == nil {
		t.Error("expected error when no registry URL is configured")
	}
	if !containsStr(err.Error(), "ORK_REGISTRY") {
		t.Errorf("error should mention ORK_REGISTRY, got: %q", err.Error())
	}
}

func TestLoadRegistrySource_ExplicitURLOverridesEnv(t *testing.T) {
	// Explicit URL in source block should override ORK_REGISTRY
	srv := registryTestServer(t, map[string][]byte{
		"/main/registry/katalogs/website/katalog.yaml": websiteKatalogYAML,
	})
	defer srv.Close()

	// Set ORK_REGISTRY to a different (non-serving) URL
	os.Setenv("ORK_REGISTRY", "https://wrong-registry.example.com")
	defer os.Unsetenv("ORK_REGISTRY")

	m := merger.New()

	src := orktypes.RegistrySource{
		URL: srv.URL, // explicit override
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("explicit URL should override ORK_REGISTRY: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
	}
}

func TestLoadRegistrySource_ORKRegistryEnvVar(t *testing.T) {
	srv := registryTestServer(t, map[string][]byte{
		"/main/registry/katalogs/website/katalog.yaml": websiteKatalogYAML,
	})
	defer srv.Close()

	os.Setenv("ORK_REGISTRY", srv.URL)
	defer os.Unsetenv("ORK_REGISTRY")

	m := merger.New()
	// No SetRegistryURL — should pick up from env

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	crds, err := merger.ExportedLoadRegistrySource(m, src)
	if err != nil {
		t.Fatalf("should pick up ORK_REGISTRY from env: %v", err)
	}
	if len(crds) != 1 {
		t.Errorf("expected 1 CRD, got %d", len(crds))
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

func TestLoadRegistrySource_InvalidKatalogYAML_Error(t *testing.T) {
	srv := registryTestServer(t, map[string][]byte{
		"/main/registry/katalogs/website/katalog.yaml": []byte("this: is: not: valid: yaml: {{{{"),
	})
	defer srv.Close()

	m := merger.New()
	m.SetRegistryURL(srv.URL)

	src := orktypes.RegistrySource{
		Katalog: map[string]orktypes.RegistryRef{
			"website": {Branch: "main"},
		},
	}

	_, err := merger.ExportedLoadRegistrySource(m, src)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// ── URL detection ─────────────────────────────────────────────────────────────

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/myorg/myrepo", true},
		{"https://github.com/myorg/myrepo.git", false}, // .git suffix → not detected as GitHub raw
		{"https://gitlab.com/myorg/myrepo", false},
		{"https://raw.githubusercontent.com/myorg/repo", false},
		{"https://bitbucket.org/myorg/myrepo", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := merger.ExportedIsGitHubURL(tt.url)
			if got != tt.want {
				t.Errorf("isGitHubURL(%q): expected %v, got %v", tt.url, tt.want, got)
			}
		})
	}
}

func TestIsGitLabURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://gitlab.com/myorg/myrepo", true},
		{"https://gitlab.com/myorg/myrepo.git", false},
		{"https://github.com/myorg/myrepo", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := merger.ExportedIsGitLabURL(tt.url)
			if got != tt.want {
				t.Errorf("isGitLabURL(%q): expected %v, got %v", tt.url, tt.want, got)
			}
		})
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || strings.Contains(s, sub))
}
