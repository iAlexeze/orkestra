package intake

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orkspace/orkestra/pkg/registry/simulate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
)

func gitlabPushBody(t *testing.T, ref string, added, modified []string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{
		"ref":          ref,
		"checkout_sha": "abc123",
		"project":      map[string]interface{}{"id": 123},
		"commits": []map[string]interface{}{
			{"added": added, "modified": modified},
		},
	})
	if err != nil {
		t.Fatalf("marshal push body: %v", err)
	}
	return body
}

func gitlabPushRequest(t *testing.T, token string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/gitlab/payments", bytes.NewReader(body))
	req.Header.Set("X-Gitlab-Token", token)
	return req
}

func TestGitLabHandler_MethodNotAllowed(t *testing.T) {
	h := NewGitLabHandler(testGitSource("s3cr3t", "gl-token", "main"), nil, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodGet, "/webhooks/gitlab/payments", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestGitLabHandler_InvalidToken(t *testing.T) {
	h := NewGitLabHandler(testGitSource("s3cr3t", "gl-token", "main"), nil, nil, orktypes.NoteRegistry{})
	body := gitlabPushBody(t, "refs/heads/main", []string{"intent.yaml"}, nil)
	req := gitlabPushRequest(t, "wrong-token", body)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestGitLabHandler_BranchNotWatched(t *testing.T) {
	h := NewGitLabHandler(testGitSource("s3cr3t", "gl-token", "main"), nil, nil, orktypes.NoteRegistry{})
	body := gitlabPushBody(t, "refs/heads/feature-x", []string{"intent.yaml"}, nil)
	req := gitlabPushRequest(t, "s3cr3t", body)
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

func TestGitLabHandler_NoWatchedFilesChanged(t *testing.T) {
	h := NewGitLabHandler(testGitSource("s3cr3t", "gl-token", "main", "services/*/intent.yaml"), nil, nil, orktypes.NoteRegistry{})
	body := gitlabPushBody(t, "refs/heads/main", []string{"README.md"}, nil)
	req := gitlabPushRequest(t, "s3cr3t", body)
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

func TestGitLabHandler_MatchedFile_FetchesAndReachesApplyPipeline(t *testing.T) {
	intent := []byte("target: apprequest\n")
	withGitLabTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/123/repository/files/services/payments/intent.yaml/raw" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write(intent)
	})

	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := intakeTestKatalog("")
	h := NewGitLabHandler(testGitSource("s3cr3t", "gl-token", "main", "services/*/intent.yaml"), kube, kat, orktypes.NoteRegistry{})

	body := gitlabPushBody(t, "refs/heads/main", []string{"services/payments/intent.yaml"}, nil)
	req := gitlabPushRequest(t, "s3cr3t", body)
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
	if resp.Applied[0].Message != "name is required" {
		t.Errorf("Message = %q, want %q", resp.Applied[0].Message, "name is required")
	}
}

func TestGitLabHandler_ContentFetchFailure_ReportedPerFile(t *testing.T) {
	withGitLabTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := intakeTestKatalog("")
	h := NewGitLabHandler(testGitSource("s3cr3t", "gl-token", "main", "services/*/intent.yaml"), kube, kat, orktypes.NoteRegistry{})

	body := gitlabPushBody(t, "refs/heads/main", []string{"services/payments/intent.yaml"}, nil)
	req := gitlabPushRequest(t, "s3cr3t", body)
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
