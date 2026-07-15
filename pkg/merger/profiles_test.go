// pkg/merger/profiles_test.go
package merger

import (
	"strings"
	"testing"
)

const katalogWithProfilesYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: profiled
profiles:
  reconciler:
    - name: fast
      workers: 10
    - name: slow
      workers: 1
spec:
  crds:
    widget:
      apiTypes:
        kind: Widget
        group: example.io
`

// TestKatalog_InlineProfiles_ForwardedToToProfiles verifies that profiles:
// declared directly in a Katalog are returned by ToProfiles() after Merge().
func TestKatalog_InlineProfiles_ForwardedToToProfiles(t *testing.T) {
	dir := t.TempDir()
	path := writeTempKatalog(t, dir, "katalog.yaml", katalogWithProfilesYAML)

	m := New(path)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	profiles := m.ToProfiles()
	if len(profiles.Reconciler) != 2 {
		t.Fatalf("expected 2 reconciler profiles, got %d", len(profiles.Reconciler))
	}
	if _, ok := profiles.LookupReconciler("fast"); !ok {
		t.Error("expected reconciler profile 'fast'")
	}
	if _, ok := profiles.LookupReconciler("slow"); !ok {
		t.Error("expected reconciler profile 'slow'")
	}
}

// TestKomposer_InlineProfiles_MergedWithKatalogProfiles verifies that a Komposer
// can add profiles with names that do not conflict with imported Katalog profiles,
// and that both sets appear in ToProfiles().
func TestKomposer_InlineProfiles_MergedWithKatalogProfiles(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "katalog.yaml", katalogWithProfilesYAML)

	// Komposer declares a profile with a different name — no conflict.
	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: k\nprofiles:\n  reconciler:\n    - name: batch\n      workers: 5\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	profiles := m.ToProfiles()
	if _, ok := profiles.LookupReconciler("fast"); !ok {
		t.Error("expected katalog profile 'fast' to pass through")
	}
	if _, ok := profiles.LookupReconciler("slow"); !ok {
		t.Error("expected katalog profile 'slow' to pass through")
	}
	if _, ok := profiles.LookupReconciler("batch"); !ok {
		t.Error("expected Komposer profile 'batch'")
	}
}

// TestKomposer_InlineProfiles_ConflictErrors verifies that a Komposer declaring
// a profile with the same name as one from an imported Katalog returns an error.
// Unlike notes, profiles use conflict-detection rather than last-wins.
func TestKomposer_InlineProfiles_ConflictErrors(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "katalog.yaml", katalogWithProfilesYAML)

	// 'fast' is also declared in the Katalog — this must error.
	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: k\nprofiles:\n  reconciler:\n    - name: fast\n      workers: 99\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	err := m.Merge()
	if err == nil {
		t.Fatal("expected conflict error for duplicate profile name 'fast', got nil")
	}
	if !strings.Contains(err.Error(), "fast") {
		t.Errorf("expected error to mention the conflicting profile name 'fast', got: %v", err)
	}
}
