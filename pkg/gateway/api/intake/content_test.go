package intake

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestEscapeFilePath_PreservesSlashes(t *testing.T) {
	got := escapeFilePath("services/my app/intent.yaml")
	want := "services/my%20app/intent.yaml"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseIntentContent_YAML(t *testing.T) {
	fields, err := ParseIntentContent("intent.yaml", []byte("target: cronjob-tutorial\nname: daily-backup\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["target"] != "cronjob-tutorial" || fields["name"] != "daily-backup" {
		t.Errorf("fields = %+v, want target/name populated", fields)
	}
}

func TestParseIntentContent_JSON(t *testing.T) {
	fields, err := ParseIntentContent("intent.json", []byte(`{"target": "cronjob-tutorial", "name": "daily-backup"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields["target"] != "cronjob-tutorial" || fields["name"] != "daily-backup" {
		t.Errorf("fields = %+v, want target/name populated", fields)
	}
}

func TestParseIntentContent_InvalidJSON(t *testing.T) {
	_, err := ParseIntentContent("intent.json", []byte("not-json"))
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestParseIntentContent_InvalidYAML(t *testing.T) {
	_, err := ParseIntentContent("intent.yaml", []byte("key: [1, 2"))
	if err == nil {
		t.Fatal("expected an error for invalid YAML")
	}
}

func withGitHubTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	orig := githubAPIBaseURL
	githubAPIBaseURL = srv.URL
	t.Cleanup(func() { githubAPIBaseURL = orig })
}

func withGitLabTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	orig := gitlabAPIBaseURL
	gitlabAPIBaseURL = srv.URL
	t.Cleanup(func() { gitlabAPIBaseURL = orig })
}

func TestFetchGitHubFileContent_DecodesBase64(t *testing.T) {
	want := []byte("target: apprequest\nname: payments-api\n")
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/myorg/myrepo/contents/intent.yaml" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "token gh-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":"` + base64.StdEncoding.EncodeToString(want) + `","encoding":"base64"}`))
	})

	got, err := fetchGitHubFileContent(context.Background(), "gh-token", "myorg", "myrepo", "intent.yaml", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFetchGitHubFileContent_NonOKStatus(t *testing.T) {
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := fetchGitHubFileContent(context.Background(), "gh-token", "myorg", "myrepo", "intent.yaml", "main")
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}

func TestFetchGitLabFileContent_ReturnsRawBody(t *testing.T) {
	want := []byte("target: apprequest\nname: payments-api\n")
	withGitLabTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/123/repository/files/intent.yaml/raw" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("PRIVATE-TOKEN") != "gl-token" {
			t.Errorf("PRIVATE-TOKEN = %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		_, _ = w.Write(want)
	})

	got, err := fetchGitLabFileContent(context.Background(), "gl-token", "123", "intent.yaml", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFetchGitLabFileContent_NonOKStatus(t *testing.T) {
	withGitLabTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := fetchGitLabFileContent(context.Background(), "gl-token", "123", "intent.yaml", "main")
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}
