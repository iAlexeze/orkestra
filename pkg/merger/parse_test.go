// pkg/merger/parse_test.go
package merger

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/konfig"
)

// ── containsValidKind ─────────────────────────────────────────────────────────

func TestContainsValidKind_Katalog(t *testing.T) {
	doc := []byte("kind: " + konfig.KatalogKind() + "\napiVersion: orkestra.orkspace.io/v1\n")
	if !containsValidKind(doc) {
		t.Error("document with Katalog kind must return true")
	}
}

func TestContainsValidKind_Komposer(t *testing.T) {
	doc := []byte("kind: " + konfig.KomposerKind() + "\napiVersion: orkestra.orkspace.io/v1\n")
	if !containsValidKind(doc) {
		t.Error("document with Komposer kind must return true")
	}
}

func TestContainsValidKind_OtherKind(t *testing.T) {
	doc := []byte("kind: Deployment\napiVersion: apps/v1\n")
	if containsValidKind(doc) {
		t.Error("Deployment kind must return false")
	}
}

func TestContainsValidKind_Empty(t *testing.T) {
	if containsValidKind([]byte{}) {
		t.Error("empty document must return false")
	}
}

func TestContainsValidKind_KindInComment(t *testing.T) {
	// The word "Katalog" appears in a comment — fast check will match.
	// This is known behaviour: the full parse handles it, not containsValidKind.
	doc := []byte("# kind: Katalog\napiVersion: apps/v1\nkind: Deployment\n")
	// We don't assert here — the fast check can produce false positives.
	// The test documents the known limitation.
	_ = containsValidKind(doc)
}

// ── parseKatalogDoc ───────────────────────────────────────────────────────────

func validKatalogDoc() []byte {
	return []byte(`apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: test-katalog
`)
}

func TestParseKatalogDoc_Valid(t *testing.T) {
	kf, err := parseKatalogDoc(validKatalogDoc(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kf == nil {
		t.Fatal("expected non-nil KatalogFile")
	}
	if kf.Metadata.Name != "test-katalog" {
		t.Errorf("expected name=test-katalog, got %q", kf.Metadata.Name)
	}
	if kf.Kind != konfig.KatalogKind() {
		t.Errorf("expected kind=%s, got %q", konfig.KatalogKind(), kf.Kind)
	}
}

func TestParseKatalogDoc_NonKatalogReturnsNil(t *testing.T) {
	doc := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: foo\n")
	kf, err := parseKatalogDoc(doc, "test")
	if err != nil {
		t.Fatalf("non-katalog doc must not error: %v", err)
	}
	if kf != nil {
		t.Error("non-katalog doc must return nil KatalogFile")
	}
}

func TestParseKatalogDoc_MissingName_Error(t *testing.T) {
	doc := []byte("apiVersion: orkestra.orkspace.io/v1\nkind: Katalog\nmetadata:\n  name: \"\"\n")
	_, err := parseKatalogDoc(doc, "test")
	if err == nil {
		t.Error("missing metadata.name must return error")
	}
}

func TestParseKatalogDoc_UnsupportedApiVersion_Error(t *testing.T) {
	doc := []byte("apiVersion: example.io/v999\nkind: Katalog\nmetadata:\n  name: foo\n")
	_, err := parseKatalogDoc(doc, "test")
	if err == nil {
		t.Error("unsupported apiVersion must return error")
	}
}

func TestParseKatalogDoc_ValidKomposer(t *testing.T) {
	doc := []byte(`apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: my-komposer
`)
	kf, err := parseKatalogDoc(doc, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kf == nil || kf.Kind != konfig.KomposerKind() {
		t.Errorf("expected Komposer kind, got %v", kf)
	}
}

func TestParseKatalogDoc_EmptyDoc(t *testing.T) {
	kf, err := parseKatalogDoc([]byte{}, "empty")
	if err != nil {
		t.Fatalf("empty doc must not error: %v", err)
	}
	if kf != nil {
		t.Error("empty doc must return nil")
	}
}

func TestParseKatalogDoc_UnknownField_Error(t *testing.T) {
	doc := []byte(`apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: test
unknownTopLevelField: boom
`)
	_, err := parseKatalogDoc(doc, "test")
	if err == nil {
		t.Error("unknown field in strict parse must return error")
	}
}
