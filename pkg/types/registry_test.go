// Tests for RegistryRef and RegistrySource methods (registry.go).
package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

// ── RegistryRef.Ref ───────────────────────────────────────────────────────────

func TestRegistryRef_Ref_SHA_Priority(t *testing.T) {
	r := orktypes.RegistryRef{SHA: "abc123", Version: "v1.0.0", Branch: "main"}
	assert.Equal(t, "abc123", r.Ref())
}

func TestRegistryRef_Ref_Version_Priority(t *testing.T) {
	r := orktypes.RegistryRef{Version: "v1.0.0", Branch: "main"}
	assert.Equal(t, "v1.0.0", r.Ref())
}

func TestRegistryRef_Ref_Branch(t *testing.T) {
	r := orktypes.RegistryRef{Branch: "develop"}
	assert.Equal(t, "develop", r.Ref())
}

func TestRegistryRef_Ref_Default(t *testing.T) {
	r := orktypes.RegistryRef{}
	assert.Equal(t, "main", r.Ref())
}

// ── RegistryRef.IsDefault ─────────────────────────────────────────────────────

func TestRegistryRef_IsDefault_Empty(t *testing.T) {
	r := orktypes.RegistryRef{}
	assert.True(t, r.IsDefault())
}

func TestRegistryRef_IsDefault_WithBranch(t *testing.T) {
	r := orktypes.RegistryRef{Branch: "main"}
	assert.False(t, r.IsDefault())
}

func TestRegistryRef_IsDefault_WithSHA(t *testing.T) {
	r := orktypes.RegistryRef{SHA: "abc123"}
	assert.False(t, r.IsDefault())
}

// ── RegistrySource.ResolvedURL ────────────────────────────────────────────────

func TestRegistrySource_ResolvedURL_AtShorthand(t *testing.T) {
	s := orktypes.RegistrySource{URL: "ghcr.io/orkspace/orkestra-registry/postgres@v14"}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "ghcr.io/orkspace/orkestra-registry/postgres", clean)
	assert.Equal(t, "v14", ver)
}

func TestRegistrySource_ResolvedURL_ExplicitVersion(t *testing.T) {
	s := orktypes.RegistrySource{URL: "ghcr.io/orkspace/postgres", Version: "v14.2.0", OCI: true}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "ghcr.io/orkspace/postgres", clean)
	assert.Equal(t, "v14.2.0", ver)
}

func TestRegistrySource_ResolvedURL_DefaultGit(t *testing.T) {
	s := orktypes.RegistrySource{URL: "https://github.com/myorg/registry", OCI: false}
	_, ver := s.ResolvedURL()
	assert.Equal(t, "main", ver)
}

func TestRegistrySource_ResolvedURL_DefaultOCI(t *testing.T) {
	s := orktypes.RegistrySource{URL: "ghcr.io/myorg/postgres", OCI: true}
	_, ver := s.ResolvedURL()
	assert.Equal(t, "latest", ver)
}

func TestRegistrySource_ResolvedURL_AtOverridesVersion(t *testing.T) {
	// @ shorthand wins even when Version is also set
	s := orktypes.RegistrySource{URL: "ghcr.io/myorg/postgres@v12", Version: "v14", OCI: true}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "ghcr.io/myorg/postgres", clean)
	assert.Equal(t, "v12", ver)
}

func TestRegistrySource_ResolvedURL_GitWithAt(t *testing.T) {
	s := orktypes.RegistrySource{URL: "https://github.com/myorg/registry@main"}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "https://github.com/myorg/registry", clean)
	assert.Equal(t, "main", ver)
}

func TestRegistrySource_ResolvedURL_OCIColonTag(t *testing.T) {
	// ork pull tells users to write oci://host/repo:tag — this must not produce host/repo:tag:latest
	s := orktypes.RegistrySource{URL: "oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0"}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "ghcr.io/myorg/katalogs/webapp-operator", clean)
	assert.Equal(t, "v1.0.0", ver)
}

func TestRegistrySource_ResolvedURL_OCIColonTagNoScheme(t *testing.T) {
	s := orktypes.RegistrySource{URL: "ghcr.io/myorg/katalogs/postgres:v14", OCI: true}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "ghcr.io/myorg/katalogs/postgres", clean)
	assert.Equal(t, "v14", ver)
}

func TestRegistrySource_ResolvedURL_OCIColonTagDoesNotSplitOnPort(t *testing.T) {
	// localhost:5000/repo:v1.0.0 — port should not be split; tag after last / should
	s := orktypes.RegistrySource{URL: "oci://localhost:5000/myorg/webapp:v1.0.0"}
	clean, ver := s.ResolvedURL()
	assert.Equal(t, "localhost:5000/myorg/webapp", clean)
	assert.Equal(t, "v1.0.0", ver)
}

// ── RegistrySource.SourceFile ─────────────────────────────────────────────────

func TestRegistrySource_SourceFile_DefaultKatalog(t *testing.T) {
	s := orktypes.RegistrySource{}
	assert.Equal(t, "katalog.yaml", s.SourceFile())
}

func TestRegistrySource_SourceFile_Komposer(t *testing.T) {
	s := orktypes.RegistrySource{UseKomposer: true}
	assert.Equal(t, "komposer.yaml", s.SourceFile())
}
