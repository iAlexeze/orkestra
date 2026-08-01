package applyapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParsePath(t *testing.T) {
	cases := []struct {
		path      string
		wantKind  string
		wantNS    string
		wantName  string
		wantError bool
	}{
		{"/api/v1/resources/platformresource/team-payments", "platformresource", "team-payments", "", false},
		{"/api/v1/resources/platformresource/team-payments/my-app", "platformresource", "team-payments", "my-app", false},
		{"/api/v1/resources/platform/default/app-1", "platform", "default", "app-1", false},
		{"/api/v1/resources/kind", "", "", "", true},
		{"/api/v1/resources/", "", "", "", true},
	}
	for _, tc := range cases {
		kind, ns, name, err := parsePath(tc.path)
		if tc.wantError {
			if err == nil {
				t.Errorf("parsePath(%q): want error, got nil", tc.path)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePath(%q): unexpected error: %v", tc.path, err)
			continue
		}
		if kind != tc.wantKind || ns != tc.wantNS || name != tc.wantName {
			t.Errorf("parsePath(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.path, kind, ns, name, tc.wantKind, tc.wantNS, tc.wantName)
		}
	}
}

func TestResourcesHandler_UnknownKind(t *testing.T) {
	errMapper := KindMapper(func(kind string) (schema.GroupVersionResource, error) {
		return schema.GroupVersionResource{}, fmt.Errorf("no mapping for %q", kind)
	})
	h := resourcesHandler(nil, errMapper, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/unknown/default", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestResourcesHandler_MethodNotAllowed(t *testing.T) {
	h := resourcesHandler(nil, func(kind string) (schema.GroupVersionResource, error) {
		return schema.GroupVersionResource{}, nil
	}, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/resources/platform/default/x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestResourcesHandler_BadPath(t *testing.T) {
	h := resourcesHandler(nil, func(kind string) (schema.GroupVersionResource, error) {
		return schema.GroupVersionResource{}, nil
	}, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/onlyone", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestResourcesHandler_DeleteRequiresName(t *testing.T) {
	h := resourcesHandler(nil, func(kind string) (schema.GroupVersionResource, error) {
		return schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "things"}, nil
	}, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/resources/thing/default", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestWriteKubeError(t *testing.T) {
	cases := []struct {
		msg      string
		wantCode int
	}{
		{"resource not found", http.StatusNotFound},
		{"NotFound: no such resource", http.StatusNotFound},
		{"forbidden: access denied", http.StatusForbidden},
		{"internal server error", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		writeKubeError(rr, fmt.Errorf("%s", tc.msg))
		if rr.Code != tc.wantCode {
			t.Errorf("msg=%q: status = %d, want %d", tc.msg, rr.Code, tc.wantCode)
		}
	}
}
