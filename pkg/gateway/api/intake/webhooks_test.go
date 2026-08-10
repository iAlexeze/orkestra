package intake

import (
	"context"
	"testing"

	"github.com/orkspace/orkestra/pkg/registry/simulate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestLoad_EmptyConfig(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	set, err := Load(context.Background(), &orktypes.GatewayWebhookConfig{}, kube, "orkestra-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set.GitHub) != 0 || len(set.GitLab) != 0 || len(set.Slack) != 0 || len(set.Generic) != 0 {
		t.Errorf("expected an empty Set, got %+v", set)
	}
}

func TestLoad_NilConfig(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	set, err := Load(context.Background(), nil, kube, "orkestra-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if set == nil {
		t.Fatal("expected a non-nil empty Set for nil config")
	}
}

func TestLoad_DisabledEntrySkipped(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	cfg := &orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{
			Name:    "payments-repo",
			Enabled: false,
			// No SecretRef/ContentTokenRef — if Load tried to resolve a
			// disabled entry's credentials, this would fail with a nil
			// pointer dereference. Its absence here is the assertion.
		}},
	}
	set, err := Load(context.Background(), cfg, kube, "orkestra-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set.GitHub) != 0 {
		t.Errorf("GitHub = %+v, want empty — the only entry is disabled", set.GitHub)
	}
}

func TestLoad_GitHubEntry_SelfBootstrapsSecrets(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	cfg := &orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{
			Name:            "payments-repo",
			Enabled:         true,
			Path:            "/webhooks/github/payments",
			Branch:          "main",
			SecretRef:       &orktypes.APISecretRef{Name: "ork-payments-github-secret", Key: "secret"},
			ContentTokenRef: &orktypes.APISecretRef{Name: "ork-payments-github-app-token", Key: "token"},
		}},
	}
	set, err := Load(context.Background(), cfg, kube, "orkestra-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set.GitHub) != 1 {
		t.Fatalf("GitHub = %+v, want 1 resolved entry", set.GitHub)
	}
	got := set.GitHub[0]
	if got.Config.Name != "payments-repo" {
		t.Errorf("Config.Name = %q, want payments-repo", got.Config.Name)
	}
	if got.Secret == "" {
		t.Error("Secret should be a self-bootstrapped, non-empty value")
	}
	if got.ContentToken == "" {
		t.Error("ContentToken should be a self-bootstrapped, non-empty value")
	}
	if got.Secret == got.ContentToken {
		t.Error("Secret and ContentToken resolved from different Secrets should not collide")
	}
}

func TestLoad_SlackEntry(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	cfg := &orktypes.GatewayWebhookConfig{
		Slack: []orktypes.SlackWebhookConfig{{
			Name:             "platform-workspace",
			Enabled:          true,
			Path:             "/webhooks/slack",
			SigningSecretRef: &orktypes.APISecretRef{Name: "ork-slack-signing-secret", Key: "secret"},
			Commands:         []string{"/deploy"},
		}},
	}
	set, err := Load(context.Background(), cfg, kube, "orkestra-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set.Slack) != 1 || set.Slack[0].SigningSecret == "" {
		t.Errorf("Slack = %+v, want 1 entry with a resolved SigningSecret", set.Slack)
	}
}

func TestLoad_GenericEntry(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	cfg := &orktypes.GatewayWebhookConfig{
		Generic: []orktypes.GenericWebhookConfig{{
			Name:      "pagerduty",
			Enabled:   true,
			Path:      "/webhooks/generic/pagerduty",
			SecretRef: &orktypes.APISecretRef{Name: "ork-pagerduty-webhook-secret", Key: "secret"},
		}},
	}
	set, err := Load(context.Background(), cfg, kube, "orkestra-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set.Generic) != 1 || set.Generic[0].Secret == "" {
		t.Errorf("Generic = %+v, want 1 entry with a resolved Secret", set.Generic)
	}
}
