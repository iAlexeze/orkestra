//go:build !runtime && !gateway

package cli

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func TestFindGitWebhookEntry_Found(t *testing.T) {
	entries := []orktypes.GitWebhookConfig{{Name: "payments-repo"}, {Name: "orders-repo"}}
	got, err := findGitWebhookEntry(entries, "github", "orders-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "orders-repo" {
		t.Errorf("Name = %q, want orders-repo", got.Name)
	}
}

func TestFindGitWebhookEntry_NotFound(t *testing.T) {
	entries := []orktypes.GitWebhookConfig{{Name: "payments-repo"}}
	_, err := findGitWebhookEntry(entries, "github", "unknown")
	if err == nil {
		t.Fatal("expected an error for an unknown entry name")
	}
}

func TestFindSlackWebhookEntry_Found(t *testing.T) {
	entries := []orktypes.SlackWebhookConfig{{Name: "platform-workspace"}}
	got, err := findSlackWebhookEntry(entries, "platform-workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "platform-workspace" {
		t.Errorf("Name = %q", got.Name)
	}
}

func TestFindSlackWebhookEntry_NotFound(t *testing.T) {
	_, err := findSlackWebhookEntry(nil, "missing")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestFindGenericWebhookEntry_Found(t *testing.T) {
	entries := []orktypes.GenericWebhookConfig{{Name: "pagerduty"}}
	got, err := findGenericWebhookEntry(entries, "pagerduty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "pagerduty" {
		t.Errorf("Name = %q", got.Name)
	}
}

func TestFindGenericWebhookEntry_NotFound(t *testing.T) {
	_, err := findGenericWebhookEntry(nil, "missing")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestJoinOrNone_Empty(t *testing.T) {
	if got := joinOrNone(nil); got != "(none declared)" {
		t.Errorf("got %q", got)
	}
}

func TestJoinOrNone_NonEmpty(t *testing.T) {
	if got := joinOrNone([]string{"a", "b"}); got != "a, b" {
		t.Errorf("got %q", got)
	}
}
