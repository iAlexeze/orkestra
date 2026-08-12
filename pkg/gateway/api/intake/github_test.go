package intake

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orkspace/orkestra/pkg/registry/simulate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
)

func testGitSource(secret, contentToken, branch string, watch ...string) ResolvedGitSource {
	return ResolvedGitSource{
		Config: orktypes.GitWebhookConfig{
			Name:    "payments-repo",
			Enabled: true,
			Path:    "/webhooks/github/payments",
			Branch:  branch,
			Watch:   watch,
		},
		Secret:       secret,
		ContentToken: contentToken,
	}
}

func githubPushBody(t *testing.T, ref string, added, modified []string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"ref": ref,
		"repository": map[string]interface{}{
			"name":  "myrepo",
			"owner": map[string]interface{}{"login": "myorg"},
		},
		"commits": []map[string]interface{}{
			{"added": added, "modified": modified},
		},
		"after": "abc123",
	})
	if err != nil {
		t.Fatalf("marshal push body: %v", err)
	}
	return body
}

func githubPushRequest(t *testing.T, secret string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github/payments", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", signBody(secret, body))
	return req
}

func TestGitHubHandler_MethodNotAllowed(t *testing.T) {
	h := NewGitHubHandler(testGitSource("s3cr3t", "gh-token", "main"), nil, nil, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodGet, "/webhooks/github/payments", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestGitHubHandler_InvalidSignature(t *testing.T) {
	h := NewGitHubHandler(testGitSource("s3cr3t", "gh-token", "main"), nil, nil, nil, orktypes.NoteRegistry{})
	body := githubPushBody(t, "refs/heads/main", []string{"intent.yaml"}, nil)
	req := githubPushRequest(t, "wrong-secret", body)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestGitHubHandler_BranchNotWatched(t *testing.T) {
	h := NewGitHubHandler(testGitSource("s3cr3t", "gh-token", "main"), nil, nil, nil, orktypes.NoteRegistry{})
	body := githubPushBody(t, "refs/heads/feature-x", []string{"intent.yaml"}, nil)
	req := githubPushRequest(t, "s3cr3t", body)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp PushResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Message != "branch not watched" {
		t.Errorf("Message = %q, want %q", resp.Message, "branch not watched")
	}
}

func TestGitHubHandler_NoWatchedFilesChanged(t *testing.T) {
	h := NewGitHubHandler(testGitSource("s3cr3t", "gh-token", "main", "services/*/intent.yaml"), nil, nil, nil, orktypes.NoteRegistry{})
	body := githubPushBody(t, "refs/heads/main", []string{"README.md"}, nil)
	req := githubPushRequest(t, "s3cr3t", body)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp PushResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Message != "no watched files changed" {
		t.Errorf("Message = %q, want %q", resp.Message, "no watched files changed")
	}
}

func TestGitHubHandler_MatchedFile_FetchesAndReachesApplyPipeline(t *testing.T) {
	intent := []byte("target: apprequest\n")
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/myorg/myrepo/contents/services/payments/intent.yaml" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":"` + base64.StdEncoding.EncodeToString(intent) + `","encoding":"base64"}`))
	})

	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := intakeTestKatalog("")
	h := NewGitHubHandler(testGitSource("s3cr3t", "gh-token", "main", "services/*/intent.yaml"), kube, nil, kat, orktypes.NoteRegistry{})

	body := githubPushBody(t, "refs/heads/main", []string{"services/payments/intent.yaml"}, nil)
	req := githubPushRequest(t, "s3cr3t", body)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp PushResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Applied) != 1 {
		t.Fatalf("Applied = %+v, want 1 entry", resp.Applied)
	}
	// serve.name isn't declared and the intent carries no raw "name" — this
	// should reach the apply pipeline's own "name is required" rejection,
	// not get stuck at content fetch or JSON/YAML parsing. That's the signal
	// the request made it all the way through.
	if resp.Applied[0].Message != "name is required" {
		t.Errorf("Message = %q, want %q", resp.Applied[0].Message, "name is required")
	}
}

func TestGitHubHandler_ContentFetchFailure_ReportedPerFile(t *testing.T) {
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := intakeTestKatalog("")
	h := NewGitHubHandler(testGitSource("s3cr3t", "gh-token", "main", "services/*/intent.yaml"), kube, nil, kat, orktypes.NoteRegistry{})

	body := githubPushBody(t, "refs/heads/main", []string{"services/payments/intent.yaml"}, nil)
	req := githubPushRequest(t, "s3cr3t", body)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fetch failures are reported per-file, not a delivery failure)", rr.Code)
	}
	var resp PushResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Applied) != 1 || resp.Applied[0].Accepted {
		t.Errorf("Applied = %+v, want one unaccepted entry", resp.Applied)
	}
}
