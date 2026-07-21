package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestApplyAPIConfigYAML(t *testing.T) {
	input := `
enabled: true
applyAPI:
  enabled: true
  auth:
    tokens:
      - name: ci-pipeline
        secretRef:
          name: ork-apply-token
          key: token
          rotateAfter: 90d
      - name: local-dev
        token: "${ORK_DEV_TOKEN}"
`
	var gw GatewayConfig
	if err := yaml.Unmarshal([]byte(input), &gw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if gw.ApplyAPI == nil {
		t.Fatal("ApplyAPI is nil")
	}
	if !gw.ApplyAPI.Enabled {
		t.Error("ApplyAPI.Enabled should be true")
	}
	tokens := gw.ApplyAPI.Auth.Tokens
	if len(tokens) != 2 {
		t.Fatalf("want 2 tokens, got %d", len(tokens))
	}

	sr := tokens[0]
	if sr.Name != "ci-pipeline" {
		t.Errorf("token[0].Name = %q", sr.Name)
	}
	if sr.SecretRef == nil {
		t.Fatal("token[0].SecretRef is nil")
	}
	if sr.SecretRef.Name != "ork-apply-token" {
		t.Errorf("secretRef.Name = %q", sr.SecretRef.Name)
	}
	if sr.SecretRef.Key != "token" {
		t.Errorf("secretRef.Key = %q", sr.SecretRef.Key)
	}
	if sr.SecretRef.RotateAfter != "90d" {
		t.Errorf("secretRef.RotateAfter = %q", sr.SecretRef.RotateAfter)
	}

	ev := tokens[1]
	if ev.Token != "${ORK_DEV_TOKEN}" {
		t.Errorf("token[1].Token = %q", ev.Token)
	}
}

func TestIDPConfigYAML(t *testing.T) {
	input := `
idp:
  enabled: true
  fields:
    workloadType:
      label: "Workload Type"
      hint: "app or cert"
      order: 1
    team:
      label: "Team"
      placeholder: "team-payments"
      order: 2
`
	var entry CRDEntry
	if err := yaml.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.IDP == nil {
		t.Fatal("IDP is nil")
	}
	if !entry.IDP.Enabled {
		t.Error("IDP.Enabled should be true")
	}
	wt, ok := entry.IDP.Fields["workloadType"]
	if !ok {
		t.Fatal("missing workloadType field")
	}
	if wt.Label != "Workload Type" {
		t.Errorf("label = %q", wt.Label)
	}
	if wt.Order != 1 {
		t.Errorf("order = %d", wt.Order)
	}
	team := entry.IDP.Fields["team"]
	if team.Placeholder != "team-payments" {
		t.Errorf("placeholder = %q", team.Placeholder)
	}
}

func TestIDPConfigIgnoreFieldsAndWhen(t *testing.T) {
	input := `
idp:
  enabled: true
  ignoreFields:
    - internalId
    - createdBy
  category: "Compute"
  description: "Self-service application deployment"
  fields:
    workloadType:
      label: "Workload Type"
      order: 1
      category: "Basic"
    repoURL:
      label: "Repository URL"
      order: 2
      when:
        - field: workloadType
          equals: app
    certIssuer:
      label: "Issuer"
      order: 2
      when:
        - field: workloadType
          equals: cert
        - field: workloadType
          notEquals: app
`
	var entry CRDEntry
	if err := yaml.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	idp := entry.IDP
	if idp == nil {
		t.Fatal("IDP is nil")
	}
	if len(idp.IgnoreFields) != 2 || idp.IgnoreFields[0] != "internalId" {
		t.Errorf("IgnoreFields = %v", idp.IgnoreFields)
	}
	if idp.Category != "Compute" {
		t.Errorf("Category = %q", idp.Category)
	}
	if idp.Description != "Self-service application deployment" {
		t.Errorf("Description = %q", idp.Description)
	}

	wt := idp.Fields["workloadType"]
	if wt.Category != "Basic" {
		t.Errorf("workloadType.Category = %q", wt.Category)
	}
	if len(wt.When) != 0 {
		t.Errorf("workloadType should have no when conditions")
	}

	repo := idp.Fields["repoURL"]
	if len(repo.When) != 1 {
		t.Fatalf("repoURL.When len = %d, want 1", len(repo.When))
	}
	if repo.When[0].Field != "workloadType" || repo.When[0].Equals != "app" {
		t.Errorf("repoURL.When[0] = %+v", repo.When[0])
	}

	cert := idp.Fields["certIssuer"]
	if len(cert.When) != 2 {
		t.Fatalf("certIssuer.When len = %d, want 2", len(cert.When))
	}
	if cert.When[1].NotEquals != "app" {
		t.Errorf("certIssuer.When[1].NotEquals = %q", cert.When[1].NotEquals)
	}

	// time-based when uses the same Condition infrastructure
	inputWithTime := `
idp:
  enabled: true
  fields:
    environment:
      label: "Environment"
      order: 1
    prodDeploy:
      label: "Production Deploy"
      when:
        - time:
            after: "08:00"
            before: "18:00"
          dayOfWeek:
            weekday: true
`
	var entryTime CRDEntry
	if err := yaml.Unmarshal([]byte(inputWithTime), &entryTime); err != nil {
		t.Fatalf("unmarshal time-when: %v", err)
	}
	pd := entryTime.IDP.Fields["prodDeploy"]
	if len(pd.When) != 1 {
		t.Fatalf("prodDeploy.When len = %d, want 1", len(pd.When))
	}
	if pd.When[0].Time == nil || pd.When[0].Time.After != "08:00" {
		t.Errorf("prodDeploy.When[0].Time = %+v", pd.When[0].Time)
	}
	if pd.When[0].DayOfWeek == nil || pd.When[0].DayOfWeek.Weekday == nil {
		t.Errorf("prodDeploy.When[0].DayOfWeek = %+v", pd.When[0].DayOfWeek)
	}
}

func TestIDPFieldAnyOfAndDisabled(t *testing.T) {
	input := `
idp:
  enabled: true
  fields:
    environment:
      label: "Environment"
      order: 1
    prodDeploy:
      label: "Production Deploy"
      anyOf:
        - time:
            after: "08:00"
            before: "18:00"
        - dayOfWeek:
            weekday: true
    legacyFeature:
      label: "Legacy Feature"
      disabled: "Deprecated — use NewFeature instead"
`
	var entry CRDEntry
	if err := yaml.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pd := entry.IDP.Fields["prodDeploy"]
	if len(pd.AnyOf) != 2 {
		t.Fatalf("prodDeploy.AnyOf len = %d, want 2", len(pd.AnyOf))
	}
	if pd.AnyOf[0].Time == nil || pd.AnyOf[0].Time.After != "08:00" {
		t.Errorf("prodDeploy.AnyOf[0].Time = %+v", pd.AnyOf[0].Time)
	}
	if pd.AnyOf[1].DayOfWeek == nil || pd.AnyOf[1].DayOfWeek.Weekday == nil {
		t.Errorf("prodDeploy.AnyOf[1].DayOfWeek = %+v", pd.AnyOf[1].DayOfWeek)
	}

	lf := entry.IDP.Fields["legacyFeature"]
	if lf.Disabled != "Deprecated — use NewFeature instead" {
		t.Errorf("legacyFeature.Disabled = %q", lf.Disabled)
	}
}

func TestApplyAPITokenValidation(t *testing.T) {
	cases := []struct {
		name    string
		token   ApplyAPIToken
		wantSR  bool
		wantEnv bool
	}{
		{"secretRef", ApplyAPIToken{Name: "x", SecretRef: &ApplyAPISecretRef{Name: "s", Key: "k"}}, true, false},
		{"envvar", ApplyAPIToken{Name: "x", Token: "${MY_TOKEN}"}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if (tc.token.SecretRef != nil) != tc.wantSR {
				t.Errorf("SecretRef presence mismatch")
			}
			if (tc.token.Token != "") != tc.wantEnv {
				t.Errorf("Token presence mismatch")
			}
		})
	}
}
