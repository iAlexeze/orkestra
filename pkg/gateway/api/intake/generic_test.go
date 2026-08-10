package intake

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/registry/simulate"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
)

// intakeTestKatalog builds a real, lookup-ready *katalog.Katalog with one
// serve-enabled, target-mode CRD — mirrors api's own appRequestKatalog
// test helper, duplicated here since it's unexported in that package.
func intakeTestKatalog(serveName string) *katalog.Katalog {
	return katalog.NewFromEntryPointers(map[string]*orktypes.CRDEntry{
		"apprequest": {
			APITypes: orktypes.APITypes{
				Group:   "platform.myorg.io",
				Version: "v1",
				Kind:    "AppRequest",
				Plural:  "apprequests",
			},
			Serve: &orktypes.ServeConfig{Enabled: true, Name: serveName, Namespace: "default"},
		},
	})
}

func testGenericSource(secret string) ResolvedGenericSource {
	return ResolvedGenericSource{
		Config: orktypes.GenericWebhookConfig{Name: "pagerduty", Enabled: true, Path: "/webhooks/generic/pagerduty"},
		Secret: secret,
	}
}

func TestGenericHandler_MethodNotAllowed(t *testing.T) {
	h := NewGenericHandler(testGenericSource("s3cr3t"), nil, nil, orktypes.NoteRegistry{})
	req := httptest.NewRequest(http.MethodGet, "/webhooks/generic/pagerduty", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestGenericHandler_MissingSignature(t *testing.T) {
	h := NewGenericHandler(testGenericSource("s3cr3t"), nil, nil, orktypes.NoteRegistry{})
	body, _ := json.Marshal(map[string]interface{}{"target": "apprequest", "name": "my-app"})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/generic/pagerduty", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestGenericHandler_WrongSignature(t *testing.T) {
	h := NewGenericHandler(testGenericSource("s3cr3t"), nil, nil, orktypes.NoteRegistry{})
	body, _ := json.Marshal(map[string]interface{}{"target": "apprequest", "name": "my-app"})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/generic/pagerduty", bytes.NewReader(body))
	req.Header.Set("X-Signature-256", signBody("wrong-secret", body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestGenericHandler_InvalidJSON(t *testing.T) {
	h := NewGenericHandler(testGenericSource("s3cr3t"), nil, nil, orktypes.NoteRegistry{})
	body := []byte("not-json")
	req := httptest.NewRequest(http.MethodPost, "/webhooks/generic/pagerduty", bytes.NewReader(body))
	req.Header.Set("X-Signature-256", signBody("s3cr3t", body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestGenericHandler_ValidSignature_ReachesApplyPipeline(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := intakeTestKatalog("")
	h := NewGenericHandler(testGenericSource("s3cr3t"), kube, kat, orktypes.NoteRegistry{})

	// No "name" and serve.name isn't declared — this should reach the apply
	// pipeline's own "name is required" rejection (422), not get stuck at
	// signature verification (401) or JSON decoding (400). That's the
	// signal the request made it all the way through.
	body, _ := json.Marshal(map[string]interface{}{"target": "apprequest"})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/generic/pagerduty", bytes.NewReader(body))
	req.Header.Set("X-Signature-256", signBody("s3cr3t", body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (reached the apply pipeline's name-required check)", rr.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["message"] != "name is required" {
		t.Errorf("message = %v, want %q", resp["message"], "name is required")
	}
}

func TestGenericHandler_DryRunPropagates(t *testing.T) {
	kube := simulate.NewFakeKubeclient(runtime.NewScheme())
	kat := intakeTestKatalog("resolved-name")
	h := NewGenericHandler(testGenericSource("s3cr3t"), kube, kat, orktypes.NoteRegistry{})

	body, _ := json.Marshal(map[string]interface{}{"target": "apprequest"})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/generic/pagerduty?dryRun=true", bytes.NewReader(body))
	req.Header.Set("X-Signature-256", signBody("s3cr3t", body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dryRun, _ := resp["dryRun"].(bool); !dryRun {
		t.Errorf("dryRun = %v, want true", resp["dryRun"])
	}
}
