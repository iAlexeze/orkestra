package types

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test fixture %q: %v", path, err)
	}
	return path
}

func TestExpandGatewayWebhookIncludes_Nil(t *testing.T) {
	if err := ExpandGatewayWebhookIncludes(nil, "."); err != nil {
		t.Errorf("nil gw: err = %v, want nil", err)
	}
	if err := ExpandGatewayWebhookIncludes(&GatewayConfig{}, "."); err != nil {
		t.Errorf("nil gw.Webhooks: err = %v, want nil", err)
	}
}

func TestExpandGatewayWebhookIncludes_NoIncludes(t *testing.T) {
	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		GitHub: []GitWebhookConfig{{Name: "orders-repo", Enabled: true, Path: "/webhooks/github/orders"}},
	}}
	if err := ExpandGatewayWebhookIncludes(gw, "."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gw.Webhooks.GitHub) != 1 || gw.Webhooks.GitHub[0].Name != "orders-repo" {
		t.Errorf("GitHub = %+v, want unchanged single entry", gw.Webhooks.GitHub)
	}
}

func TestExpandGatewayWebhookIncludes_PerEntryInclude(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "payments-github.yaml", `
github:
  - name: payments-repo
    enabled: true
    path: /webhooks/github/payments
    branch: main
`)

	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		GitHub: []GitWebhookConfig{
			{Include: "payments-github.yaml"},
			{Name: "orders-repo", Enabled: true, Path: "/webhooks/github/orders"},
		},
	}}

	if err := ExpandGatewayWebhookIncludes(gw, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gw.Webhooks.GitHub) != 2 {
		t.Fatalf("GitHub = %+v, want 2 entries (1 spliced in + 1 inline)", gw.Webhooks.GitHub)
	}
	names := map[string]bool{gw.Webhooks.GitHub[0].Name: true, gw.Webhooks.GitHub[1].Name: true}
	if !names["payments-repo"] || !names["orders-repo"] {
		t.Errorf("GitHub names = %v, want payments-repo and orders-repo", names)
	}
	for _, e := range gw.Webhooks.GitHub {
		if e.Include != "" {
			t.Errorf("entry %q: Include = %q, want cleared after expansion", e.Name, e.Include)
		}
	}
}

func TestExpandGatewayWebhookIncludes_PerEntryInclude_ExpandsToMultiple(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "team-repos.yaml", `
github:
  - name: payments-repo
    enabled: true
    path: /webhooks/github/payments
    branch: main
  - name: orders-repo
    enabled: true
    path: /webhooks/github/orders
    branch: main
`)

	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		GitHub: []GitWebhookConfig{{Include: "team-repos.yaml"}},
	}}

	if err := ExpandGatewayWebhookIncludes(gw, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gw.Webhooks.GitHub) != 2 {
		t.Fatalf("GitHub = %+v, want 2 entries spliced in from one include", gw.Webhooks.GitHub)
	}
}

func TestExpandGatewayWebhookIncludes_PerEntryInclude_WrongKeyErrors(t *testing.T) {
	// A github entry's include file has a "gitlab:" key instead of
	// "github:" — StrictUnmarshal rejects it as an unrecognized field
	// rather than silently yielding zero entries, same as
	// ExpandExternalCalls would for a "calls:" file with an unrelated key.
	dir := t.TempDir()
	writeTestFile(t, dir, "gitlab-only.yaml", `
gitlab:
  - name: orders-repo
    enabled: true
    path: /webhooks/gitlab/orders
    branch: main
`)

	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		GitHub: []GitWebhookConfig{{Include: "gitlab-only.yaml"}},
	}}
	if err := ExpandGatewayWebhookIncludes(gw, dir); err == nil {
		t.Fatal("expected an error — included file's only key is gitlab:, not github:")
	}
}

func TestExpandGatewayWebhookIncludes_PerEntryInclude_EmptyFileIsEmpty(t *testing.T) {
	// An include file with no top-level "github:" key at all (nothing to
	// conflict with) expands to zero entries, not an error.
	dir := t.TempDir()
	writeTestFile(t, dir, "empty.yaml", `{}`)

	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		GitHub: []GitWebhookConfig{{Include: "empty.yaml"}},
	}}
	if err := ExpandGatewayWebhookIncludes(gw, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gw.Webhooks.GitHub) != 0 {
		t.Errorf("GitHub = %+v, want empty — included file declared no github: key", gw.Webhooks.GitHub)
	}
}

func TestExpandGatewayWebhookIncludes_PerEntryInclude_MissingFile(t *testing.T) {
	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		GitHub: []GitWebhookConfig{{Include: "does-not-exist.yaml"}},
	}}
	if err := ExpandGatewayWebhookIncludes(gw, t.TempDir()); err == nil {
		t.Fatal("expected an error for a missing include file")
	}
}

func TestExpandGatewayWebhookIncludes_BlockLevelInclude_MergeByName(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "shared-webhooks.yaml", `
webhooks:
  github:
    - name: payments-repo
      enabled: true
      path: /webhooks/github/payments
      branch: main
  slack:
    - name: platform-workspace
      enabled: true
      path: /webhooks/slack
`)

	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{
		Include: "shared-webhooks.yaml",
		GitHub: []GitWebhookConfig{
			// Same name as the included entry — must win over it.
			{Name: "payments-repo", Enabled: false, Path: "/webhooks/github/payments-disabled", Branch: "main"},
			{Name: "orders-repo", Enabled: true, Path: "/webhooks/github/orders", Branch: "main"},
		},
	}}

	if err := ExpandGatewayWebhookIncludes(gw, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gw.Webhooks.GitHub) != 2 {
		t.Fatalf("GitHub = %+v, want 2 entries (payments-repo overridden + orders-repo added)", gw.Webhooks.GitHub)
	}
	var payments *GitWebhookConfig
	for i := range gw.Webhooks.GitHub {
		if gw.Webhooks.GitHub[i].Name == "payments-repo" {
			payments = &gw.Webhooks.GitHub[i]
		}
	}
	if payments == nil {
		t.Fatal("payments-repo entry missing after merge")
	}
	if payments.Enabled {
		t.Error("payments-repo: Enabled = true, want the inline override (false) to win over the included entry")
	}
	if payments.Path != "/webhooks/github/payments-disabled" {
		t.Errorf("payments-repo: Path = %q, want the inline override's path", payments.Path)
	}

	if len(gw.Webhooks.Slack) != 1 || gw.Webhooks.Slack[0].Name != "platform-workspace" {
		t.Errorf("Slack = %+v, want the included platform-workspace entry (nothing inline to override it)", gw.Webhooks.Slack)
	}

	if gw.Webhooks.Include != "" {
		t.Errorf("Include = %q, want cleared after expansion", gw.Webhooks.Include)
	}
}

func TestExpandGatewayWebhookIncludes_BlockLevelInclude_MissingFile(t *testing.T) {
	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{Include: "does-not-exist.yaml"}}
	if err := ExpandGatewayWebhookIncludes(gw, t.TempDir()); err == nil {
		t.Fatal("expected an error for a missing block-level include file")
	}
}

func TestExpandGatewayWebhookIncludes_BlockLevelInclude_StrictUnknownField(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "bad.yaml", `
webhooks:
  github:
    - name: payments-repo
      enabled: true
      path: /webhooks/github/payments
      branch: main
      typo_field: oops
`)
	gw := &GatewayConfig{Webhooks: &GatewayWebhookConfig{Include: "bad.yaml"}}
	if err := ExpandGatewayWebhookIncludes(gw, dir); err == nil {
		t.Fatal("expected a strict-unmarshal error for an unknown field")
	}
}
