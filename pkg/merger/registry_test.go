// pkg/merger/registry_test.go
package merger_test

import (
	"os"
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

// ── loadRegistrySource — pure-logic paths (no network) ───────────────────────

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

// NOTE: TestLoadRegistrySource_* tests that fetch real URLs (GitHub, GitLab,
// or httptest servers) belong in tests/integration/komposer/ behind the
// `integration` build tag. The deprecated registry protocol uses git clone or
// raw HTTP fetch — either way, network access is required.
