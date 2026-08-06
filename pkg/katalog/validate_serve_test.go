package katalog

import (
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func katalogWithServe(serve *orktypes.ServeConfig) *Katalog {
	return &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"myresource": {Serve: serve},
		},
	}
}

func TestValidateServeAdditionalFields_NilServe(t *testing.T) {
	k := katalogWithServe(nil)
	if err := k.validateServeAdditionalFields(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateServeAdditionalFields_NoAdditionalFields(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{Enabled: true})
	if err := k.validateServeAdditionalFields(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestValidateServeAdditionalFields_TypeOmittedDefaultsToValid guards against a
// regression where an omitted Type (which defaults to "string" per
// ServeFieldConfig's doc comment) was incorrectly rejected as an invalid type.
// Deliberately uses a single field with no collision and no key-syntax issue
// so nothing else can make this pass for the wrong reason.
func TestValidateServeAdditionalFields_TypeOmittedDefaultsToValid(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Labels: map[string]orktypes.ServeFieldConfig{
			"tier": {Label: "Tier"}, // Type intentionally omitted
		},
	})
	if err := k.validateServeAdditionalFields(); err != nil {
		t.Fatalf("omitted Type should default to valid (string), got error: %v", err)
	}
}

func TestValidateServeAdditionalFields_InvalidType(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Labels: map[string]orktypes.ServeFieldConfig{
			"tier": {Label: "Tier", Type: "sting"}, // typo
		},
	})
	err := k.validateServeAdditionalFields()
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "sting") {
		t.Errorf("error should mention the bad type, got: %v", err)
	}
}

func TestValidateServeAdditionalFields_Valid(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Fields: map[string]orktypes.ServeFieldConfig{
			"image": {Label: "Image"},
		},
		Labels: map[string]orktypes.ServeFieldConfig{
			"tier": {Label: "Tier", Type: "enum", Enum: []string{"free", "pro"}},
		},
		Annotations: map[string]orktypes.ServeFieldConfig{
			"platform.example.io/monitoring": {Label: "Monitoring", Type: "boolean"},
		},
	})
	if err := k.validateServeAdditionalFields(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateServeAdditionalFields_InvalidLabelKey(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Labels: map[string]orktypes.ServeFieldConfig{
			"Not Valid Key!": {Label: "Bad"},
		},
	})
	err := k.validateServeAdditionalFields()
	if err == nil {
		t.Fatal("expected error for invalid label key")
	}
	if !strings.Contains(err.Error(), "Not Valid Key!") {
		t.Errorf("error should mention the bad key, got: %v", err)
	}
}

func TestValidateServeAdditionalFields_InvalidAnnotationKey(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Annotations: map[string]orktypes.ServeFieldConfig{
			"has spaces": {Label: "Bad"},
		},
	})
	if err := k.validateServeAdditionalFields(); err == nil {
		t.Fatal("expected error for invalid annotation key")
	}
}

func TestValidateServeAdditionalFields_EnumEmpty(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Labels: map[string]orktypes.ServeFieldConfig{
			"tier": {Label: "Tier", Type: "enum"},
		},
	})
	err := k.validateServeAdditionalFields()
	if err == nil {
		t.Fatal("expected error for empty enum")
	}
	if !strings.Contains(err.Error(), "tier") {
		t.Errorf("error should mention the field name, got: %v", err)
	}
}

func TestValidateServeAdditionalFields_CollisionWithFields(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Fields: map[string]orktypes.ServeFieldConfig{
			"image": {Label: "Image"},
		},
		Labels: map[string]orktypes.ServeFieldConfig{
			"image": {Label: "Image dup"},
		},
	})
	err := k.validateServeAdditionalFields()
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "image") {
		t.Errorf("error should mention the colliding key, got: %v", err)
	}
}

func TestValidateServeAdditionalFields_CollisionBetweenLabelsAndAnnotations(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Enabled: true,
		Labels: map[string]orktypes.ServeFieldConfig{
			"tier": {Label: "Tier (label)"},
		},
		Annotations: map[string]orktypes.ServeFieldConfig{
			"tier": {Label: "Tier (annotation)"},
		},
	})
	if err := k.validateServeAdditionalFields(); err == nil {
		t.Fatal("expected collision error between labels and annotations")
	}
}

func TestValidateServeAdditionalFields_MultipleCRDs(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"good": {Serve: &orktypes.ServeConfig{
				Enabled: true,
				Labels:  map[string]orktypes.ServeFieldConfig{"tier": {}},
			}},
			"bad": {Serve: &orktypes.ServeConfig{
				Enabled: true,
				Labels:  map[string]orktypes.ServeFieldConfig{"Not Valid!": {}},
			}},
		},
	}
	if err := k.validateServeAdditionalFields(); err == nil {
		t.Fatal("expected error from the bad CRD entry")
	}
}

func TestValidateServeFieldOrder_NoCollision(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Fields: map[string]orktypes.ServeFieldConfig{
			"image":    {Order: 1},
			"replicas": {Order: 2},
		},
	})
	if err := k.validateServeFieldOrder(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateServeFieldOrder_UnsetNeverCollides(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Fields: map[string]orktypes.ServeFieldConfig{
			"image":    {},
			"replicas": {},
		},
	})
	if err := k.validateServeFieldOrder(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateServeFieldOrder_Collision(t *testing.T) {
	k := katalogWithServe(&orktypes.ServeConfig{
		Fields: map[string]orktypes.ServeFieldConfig{
			"image": {Order: 3},
		},
		Labels: map[string]orktypes.ServeFieldConfig{
			"team": {Order: 3},
		},
	})
	err := k.validateServeFieldOrder()
	if err == nil {
		t.Fatal("expected a collision error")
	}
	if !strings.Contains(err.Error(), "image") || !strings.Contains(err.Error(), "team") {
		t.Errorf("error = %q, want it to name both colliding fields", err.Error())
	}
}
