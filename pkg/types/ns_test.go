// Tests for AllowedNamespaces and RestrictedNamespaces (ns_allowed.go, ns_restricted.go).
package types_test

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
)

// ── AllowedNamespaces.IsAllowed ───────────────────────────────────────────────

func TestAllowedNamespaces_EmptyAllowsAll(t *testing.T) {
	var a orktypes.AllowedNamespaces
	assert.True(t, a.IsAllowed("anything"))
	assert.True(t, a.IsAllowed("kube-system"))
	assert.True(t, a.IsAllowed(""))
}

func TestAllowedNamespaces_ExactMatch(t *testing.T) {
	a := orktypes.AllowedNamespaces{"apps", "workloads"}
	assert.True(t, a.IsAllowed("apps"))
	assert.True(t, a.IsAllowed("workloads"))
	assert.False(t, a.IsAllowed("kube-system"))
	assert.False(t, a.IsAllowed("app")) // partial does not match
}

func TestAllowedNamespaces_PrefixWildcard(t *testing.T) {
	a := orktypes.AllowedNamespaces{"team-*"}
	assert.True(t, a.IsAllowed("team-alpha"))
	assert.True(t, a.IsAllowed("team-beta"))
	assert.False(t, a.IsAllowed("myteam-alpha"))
	assert.False(t, a.IsAllowed("team"))
}

func TestAllowedNamespaces_SuffixWildcard(t *testing.T) {
	a := orktypes.AllowedNamespaces{"*-sandbox"}
	assert.True(t, a.IsAllowed("dev-sandbox"))
	assert.True(t, a.IsAllowed("prod-sandbox"))
	assert.False(t, a.IsAllowed("sandbox-dev"))
	assert.False(t, a.IsAllowed("sandbox"))
}

func TestAllowedNamespaces_MultiplePatterns(t *testing.T) {
	a := orktypes.AllowedNamespaces{"apps", "team-*", "*-sandbox"}
	assert.True(t, a.IsAllowed("apps"))
	assert.True(t, a.IsAllowed("team-payments"))
	assert.True(t, a.IsAllowed("dev-sandbox"))
	assert.False(t, a.IsAllowed("kube-system"))
}

// ── AllowedNamespaces.Merge ───────────────────────────────────────────────────

func TestAllowedNamespaces_Merge_Deduplicates(t *testing.T) {
	a := orktypes.AllowedNamespaces{"apps", "workloads"}
	b := orktypes.AllowedNamespaces{"workloads", "team-*"}
	merged := a.Merge(b)
	assert.Len(t, merged, 3)
	assert.Contains(t, []string(merged), "apps")
	assert.Contains(t, []string(merged), "workloads")
	assert.Contains(t, []string(merged), "team-*")
}

func TestAllowedNamespaces_Merge_EmptyRight(t *testing.T) {
	a := orktypes.AllowedNamespaces{"apps"}
	merged := a.Merge(nil)
	assert.Equal(t, orktypes.AllowedNamespaces{"apps"}, merged)
}

func TestAllowedNamespaces_Merge_EmptyLeft(t *testing.T) {
	var a orktypes.AllowedNamespaces
	b := orktypes.AllowedNamespaces{"apps"}
	merged := a.Merge(b)
	assert.Equal(t, orktypes.AllowedNamespaces{"apps"}, merged)
}

// ── RestrictedNamespaces.IsRestricted ────────────────────────────────────────

func TestRestrictedNamespaces_EmptyAllowsAll(t *testing.T) {
	var r orktypes.RestrictedNamespaces
	assert.False(t, r.IsRestricted("kube-system"))
	assert.False(t, r.IsRestricted("anything"))
}

func TestRestrictedNamespaces_ExactMatch(t *testing.T) {
	r := orktypes.RestrictedNamespaces{"kube-system", "cert-manager"}
	assert.True(t, r.IsRestricted("kube-system"))
	assert.True(t, r.IsRestricted("cert-manager"))
	assert.False(t, r.IsRestricted("apps"))
}

func TestRestrictedNamespaces_PrefixWildcard(t *testing.T) {
	r := orktypes.RestrictedNamespaces{"kube-*"}
	assert.True(t, r.IsRestricted("kube-system"))
	assert.True(t, r.IsRestricted("kube-public"))
	assert.False(t, r.IsRestricted("notakube"))
}

func TestRestrictedNamespaces_SuffixWildcard(t *testing.T) {
	r := orktypes.RestrictedNamespaces{"*-system"}
	assert.True(t, r.IsRestricted("kube-system"))
	assert.True(t, r.IsRestricted("monitoring-system"))
	assert.False(t, r.IsRestricted("system-kube"))
}

// ── RestrictedNamespaces.Merge ────────────────────────────────────────────────

func TestRestrictedNamespaces_Merge_Deduplicates(t *testing.T) {
	r := orktypes.RestrictedNamespaces{"kube-system", "monitoring"}
	s := orktypes.RestrictedNamespaces{"monitoring", "cert-manager"}
	merged := r.Merge(s)
	assert.Len(t, merged, 3)
	assert.Contains(t, []string(merged), "kube-system")
	assert.Contains(t, []string(merged), "monitoring")
	assert.Contains(t, []string(merged), "cert-manager")
}

func TestRestrictedNamespaces_Merge_EmptyRight(t *testing.T) {
	r := orktypes.RestrictedNamespaces{"kube-system"}
	merged := r.Merge(nil)
	assert.Equal(t, orktypes.RestrictedNamespaces{"kube-system"}, merged)
}
