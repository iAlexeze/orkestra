package intake

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestApplyState(t *testing.T) {
	cases := []struct {
		accepted bool
		gitlab   bool
		want     string
	}{
		{accepted: true, gitlab: false, want: "success"},
		{accepted: true, gitlab: true, want: "success"},
		{accepted: false, gitlab: false, want: "failure"},
		{accepted: false, gitlab: true, want: "failed"},
	}
	for _, c := range cases {
		if got := applyState(c.accepted, c.gitlab); got != c.want {
			t.Errorf("applyState(%v, %v) = %q, want %q", c.accepted, c.gitlab, got, c.want)
		}
	}
}

func TestStatusDescription(t *testing.T) {
	accepted := PushApplyResult{Path: "intent.yaml", Accepted: true}
	if got := statusDescription(accepted); got != "orkestra: applied intent.yaml" {
		t.Errorf("got %q", got)
	}

	rejected := PushApplyResult{Path: "intent.yaml", Accepted: false, Message: "name is required"}
	if got := statusDescription(rejected); got != "orkestra: rejected — name is required" {
		t.Errorf("got %q", got)
	}

	rejectedNoMessage := PushApplyResult{Path: "intent.yaml", Accepted: false}
	if got := statusDescription(rejectedNoMessage); got != "orkestra: rejected" {
		t.Errorf("got %q", got)
	}
}

func TestReportGitHubCommitStatus_PostsExpectedBody(t *testing.T) {
	var gotState, gotAuth string
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/myorg/myrepo/statuses/abc123" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			State string `json:"state"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotState = body.State
		w.WriteHeader(http.StatusCreated)
	})

	err := reportGitHubCommitStatus(context.Background(), "gh-token", "myorg", "myrepo", "abc123", "success", "orkestra: applied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "token gh-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotState != "success" {
		t.Errorf("state = %q, want success", gotState)
	}
}

func TestReportGitHubCommitStatus_NonOKStatus(t *testing.T) {
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	err := reportGitHubCommitStatus(context.Background(), "gh-token", "myorg", "myrepo", "abc123", "success", "desc")
	if err == nil {
		t.Fatal("expected an error for a non-2xx response")
	}
}

func TestReportGitLabCommitStatus_PostsExpectedQuery(t *testing.T) {
	var gotToken, gotState string
	withGitLabTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/123/statuses/abc123" {
			t.Errorf("path = %q", r.URL.Path)
		}
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		gotState = r.URL.Query().Get("state")
		w.WriteHeader(http.StatusOK)
	})

	err := reportGitLabCommitStatus(context.Background(), "gl-token", "123", "abc123", "success", "orkestra: applied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotToken != "gl-token" {
		t.Errorf("PRIVATE-TOKEN = %q", gotToken)
	}
	if gotState != "success" {
		t.Errorf("state = %q, want success", gotState)
	}
}

func TestReportGitLabCommitStatus_NonOKStatus(t *testing.T) {
	withGitLabTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	err := reportGitLabCommitStatus(context.Background(), "gl-token", "123", "abc123", "success", "desc")
	if err == nil {
		t.Fatal("expected an error for a non-2xx response")
	}
}
