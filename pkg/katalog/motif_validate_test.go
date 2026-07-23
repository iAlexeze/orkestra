package katalog_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
)

const validMotifPath = "../registry/motif/testdata/valid.yaml"
const simpleMotifPath = "../registry/motif/testdata/simple.yaml"

func TestValidateMotif_Valid(t *testing.T) {
	errs := katalog.ValidateMotif(validMotifPath)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid motif, got: %v", errs)
	}
}

func TestValidateMotif_NotFound(t *testing.T) {
	errs := katalog.ValidateMotif("testdata/does-not-exist.yaml")
	if len(errs) == 0 {
		t.Fatal("expected errors for missing file, got none")
	}
}

func TestValidateMotif_WrongKind(t *testing.T) {
	errs := katalog.ValidateMotif("../registry/motif/testdata/wrong_kind.yaml")
	if len(errs) == 0 {
		t.Fatal("expected errors for wrong kind, got none")
	}
}

func TestValidateMotif_NoName(t *testing.T) {
	errs := katalog.ValidateMotif("../registry/motif/testdata/no_name.yaml")
	if len(errs) == 0 {
		t.Fatal("expected errors for missing name, got none")
	}
}

func TestValidateMotif_Simple(t *testing.T) {
	errs := katalog.ValidateMotif(simpleMotifPath)
	if len(errs) != 0 {
		t.Errorf("expected no errors for simple motif, got: %v", errs)
	}
}
