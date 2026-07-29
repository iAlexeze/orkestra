package e2e

import "testing"

func TestApplyAssertions_OutputContains_Pass(t *testing.T) {
	if err := applyAssertions("hello world", assertions{OutputContains: "world"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_OutputContains_Fail(t *testing.T) {
	if err := applyAssertions("hello world", assertions{OutputContains: "missing"}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_OutputNotContains_Pass(t *testing.T) {
	if err := applyAssertions("hello world", assertions{OutputNotContains: "missing"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_OutputNotContains_Fail(t *testing.T) {
	if err := applyAssertions("hello world", assertions{OutputNotContains: "world"}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_Equals_TrimsOutput(t *testing.T) {
	if err := applyAssertions("  exact  \n", assertions{Equals: "exact"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_NotEquals_Pass(t *testing.T) {
	if err := applyAssertions("actual", assertions{NotEquals: "expected"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_Exists_EmptyFails(t *testing.T) {
	if err := applyAssertions("", assertions{Exists: true}); err == nil {
		t.Error("expected error for empty output with exists: true")
	}
}

func TestApplyAssertions_NotExists_NonEmptyFails(t *testing.T) {
	if err := applyAssertions("something", assertions{NotExists: true}); err == nil {
		t.Error("expected error for non-empty output with notExists: true")
	}
}

func TestApplyAssertions_OneOf_Match(t *testing.T) {
	if err := applyAssertions("b", assertions{OneOf: []string{"a", "b", "c"}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_OneOf_NoMatch(t *testing.T) {
	if err := applyAssertions("z", assertions{OneOf: []string{"a", "b", "c"}}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_NotOneOf_Pass(t *testing.T) {
	if err := applyAssertions("z", assertions{NotOneOf: []string{"a", "b", "c"}}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_NotOneOf_Fail(t *testing.T) {
	if err := applyAssertions("b", assertions{NotOneOf: []string{"a", "b", "c"}}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_GreaterThan_Strict(t *testing.T) {
	if err := applyAssertions("5", assertions{GreaterThan: "5"}); err == nil {
		t.Error("gt is strict — equal value should fail")
	}
	if err := applyAssertions("6", assertions{GreaterThan: "5"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_LessThan_Strict(t *testing.T) {
	if err := applyAssertions("5", assertions{LessThan: "5"}); err == nil {
		t.Error("lt is strict — equal value should fail")
	}
	if err := applyAssertions("4", assertions{LessThan: "5"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_GreaterThan_NonNumericOutput(t *testing.T) {
	if err := applyAssertions("not-a-number", assertions{GreaterThan: "5"}); err == nil {
		t.Error("expected error for non-numeric output")
	}
}

func TestApplyAssertions_GreaterThanOrEqual_Inclusive(t *testing.T) {
	if err := applyAssertions("5", assertions{GreaterThanOrEqual: "5"}); err != nil {
		t.Errorf("gte is inclusive — equal value should pass: %v", err)
	}
	if err := applyAssertions("4", assertions{GreaterThanOrEqual: "5"}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_LessThanOrEqual_Inclusive(t *testing.T) {
	if err := applyAssertions("5", assertions{LessThanOrEqual: "5"}); err != nil {
		t.Errorf("lte is inclusive — equal value should pass: %v", err)
	}
	if err := applyAssertions("6", assertions{LessThanOrEqual: "5"}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_Between_InclusiveBounds(t *testing.T) {
	for _, v := range []string{"1", "5", "10"} {
		if err := applyAssertions(v, assertions{Between: "1,10"}); err != nil {
			t.Errorf("value %q should be within [1,10]: %v", v, err)
		}
	}
	if err := applyAssertions("11", assertions{Between: "1,10"}); err == nil {
		t.Error("expected error for value outside range")
	}
}

func TestApplyAssertions_NotBetween(t *testing.T) {
	if err := applyAssertions("11", assertions{NotBetween: "1,10"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := applyAssertions("5", assertions{NotBetween: "1,10"}); err == nil {
		t.Error("expected error for value inside excluded range")
	}
}

func TestApplyAssertions_Regex_Match(t *testing.T) {
	if err := applyAssertions("app-prod-01", assertions{Regex: `^app-\w+-\d+$`}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyAssertions_Regex_NoMatch(t *testing.T) {
	if err := applyAssertions("APP", assertions{Regex: `^app-\w+-\d+$`}); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplyAssertions_MultipleAssertions_AllApply(t *testing.T) {
	// outputContains passes, greaterThan fails — the whole call must fail,
	// same "every set assertion applies" semantics as before the refactor.
	err := applyAssertions("5", assertions{OutputContains: "5", GreaterThan: "10"})
	if err == nil {
		t.Error("expected error when one of several combined assertions fails")
	}
}

func TestApplyAssertions_NoFieldsSet_Passes(t *testing.T) {
	if err := applyAssertions("anything", assertions{}); err != nil {
		t.Errorf("unexpected error with no assertions set: %v", err)
	}
}
