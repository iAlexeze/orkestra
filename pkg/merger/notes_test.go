// pkg/merger/notes_test.go
package merger

import (
	"strings"
	"testing"
)

const katalogWithNotesYAML = `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: noted
notes:
  functions:
    - name: host
      expression: "{{ .metadata.name }}.cluster.local"
    - name: img
      expression: "{{ .spec.image }}:latest"
spec:
  crds:
    widget:
      apiTypes:
        kind: Widget
        group: example.io
`

// TestKatalog_InlineNotes_ForwardedToToNotes verifies that notes: declared
// directly in a Katalog are returned by ToNotes() after Merge().
func TestKatalog_InlineNotes_ForwardedToToNotes(t *testing.T) {
	dir := t.TempDir()
	path := writeTempKatalog(t, dir, "katalog.yaml", katalogWithNotesYAML)

	m := New(path)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	notes := m.ToNotes()
	if len(notes.Functions) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes.Functions))
	}
	names := map[string]bool{}
	for _, n := range notes.Functions {
		names[n.Name] = true
	}
	if !names["host"] {
		t.Error("expected note 'host' from Katalog")
	}
	if !names["img"] {
		t.Error("expected note 'img' from Katalog")
	}
}

// TestKomposer_InlineNotes_ForwardedToToNotes verifies that notes: declared on
// a Komposer are visible via ToNotes() after the Katalog files are imported.
func TestKomposer_InlineNotes_ForwardedToToNotes(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "katalog.yaml", katalogWithNotesYAML)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: k\nnotes:\n  functions:\n    - name: env\n      expression: \"prod\"\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	notes := m.ToNotes()
	names := map[string]bool{}
	for _, n := range notes.Functions {
		names[n.Name] = true
	}
	if !names["env"] {
		t.Error("expected Komposer inline note 'env'")
	}
	// Notes from the imported Katalog must also be present.
	if !names["host"] {
		t.Error("expected Katalog note 'host' to pass through")
	}
}

// TestKomposer_InlineNotes_OverrideKatalogNote verifies that when a Komposer
// declares a note with the same name as one in an imported Katalog, the
// Komposer's expression is the one returned (last-wins via FuncMap ordering).
func TestKomposer_InlineNotes_OverrideKatalogNote(t *testing.T) {
	dir := t.TempDir()
	katalogPath := writeTempKatalog(t, dir, "katalog.yaml", katalogWithNotesYAML)

	// host is also declared in the Katalog; Komposer's value must win (appended last).
	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: k\nnotes:\n  functions:\n    - name: host\n      expression: \"{{ .metadata.name }}.prod.example.com\"\nimports:\n  files:\n    - url: " + katalogPath + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	if err := m.Merge(); err != nil {
		t.Fatalf("Merge() error: %v", err)
	}

	notes := m.ToNotes()
	// The Komposer note is appended after Katalog notes, so it appears last in the registry.
	// Confirm the last entry for 'host' comes from the Komposer.
	var lastHost string
	for _, n := range notes.Functions {
		if n.Name == "host" {
			lastHost = n.Expression
		}
	}
	if !strings.Contains(lastHost, "prod.example.com") {
		t.Errorf("expected Komposer's host expression to win, got %q", lastHost)
	}
}

// TestKomposer_CrossKatalogNoteConflict_Errors verifies that two imported Katalogs
// declaring the same note name returns an error. Unlike the Komposer's own notes:
// block (which intentionally overrides), cross-Katalog conflicts are ambiguous and
// must be surfaced rather than silently resolved by import order.
func TestKomposer_CrossKatalogNoteConflict_Errors(t *testing.T) {
	dir := t.TempDir()

	src1 := writeTempKatalog(t, dir, "src1.yaml", `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: src1
notes:
  functions:
    - name: host
      expression: "{{ .metadata.name }}.src1.local"
spec:
  crds:
    alpha:
      apiTypes:
        kind: Alpha
        group: example.io
`)
	src2 := writeTempKatalog(t, dir, "src2.yaml", `apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: src2
notes:
  functions:
    - name: host
      expression: "{{ .metadata.name }}.src2.local"
spec:
  crds:
    beta:
      apiTypes:
        kind: Beta
        group: example.io
`)

	komposer := "apiVersion: orkestra.orkspace.io/v1\nkind: Komposer\nmetadata:\n  name: k\nimports:\n  files:\n    - url: " + src1 + "\n    - url: " + src2 + "\n"
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	err := m.Merge()
	if err == nil {
		t.Fatal("expected conflict error for note 'host' declared in two Katalogs, got nil")
	}
	if !strings.Contains(err.Error(), "host") {
		t.Errorf("expected error to mention conflicting note name 'host', got: %v", err)
	}
}

// TestKomposer_SpecImports_Rejected verifies that a Komposer with spec.imports
// is rejected with an error. spec.imports is reserved for Katalogs only.
func TestKomposer_SpecImports_Rejected(t *testing.T) {
	dir := t.TempDir()

	// Komposer has both spec.crds (passes the empty-komposer guard) and spec.imports.
	komposer := `apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: bad-komposer
spec:
  imports:
    - motif: ./some-motif.yaml
  crds:
    widget:
      apiTypes:
        kind: Widget
        group: example.io
`
	komposerPath := writeTempKatalog(t, dir, "komposer.yaml", komposer)

	m := New(komposerPath)
	err := m.Merge()
	if err == nil {
		t.Fatal("expected error for Komposer with spec.imports, got nil")
	}
	if !strings.Contains(err.Error(), "spec.imports") {
		t.Errorf("expected error to mention spec.imports, got: %v", err)
	}
}
