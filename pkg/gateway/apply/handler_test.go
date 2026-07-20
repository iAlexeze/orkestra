package apply

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := Handler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apply", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	h := Handler(nil)
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

func TestHandler_EmptyBody(t *testing.T) {
	h := Handler(nil)
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
