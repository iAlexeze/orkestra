package katalog

import (
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func katalogWithIDP(idp *orktypes.IDPConfig) *Katalog {
	return &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"myresource": {IDP: idp},
		},
	}
}

func TestValidateIDPAdditionalFields_NilIDP(t *testing.T) {
	k := katalogWithIDP(nil)
	if err := k.validateIDPAdditionalFields(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPAdditionalFields_NoAdditionalFields(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{Enabled: true})
	if err := k.validateIDPAdditionalFields(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestValidateIDPAdditionalFields_TypeOmittedDefaultsToValid guards against a
// regression where an omitted Type (which defaults to "string" per
// IDPFieldConfig's doc comment) was incorrectly rejected as an invalid type.
// Deliberately uses a single field with no collision and no key-syntax issue
// so nothing else can make this pass for the wrong reason.
func TestValidateIDPAdditionalFields_TypeOmittedDefaultsToValid(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"tier": {Label: "Tier"}, // Type intentionally omitted
			},
		},
	})
	if err := k.validateIDPAdditionalFields(); err != nil {
		t.Fatalf("omitted Type should default to valid (string), got error: %v", err)
	}
}

func TestValidateIDPAdditionalFields_InvalidType(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"tier": {Label: "Tier", Type: "sting"}, // typo
			},
		},
	})
	err := k.validateIDPAdditionalFields()
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "sting") {
		t.Errorf("error should mention the bad type, got: %v", err)
	}
}

func TestValidateIDPAdditionalFields_Valid(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		Fields: map[string]orktypes.IDPFieldConfig{
			"image": {Label: "Image"},
		},
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"tier": {Label: "Tier", Type: "enum", Enum: []string{"free", "pro"}},
			},
			Annotations: map[string]orktypes.IDPFieldConfig{
				"platform.example.io/monitoring": {Label: "Monitoring", Type: "boolean"},
			},
		},
	})
	if err := k.validateIDPAdditionalFields(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIDPAdditionalFields_InvalidLabelKey(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"Not Valid Key!": {Label: "Bad"},
			},
		},
	})
	err := k.validateIDPAdditionalFields()
	if err == nil {
		t.Fatal("expected error for invalid label key")
	}
	if !strings.Contains(err.Error(), "Not Valid Key!") {
		t.Errorf("error should mention the bad key, got: %v", err)
	}
}

func TestValidateIDPAdditionalFields_InvalidAnnotationKey(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Annotations: map[string]orktypes.IDPFieldConfig{
				"has spaces": {Label: "Bad"},
			},
		},
	})
	if err := k.validateIDPAdditionalFields(); err == nil {
		t.Fatal("expected error for invalid annotation key")
	}
}

func TestValidateIDPAdditionalFields_EnumEmpty(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"tier": {Label: "Tier", Type: "enum"},
			},
		},
	})
	err := k.validateIDPAdditionalFields()
	if err == nil {
		t.Fatal("expected error for empty enum")
	}
	if !strings.Contains(err.Error(), "tier") {
		t.Errorf("error should mention the field name, got: %v", err)
	}
}

func TestValidateIDPAdditionalFields_CollisionWithFields(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		Fields: map[string]orktypes.IDPFieldConfig{
			"image": {Label: "Image"},
		},
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"image": {Label: "Image dup"},
			},
		},
	})
	err := k.validateIDPAdditionalFields()
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "image") {
		t.Errorf("error should mention the colliding key, got: %v", err)
	}
}

func TestValidateIDPAdditionalFields_CollisionBetweenLabelsAndAnnotations(t *testing.T) {
	k := katalogWithIDP(&orktypes.IDPConfig{
		Enabled: true,
		AdditionalFields: &orktypes.AdditionalIDPFields{
			Labels: map[string]orktypes.IDPFieldConfig{
				"tier": {Label: "Tier (label)"},
			},
			Annotations: map[string]orktypes.IDPFieldConfig{
				"tier": {Label: "Tier (annotation)"},
			},
		},
	})
	if err := k.validateIDPAdditionalFields(); err == nil {
		t.Fatal("expected collision error between labels and annotations")
	}
}

func TestValidateIDPAdditionalFields_MultipleCRDs(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"good": {IDP: &orktypes.IDPConfig{
				Enabled: true,
				AdditionalFields: &orktypes.AdditionalIDPFields{
					Labels: map[string]orktypes.IDPFieldConfig{"tier": {}},
				},
			}},
			"bad": {IDP: &orktypes.IDPConfig{
				Enabled: true,
				AdditionalFields: &orktypes.AdditionalIDPFields{
					Labels: map[string]orktypes.IDPFieldConfig{"Not Valid!": {}},
				},
			}},
		},
	}
	if err := k.validateIDPAdditionalFields(); err == nil {
		t.Fatal("expected error from the bad CRD entry")
	}
}
