// health/conversion.go
package health

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/metrics"
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

	metrics.IncConversionRequests()
	defer metrics.DecConversionRequests()

	var review ConversionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		h.conversionStats.RecordFailure()
		metrics.RecordConversionError("unknown", "invalid_request")
		http.Error(w, "invalid ConversionReview", http.StatusBadRequest)
		return
	}
	if review.Request == nil {
		h.conversionStats.RecordFailure()
		metrics.RecordConversionError("unknown", "missing_request")
		http.Error(w, "missing request", http.StatusBadRequest)
		return
	}

	// Extract source version from the first object
	sourceVersion := ""
	if len(review.Request.Objects) > 0 {
		var firstObj map[string]interface{}
		if err := json.Unmarshal(review.Request.Objects[0], &firstObj); err == nil {
			if apiVersion, ok := firstObj["apiVersion"].(string); ok {
				if parts := strings.Split(apiVersion, "/"); len(parts) == 2 {
					sourceVersion = parts[1]
				}
			}
		}
	}

	targetVersion := ""
	if desired := review.Request.DesiredAPIVersion; desired != "" {
		if parts := strings.Split(desired, "/"); len(parts) == 2 {
			targetVersion = parts[1]
		}
	}

	resp := &ConversionReviewResponse{
		UID:              review.Request.UID,
		ConvertedObjects: make([]json.RawMessage, len(review.Request.Objects)),
		Result:           &Status{Status: "Success"},
	}

	var kind string

	for i, raw := range review.Request.Objects {
		objStart := time.Now()

		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			resp.Result = &Status{Status: "Failure", Message: "invalid object payload"}
			metrics.RecordConversionError(kind, "invalid_payload")
			metrics.RecordConversion(kind, sourceVersion, targetVersion, "failure")
			h.conversionStats.RecordFailure()
			break
		}

		kind, _ = obj["kind"].(string)
		if kind == "" {
			resp.Result = &Status{Status: "Failure", Message: "object missing kind"}
			metrics.RecordConversionError(kind, "missing_kind")
			metrics.RecordConversion(kind, sourceVersion, targetVersion, "failure")
			h.conversionStats.RecordFailure()
			break
		}

		rules := h.conversionRegistry.GetConversionRules(kind)
		if rules == nil {
			resp.Result = &Status{Status: "Failure", Message: "no conversion rules for kind"}
			metrics.RecordConversionError(kind, "no_rules")
			metrics.RecordConversion(kind, sourceVersion, targetVersion, "failure")
			h.conversionStats.RecordFailure()
			break
		}

		converted, err := applyConversion(obj, rules, review.Request.DesiredAPIVersion)
		if err != nil {
			resp.Result = &Status{Status: "Failure", Message: err.Error()}
			metrics.RecordConversionError(kind, "apply_failed")
			metrics.RecordConversion(kind, sourceVersion, targetVersion, "failure")
			h.conversionStats.RecordFailure()
			break
		}

		// Success
		objDuration := time.Since(objStart).Seconds()
		metrics.ObserveConversionDuration(kind, sourceVersion, targetVersion, objDuration)
		metrics.RecordConversion(kind, sourceVersion, targetVersion, "success")
		h.conversionStats.RecordSuccess(time.Duration(objDuration * float64(time.Second)))

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
