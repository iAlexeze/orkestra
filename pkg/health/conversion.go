// health/conversion.go
package health

import (
	"encoding/json"
	"net/http"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/metrics"
)

// --- Kubernetes-style ConversionReview types ---
type ConversionReview struct {
	APIVersion string                    `json:"apiVersion"`
	Kind       string                    `json:"kind"`
	Request    *ConversionReviewRequest  `json:"request,omitempty"`
	Response   *ConversionReviewResponse `json:"response,omitempty"`
}

type ConversionReviewRequest struct {
	UID               string            `json:"uid"`
	DesiredAPIVersion string            `json:"desiredAPIVersion"`
	Objects           []json.RawMessage `json:"objects"`
}

type ConversionReviewResponse struct {
	UID              string            `json:"uid"`
	ConvertedObjects []json.RawMessage `json:"convertedObjects"`
	Result           *Status           `json:"result"`
}

type Status struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// --- HTTP handler for /convert ---
func (h *HealthServer) conversionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Track active requests
	metrics.ConversionActiveRequests.Inc()
	defer metrics.ConversionActiveRequests.Dec()
	
	var review ConversionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "invalid ConversionReview", http.StatusBadRequest)
		return
	}
	if review.Request == nil {
		http.Error(w, "missing request", http.StatusBadRequest)
		return
	}

	resp := &ConversionReviewResponse{
		UID:              review.Request.UID,
		ConvertedObjects: make([]json.RawMessage, len(review.Request.Objects)),
		Result:           &Status{Status: "Success"},
	}

	for i, raw := range review.Request.Objects {
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			resp.Result = &Status{Status: "Failure", Message: "invalid object payload"}
			break
		}

		kind, _ := obj["kind"].(string)
		if kind == "" {
			resp.Result = &Status{Status: "Failure", Message: "object missing kind"}
			break
		}

		rules := h.katalog.GetConversionRules(kind)
		if rules == nil {
			resp.Result = &Status{Status: "Failure", Message: "no conversion rules for kind"}
			break
		}

		converted, err := applyConversion(obj, rules, review.Request.DesiredAPIVersion)
		if err != nil {
			resp.Result = &Status{Status: "Failure", Message: err.Error()}
			break
		}

		out, _ := json.Marshal(converted)
		resp.ConvertedObjects[i] = out
	}

	response := ConversionReview{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "ConversionReview",
		Response:   resp,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// For tests
func processConversion(review ConversionReview, rulesRegistry katalog.ConversionRegistry) ConversionReview {
	resp := &ConversionReviewResponse{
		UID:              review.Request.UID,
		ConvertedObjects: make([]json.RawMessage, len(review.Request.Objects)),
		Result:           &Status{Status: "Success"},
	}

	for i, raw := range review.Request.Objects {
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			resp.Result = &Status{Status: "Failure", Message: "invalid object payload"}
			break
		}

		kind, _ := obj["kind"].(string)
		if kind == "" {
			resp.Result = &Status{Status: "Failure", Message: "object missing kind"}
			break
		}

		rules := rulesRegistry.GetConversionRules(kind)
		if rules == nil {
			resp.Result = &Status{Status: "Failure", Message: "no conversion rules for kind"}
			break
		}

		converted, err := applyConversion(obj, rules, review.Request.DesiredAPIVersion)
		if err != nil {
			resp.Result = &Status{Status: "Failure", Message: err.Error()}
			break
		}

		out, _ := json.Marshal(converted)
		resp.ConvertedObjects[i] = out
	}

	return ConversionReview{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "ConversionReview",
		Response:   resp,
	}
}

// ProcessConversionReviewForTest exposes the conversion logic for unit tests.
func ProcessConversionReviewForTest(review ConversionReview, registry katalog.ConversionRegistry) ConversionReview {
	return processConversion(review, registry)
}
