package gateway

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestTokenSet_Matches(t *testing.T) {
	ts := &TokenSet{
		entries: []resolvedToken{
			{name: "ci", value: "token-abc"},
			{name: "dev", value: "token-xyz"},
		},
	}

	if got := ts.Matches("token-abc"); got != "ci" {
		t.Errorf("Matches(token-abc) = %q, want %q", got, "ci")
	}
	if got := ts.Matches("token-xyz"); got != "dev" {
		t.Errorf("Matches(token-xyz) = %q, want %q", got, "dev")
	}
	if got := ts.Matches("unknown"); got != "" {
		t.Errorf("Matches(unknown) = %q, want empty", got)
	}
	if got := ts.Matches(""); got != "" {
		t.Errorf("Matches('') = %q, want empty", got)
	}
}

func TestAuthMiddleware_Valid(t *testing.T) {
	ts := &TokenSet{entries: []resolvedToken{{name: "ci", value: "secret-token"}}}
	called := false
	handler := AuthMiddleware(ts, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if name := TokenNameFromContext(r.Context()); name != "ci" {
			t.Errorf("token name in ctx = %q, want %q", name, "ci")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/apply", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestAuthMiddleware_Invalid(t *testing.T) {
	ts := &TokenSet{entries: []resolvedToken{{name: "ci", value: "secret-token"}}}
	handler := AuthMiddleware(ts, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called on invalid token")
	}))

	cases := []string{"Bearer wrong", "wrong", "", "Bearer "}
	for _, auth := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/apply", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("auth=%q: status = %d, want 401", auth, rr.Code)
		}
	}
}

func TestExpandEnvToken(t *testing.T) {
	t.Setenv("MY_TEST_TOKEN", "expanded-value")

	val, err := expandEnvToken("${MY_TEST_TOKEN}")
	if err != nil {
		t.Fatalf("expandEnvToken: %v", err)
	}
	if val != "expanded-value" {
		t.Errorf("got %q, want %q", val, "expanded-value")
	}

	_, err = expandEnvToken("literal-value")
	if err == nil {
		t.Error("expected error for literal value")
	}

	_, err = expandEnvToken("${UNSET_VAR_XYZ_ABC}")
	if err == nil {
		t.Error("expected error for unset env var")
	}
}

func TestGenerateUUIDv4(t *testing.T) {
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	u1, err := generateUUIDv4()
	if err != nil {
		t.Fatalf("generateUUIDv4: %v", err)
	}
	if !uuidRe.MatchString(u1) {
		t.Errorf("uuid %q does not match v4 format", u1)
	}

	u2, _ := generateUUIDv4()
	if u1 == u2 {
		t.Error("generateUUIDv4 produced duplicate value")
	}
}

func TestTokenNameFromContext_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if name := TokenNameFromContext(req.Context()); name != "" {
		t.Errorf("got %q, want empty", name)
	}
}
