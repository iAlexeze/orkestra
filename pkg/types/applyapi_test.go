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
