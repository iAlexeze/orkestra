package types

import "testing"

func findRule(t *testing.T, rules []ValidationRule, field string) ValidationRule {
	t.Helper()
	for _, r := range rules {
		if r.Field == field {
			return r
		}
	}
	t.Fatalf("no rule found for field %q in %+v", field, rules)
	return ValidationRule{}
}

func TestRequiredIDPFieldRules_NilCases(t *testing.T) {
	t.Run("nil CRDEntry", func(t *testing.T) {
		var c *CRDEntry
		if got := c.RequiredIDPFieldRules(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("nil IDP", func(t *testing.T) {
		c := &CRDEntry{}
		if got := c.RequiredIDPFieldRules(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
	t.Run("IDP with no required fields", func(t *testing.T) {
		c := &CRDEntry{IDP: &IDPConfig{
			Fields: map[string]IDPFieldConfig{
				"image": {Label: "Container Image"},
			},
		}}
		if got := c.RequiredIDPFieldRules(); got != nil {
			t.Errorf("got %v, want nil — nothing is required", got)
		}
	})
}

func TestRequiredIDPFieldRules_SpecField(t *testing.T) {
	c := &CRDEntry{IDP: &IDPConfig{
		Fields: map[string]IDPFieldConfig{
			"team":  {Label: "Team", Required: true},
			"image": {Label: "Container Image"}, // not required — no rule
		},
	}}

	rules := c.RequiredIDPFieldRules()
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1: %+v", len(rules), rules)
	}

	r := rules[0]
	if r.Field != "spec.team" {
		t.Errorf("Field = %q, want %q", r.Field, "spec.team")
	}
	if r.Operator != ConditionExists {
		t.Errorf("Operator = %q, want %q", r.Operator, ConditionExists)
	}
	if r.Message != "Team is required" {
		t.Errorf("Message = %q, want %q", r.Message, "Team is required")
	}
	if r.Action != ValidationActionDeny {
		t.Errorf("Action = %q, want %q", r.Action, ValidationActionDeny)
	}
}

func TestRequiredIDPFieldRules_InheritsWhenAndAnyOf(t *testing.T) {
	appOnly := []Condition{{Field: "spec.workloadType", Equals: "app"}}
	certOrApp := []Condition{
		{Field: "spec.workloadType", Equals: "cert"},
		{Field: "spec.workloadType", Equals: "app"},
	}

	c := &CRDEntry{IDP: &IDPConfig{
		Fields: map[string]IDPFieldConfig{
			// A discriminator-routed field: only relevant (and only required)
			// when workloadType: app — the same When already used to hide it
			// from the form for other workload types.
			"repoURL": {Label: "Repository URL", Required: true, When: appOnly},
			// AnyOf carries through the same way.
			"domain": {Label: "Domain", Required: true, AnyOf: certOrApp},
			// A universal field with no condition at all.
			"team": {Label: "Team", Required: true},
		},
	}}

	rules := c.RequiredIDPFieldRules()
	if len(rules) != 3 {
		t.Fatalf("got %d rules, want 3: %+v", len(rules), rules)
	}

	repoRule := findRule(t, rules, "spec.repoURL")
	if len(repoRule.When) != 1 || repoRule.When[0].Field != "spec.workloadType" || repoRule.When[0].Equals != "app" {
		t.Errorf("repoURL rule.When = %+v, want the field's own When condition carried through", repoRule.When)
	}
	if len(repoRule.AnyOf) != 0 {
		t.Errorf("repoURL rule.AnyOf = %+v, want empty — the field declared When, not AnyOf", repoRule.AnyOf)
	}

	domainRule := findRule(t, rules, "spec.domain")
	if len(domainRule.AnyOf) != 2 {
		t.Errorf("domain rule.AnyOf = %+v, want the field's own AnyOf conditions carried through", domainRule.AnyOf)
	}

	teamRule := findRule(t, rules, "spec.team")
	if len(teamRule.When) != 0 || len(teamRule.AnyOf) != 0 {
		t.Errorf("team rule When/AnyOf = %+v/%+v, want both empty — the field declared neither", teamRule.When, teamRule.AnyOf)
	}
}

func TestRequiredIDPFieldRules_LabelFallsBackToFieldName(t *testing.T) {
	c := &CRDEntry{IDP: &IDPConfig{
		Fields: map[string]IDPFieldConfig{
			"team": {Required: true}, // no Label set
		},
	}}

	rules := c.RequiredIDPFieldRules()
	if len(rules) != 1 || rules[0].Message != "team is required" {
		t.Errorf("got %+v, want message to fall back to the raw field name", rules)
	}
}

func TestRequiredIDPFieldRules_AdditionalFieldsUseNotes(t *testing.T) {
	c := &CRDEntry{IDP: &IDPConfig{
		AdditionalFields: &AdditionalIDPFields{
			Labels: map[string]IDPFieldConfig{
				"team": {Label: "Team", Required: true},
			},
			Annotations: map[string]IDPFieldConfig{
				"platform.myorg.io/jira-ticket": {Label: "Jira Ticket", Required: true},
			},
		},
	}}

	rules := c.RequiredIDPFieldRules()
	if len(rules) != 2 {
		t.Fatalf("got %d rules, want 2: %+v", len(rules), rules)
	}

	labelRule := findRule(t, rules, `{{ getLabel . "team" }}`)
	if labelRule.Message != "Team is required" || labelRule.Operator != ConditionExists {
		t.Errorf("label rule = %+v, want a getLabel-based exists rule", labelRule)
	}

	// A dotted annotation key must go through getAnnotation, not a raw
	// dot-path — dot-path resolution would misparse the dots inside the key
	// itself as extra path segments.
	annotationField := `{{ getAnnotation . "platform.myorg.io/jira-ticket" }}`
	annotationRule := findRule(t, rules, annotationField)
	if annotationRule.Message != "Jira Ticket is required" || annotationRule.Operator != ConditionExists {
		t.Errorf("annotation rule = %+v, want a getAnnotation-based exists rule", annotationRule)
	}
}

func TestRequiredIDPFieldRules_AllThreeBucketsCombine(t *testing.T) {
	c := &CRDEntry{IDP: &IDPConfig{
		Fields: map[string]IDPFieldConfig{
			"image": {Label: "Container Image", Required: true},
		},
		AdditionalFields: &AdditionalIDPFields{
			Labels: map[string]IDPFieldConfig{
				"team": {Label: "Team", Required: true},
			},
			Annotations: map[string]IDPFieldConfig{
				"expose": {Label: "Expose externally", Required: true},
			},
		},
	}}

	rules := c.RequiredIDPFieldRules()
	if len(rules) != 3 {
		t.Fatalf("got %d rules, want 3 (one per bucket): %+v", len(rules), rules)
	}
	findRule(t, rules, "spec.image")
	findRule(t, rules, `{{ getLabel . "team" }}`)
	findRule(t, rules, `{{ getAnnotation . "expose" }}`)
}
