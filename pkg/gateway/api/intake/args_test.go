package intake

import "testing"

func TestParseSlackArgs_Valid(t *testing.T) {
	fields, err := ParseSlackArgs("app repository=myorg/payments-api environment=staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["target"] != "app" {
		t.Errorf("target = %v, want app", fields["target"])
	}
	if fields["repository"] != "myorg/payments-api" {
		t.Errorf("repository = %v, want myorg/payments-api", fields["repository"])
	}
	if fields["environment"] != "staging" {
		t.Errorf("environment = %v, want staging", fields["environment"])
	}
}

func TestParseSlackArgs_TargetOnly(t *testing.T) {
	fields, err := ParseSlackArgs("app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 1 || fields["target"] != "app" {
		t.Errorf("fields = %v, want just {target: app}", fields)
	}
}

func TestParseSlackArgs_ValueContainsEquals(t *testing.T) {
	// SplitN with n=2 means only the first "=" splits — a value containing
	// "=" (e.g. a base64 fragment) stays intact.
	fields, err := ParseSlackArgs("app note=a=b=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["note"] != "a=b=c" {
		t.Errorf("note = %v, want a=b=c", fields["note"])
	}
}

func TestParseSlackArgs_Empty(t *testing.T) {
	if _, err := ParseSlackArgs(""); err == nil {
		t.Error("expected an error for empty text")
	}
	if _, err := ParseSlackArgs("   "); err == nil {
		t.Error("expected an error for whitespace-only text")
	}
}

func TestParseSlackArgs_InvalidArgument(t *testing.T) {
	if _, err := ParseSlackArgs("app not-a-key-value-pair"); err == nil {
		t.Error("expected an error for an argument with no '='")
	}
	if _, err := ParseSlackArgs("app =value"); err == nil {
		t.Error("expected an error for an argument with an empty key")
	}
}
