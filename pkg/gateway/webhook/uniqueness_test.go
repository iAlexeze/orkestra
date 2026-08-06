package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orkspace/orkestra/pkg/runtime/kordinator"
)

func fakeRuntimeServer(t *testing.T, wantField string, items []kordinator.CRSummary) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("field"); got != wantField {
			t.Errorf("request field param = %q, want %q", got, wantField)
		}
		_ = json.NewEncoder(w).Encode(kordinator.CRListResponse{
			CRD:   "website",
			Items: items,
		})
	}))
}

func TestRuntimeUniquenessChecker_Unique(t *testing.T) {
	srv := fakeRuntimeServer(t, "spec.domain", []kordinator.CRSummary{
		{Name: "site-a", Namespace: "default", Value: "a.example.com"},
	})
	defer srv.Close()

	checker := newRuntimeUniquenessChecker(context.Background(), srv.URL, "website")
	ok, err := checker.IsUnique("spec.domain", "b.example.com", "default", "site-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected unique")
	}
}

func TestRuntimeUniquenessChecker_Duplicate(t *testing.T) {
	srv := fakeRuntimeServer(t, "spec.domain", []kordinator.CRSummary{
		{Name: "site-a", Namespace: "default", Value: "shared.example.com"},
	})
	defer srv.Close()

	checker := newRuntimeUniquenessChecker(context.Background(), srv.URL, "website")
	ok, err := checker.IsUnique("spec.domain", "shared.example.com", "default", "site-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected not-unique — site-a already has this domain")
	}
}

func TestRuntimeUniquenessChecker_ExcludesSelf(t *testing.T) {
	srv := fakeRuntimeServer(t, "spec.domain", []kordinator.CRSummary{
		{Name: "site-a", Namespace: "default", Value: "a.example.com"},
	})
	defer srv.Close()

	checker := newRuntimeUniquenessChecker(context.Background(), srv.URL, "website")
	ok, err := checker.IsUnique("spec.domain", "a.example.com", "default", "site-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected unique — self should be excluded from the comparison")
	}
}

func TestRuntimeUniquenessChecker_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := newRuntimeUniquenessChecker(context.Background(), srv.URL, "website")
	_, err := checker.IsUnique("spec.domain", "a.example.com", "default", "site-a")
	if err == nil {
		t.Error("expected an error on a non-200 response")
	}
}

func TestRuntimeUniquenessChecker_Unreachable(t *testing.T) {
	checker := newRuntimeUniquenessChecker(context.Background(), "http://127.0.0.1:1", "website")
	_, err := checker.IsUnique("spec.domain", "a.example.com", "default", "site-a")
	if err == nil {
		t.Error("expected an error when the runtime is unreachable")
	}
}
