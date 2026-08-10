//go:build !runtime && !gateway

package cli

import (
	"path/filepath"
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// webhookPlayTestKatalog reuses chainTestKatalog's "servicerequest" CRD
// (no token restrictions -> allow all) and attaches gateway.webhooks so the
// play* functions can look up entries by name. Rebuilds the lookup indexes
// after attaching Gateway — chainTestKatalog already built them once with
// no Gateway set, so the webhook name index would otherwise stay empty.
func webhookPlayTestKatalog(webhooks *orktypes.GatewayWebhookConfig) *katalog.Katalog {
	k := chainTestKatalog("", nil)
	k.Gateway = &orktypes.GatewayConfig{Webhooks: webhooks}
	k.BuildLookupIndexes()
	return k
}

func TestPlayGenericWebhook_UnknownEntry(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{})
	if err := playGenericWebhook(k, "missing", "body.json", webhookSimulate{}); err == nil {
		t.Fatal("expected an error for an unknown webhook entry")
	}
}

func TestPlayGenericWebhook_MissingBodyFlag(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		Generic: []orktypes.GenericWebhookConfig{{Name: "pagerduty"}},
	})
	if err := playGenericWebhook(k, "pagerduty", "", webhookSimulate{}); err == nil {
		t.Fatal("expected an error when --body is empty")
	}
}

func TestPlayGenericWebhook_Success(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		Generic: []orktypes.GenericWebhookConfig{{Name: "pagerduty"}},
	})
	path := filepath.Join(t.TempDir(), "body.json")
	writeFile(t, path, `{"target":"servicerequest","name":"x"}`)

	if err := playGenericWebhook(k, "pagerduty", path, webhookSimulate{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaySlackWebhook_UnknownCommand(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		Slack: []orktypes.SlackWebhookConfig{{Name: "platform-workspace", Commands: []string{"/deploy"}}},
	})
	err := playSlackWebhook(k, "platform-workspace", "/rollback", "servicerequest name=x", webhookSimulate{})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
}

func TestPlaySlackWebhook_Success(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		Slack: []orktypes.SlackWebhookConfig{{Name: "platform-workspace", Commands: []string{"/deploy"}}},
	})
	err := playSlackWebhook(k, "platform-workspace", "/deploy", "servicerequest name=x", webhookSimulate{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaySlackWebhook_MissingCommandFlag(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		Slack: []orktypes.SlackWebhookConfig{{Name: "platform-workspace", Commands: []string{"/deploy"}}},
	})
	if err := playSlackWebhook(k, "platform-workspace", "", "text", webhookSimulate{}); err == nil {
		t.Fatal("expected an error when --command is empty")
	}
}

func TestPlaySlackWebhook_MissingTextFlag(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		Slack: []orktypes.SlackWebhookConfig{{Name: "platform-workspace", Commands: []string{"/deploy"}}},
	})
	if err := playSlackWebhook(k, "platform-workspace", "/deploy", "", webhookSimulate{}); err == nil {
		t.Fatal("expected an error when --text is empty")
	}
}

func TestPlayGitPushWebhook_UnknownEntry(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{})
	err := playGitPushWebhook(k, "github", "missing", "event.json", nil, webhookSimulate{})
	if err == nil {
		t.Fatal("expected an error for an unknown webhook entry")
	}
}

func TestPlayGitPushWebhook_MissingEventFlag(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{Name: "payments-repo", Branch: "main"}},
	})
	if err := playGitPushWebhook(k, "github", "payments-repo", "", nil, webhookSimulate{}); err == nil {
		t.Fatal("expected an error when --event is empty")
	}
}

func TestPlayGitPushWebhook_BranchMismatch(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{Name: "payments-repo", Branch: "main"}},
	})
	event := filepath.Join(t.TempDir(), "event.json")
	writeFile(t, event, `{"ref":"refs/heads/feature-x","repository":{"name":"r","owner":{"login":"o"}},"commits":[{"added":["a"]}]}`)

	if err := playGitPushWebhook(k, "github", "payments-repo", event, nil, webhookSimulate{}); err == nil {
		t.Fatal("expected an error for a branch mismatch")
	}
}

func TestPlayGitPushWebhook_NoWatchMatch(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{
			Name: "payments-repo", Branch: "main",
			Watch: []string{"services/*/intent.yaml"},
		}},
	})
	event := filepath.Join(t.TempDir(), "event.json")
	writeFile(t, event, `{"ref":"refs/heads/main","repository":{"name":"r","owner":{"login":"o"}},"commits":[{"added":["README.md"]}]}`)

	if err := playGitPushWebhook(k, "github", "payments-repo", event, nil, webhookSimulate{}); err == nil {
		t.Fatal("expected an error when no changed file matches watch")
	}
}

func TestPlayGitPushWebhook_NoFetchOverride(t *testing.T) {
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{
			Name: "payments-repo", Branch: "main",
			Watch: []string{"services/*/intent.yaml"},
		}},
	})
	event := filepath.Join(t.TempDir(), "event.json")
	writeFile(t, event, `{"ref":"refs/heads/main","repository":{"name":"r","owner":{"login":"o"}},"commits":[{"added":["services/a/intent.yaml"]}]}`)

	err := playGitPushWebhook(k, "github", "payments-repo", event, nil, webhookSimulate{})
	if err == nil {
		t.Fatal("expected an error when no --fetch override is supplied for a matched file")
	}
}

func TestPlayGitPushWebhook_GitHubSuccess(t *testing.T) {
	dir := t.TempDir()
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		GitHub: []orktypes.GitWebhookConfig{{
			Name: "payments-repo", Branch: "main",
			Watch: []string{"services/*/intent.yaml"},
		}},
	})
	event := filepath.Join(dir, "event.json")
	writeFile(t, event, `{"ref":"refs/heads/main","repository":{"name":"r","owner":{"login":"o"}},"commits":[{"added":["services/a/intent.yaml"]}]}`)
	local := filepath.Join(dir, "local-intent.yaml")
	writeFile(t, local, "target: servicerequest\nname: payments-api\n")

	err := playGitPushWebhook(k, "github", "payments-repo", event,
		[]string{"services/a/intent.yaml=" + local}, webhookSimulate{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlayGitPushWebhook_GitLabSuccess(t *testing.T) {
	dir := t.TempDir()
	k := webhookPlayTestKatalog(&orktypes.GatewayWebhookConfig{
		GitLab: []orktypes.GitWebhookConfig{{
			Name: "payments-repo-gitlab", Branch: "main",
			Watch: []string{"services/*/intent.yaml"},
		}},
	})
	event := filepath.Join(dir, "event.json")
	writeFile(t, event, `{"ref":"refs/heads/main","checkout_sha":"abc","project":{"id":1},"commits":[{"added":["services/a/intent.yaml"]}]}`)
	local := filepath.Join(dir, "local-intent.yaml")
	writeFile(t, local, "target: servicerequest\nname: payments-api\n")

	err := playGitPushWebhook(k, "gitlab", "payments-repo-gitlab", event,
		[]string{"services/a/intent.yaml=" + local}, webhookSimulate{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
