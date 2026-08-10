package api

import (
	"context"
	"testing"

	"github.com/orkspace/orkestra/pkg/registry/simulate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestApplyTargetFields_UnknownTarget(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := appRequestKatalog("")
	resp, status := ApplyTargetFields(context.Background(), kube, kat, orktypes.NoteRegistry{}, "payments-repo",
		map[string]interface{}{"target": "does-not-exist"}, false,
	)
	if status != 400 {
		t.Fatalf("status = %d, want 400", status)
	}
	if resp.Accepted {
		t.Error("Accepted should be false for an unknown target")
	}
}

func TestApplyTargetFields_MissingTarget(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := appRequestKatalog("")
	resp, status := ApplyTargetFields(context.Background(), kube, kat, orktypes.NoteRegistry{}, "payments-repo",
		map[string]interface{}{"name": "my-app"}, false,
	)
	if status != 400 {
		t.Fatalf("status = %d, want 400", status)
	}
	if resp.Accepted {
		t.Error("Accepted should be false when target is missing")
	}
}

func TestApplyTargetFields_MissingName_Rejected(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := appRequestKatalog("") // serve.name not declared
	resp, status := ApplyTargetFields(context.Background(), kube, kat, orktypes.NoteRegistry{}, "payments-repo",
		map[string]interface{}{"target": "apprequest"}, false,
	)
	if status != 422 {
		t.Fatalf("status = %d, want 422", status)
	}
	if resp.Message != "name is required" {
		t.Errorf("Message = %q, want %q", resp.Message, "name is required")
	}
}

func TestApplyTargetFields_RawNameFallback_NotRejectedForName(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := appRequestKatalog("") // serve.name not declared — falls back to raw "name"
	resp, _ := ApplyTargetFields(context.Background(), kube, kat, orktypes.NoteRegistry{}, "payments-repo",
		map[string]interface{}{"target": "apprequest", "name": "payments-api"}, false,
	)
	if resp.Message == "name is required" {
		t.Errorf("got the missing-name rejection even though a raw name was supplied: %+v", resp)
	}
}

func TestApplyTargetFields_ServeName_ResolvesWithoutRawName(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := appRequestKatalog("resolved-name")
	resp, _ := ApplyTargetFields(context.Background(), kube, kat, orktypes.NoteRegistry{}, "payments-repo",
		map[string]interface{}{"target": "apprequest"}, false,
	)
	if resp.Message == "name is required" {
		t.Errorf("got the missing-name rejection even though serve.name is declared: %+v", resp)
	}
}

func TestApplyTargetFields_DryRunPropagates(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := appRequestKatalog("resolved-name")
	resp, _ := ApplyTargetFields(context.Background(), kube, kat, orktypes.NoteRegistry{}, "payments-repo",
		map[string]interface{}{"target": "apprequest"}, true,
	)
	if !resp.DryRun {
		t.Error("DryRun should be true when dryRun=true is passed")
	}
}
