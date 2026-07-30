package applyapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// noopLookup is a CRD lookup that always returns nil — used in tests that
// don't exercise the forceConflict path.
func noopLookup(_ string) *orktypes.CRDEntry { return nil }

func TestApplyHandler_MethodNotAllowed(t *testing.T) {
	h := applyHandler(nil, noopLookup, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apply", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestApplyHandler_InvalidJSON(t *testing.T) {
	h := applyHandler(nil, noopLookup, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/apply", strings.NewReader("not-json"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
	var resp ApplyResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Accepted {
		t.Error("Accepted should be false for invalid JSON")
	}
	if !strings.Contains(resp.Message, "invalid JSON") {
		t.Errorf("message = %q, want to contain 'invalid JSON'", resp.Message)
	}
}

func TestApplyHandler_EmptyBody(t *testing.T) {
	h := applyHandler(nil, noopLookup, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/apply", bytes.NewReader([]byte{}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestApplyResponse_JSON(t *testing.T) {
	resp := ApplyResponse{
		Accepted:        true,
		Name:            "my-app",
		Namespace:       "team-payments",
		Kind:            "PlatformResource",
		APIVersion:      "platform.orkestra.io/v1alpha1",
		ResourceVersion: "12345",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ApplyResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "my-app" || got.Namespace != "team-payments" || !got.Accepted {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestApplyResponse_DryRunAndViolations(t *testing.T) {
	resp := ApplyResponse{
		Accepted: false,
		DryRun:   true,
		Message:  "admission webhook denied the request",
		Violations: []ApplyViolation{
			{Field: "spec.environment", Message: "must be staging or production", Severity: "error"},
			{Field: "spec.team", Message: "required field", Severity: "error"},
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ApplyResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Accepted {
		t.Error("Accepted should be false")
	}
	if !got.DryRun {
		t.Error("DryRun should be true")
	}
	if len(got.Violations) != 2 {
		t.Fatalf("Violations len = %d, want 2", len(got.Violations))
	}
	if got.Violations[0].Field != "spec.environment" {
		t.Errorf("Violations[0].Field = %q", got.Violations[0].Field)
	}
	if got.Violations[0].Severity != "error" {
		t.Errorf("Violations[0].Severity = %q", got.Violations[0].Severity)
	}
}

func TestExtractViolations_NoStatusError(t *testing.T) {
	// A plain error produces no violations.
	vs := extractViolations(fmt.Errorf("some error"))
	if len(vs) != 0 {
		t.Errorf("expected no violations for plain error, got %v", vs)
	}
}
