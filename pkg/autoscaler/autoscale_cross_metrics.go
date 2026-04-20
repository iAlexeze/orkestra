// pkg/autoscaler/autoscale_cross_metrics.go
//
// CrossMetricsRegistry — a shared registry of AutoMetrics instances, one per
// operatorbox. Enables autoscale conditions to reference another operatorbox's
// live metrics via the cross.metrics.* namespace.
//
// Usage in Katalog:
//
//	# The database-backed-app operatorbox scales based on the database
//	# operatorbox's queue depth — if the DB is overwhelmed, slow down.
//	operatorBox:
//	  autoscale:
//	    conditions:
//	      when:
//	        - field: cross.managed-database.metrics.queueDepth
//	          greaterThan: "500"
//	    do:
//	      workers: 2   # back off while DB is under load
//
// Resolution:
//
//	cross.<crd>.metrics.<field>
//	  → CrossMetricsRegistry.Get(<crd>).Get("metrics.<field>")
//
// This is read-only. An operatorbox can observe another's metrics but cannot
// modify them. The same principle as cross-CRD CR state observation.
//
// Thread safety: Register is called once at startup per operatorbox (under
// Kordinator init). Get is called on every autoscale tick (concurrent reads).
// sync.Map provides safe concurrent access with no lock contention on reads.
package autoscaler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// CrossMetricsRegistry holds AutoMetrics for all registered operatorboxes.
// Keyed by lowercase CRD crd name — the same key used in cross: declarations.
type CrossMetricsRegistry struct {
	m sync.Map // map[string]*AutoMetrics
}

// GlobalCrossMetricsRegistry is the process-wide registry.
// Populated during Kordinator startup, read by autoscalers at runtime.
var GlobalCrossMetricsRegistry = &CrossMetricsRegistry{}

// Register registers an operatorbox's AutoMetrics under its crd name.
// Called once per operatorbox during Kordinator startup.
// crd is normalized to lowercase for consistent lookup.
func (r *CrossMetricsRegistry) Register(crd string, metrics *AutoMetrics) {
	if crd == "" || metrics == nil {
		return
	}
	r.m.Store(strings.ToLower(crd), metrics)
}

// Get returns the AutoMetrics for the given crd, or nil if not registered.
// crd is normalized to lowercase — "ManagedDatabase" and "manageddatabase" are equivalent.
func (r *CrossMetricsRegistry) Get(crd string) *AutoMetrics {
	v, ok := r.m.Load(strings.ToLower(crd))
	if !ok {
		return nil
	}
	return v.(*AutoMetrics)
}

// ResolveCrossMetric resolves a cross.metrics field path of the form:
//
//	cross.<crd>.metrics.<field>
//
// Resolution order:
//  1. GlobalCrossMetricsRegistry — zero hops, same-binary CRDs
//  2. source.Endpoint HTTP call  — one hop, cross-binary CRDs
//     The endpoint must be the remote operator's /katalog/{crd} URL.
//     The response is expected to carry a "metrics" object with the same
//     field names as AutoMetrics.AsMap().
//
// Returns "" when neither path finds a value.
func ResolveCrossMetric(registry *CrossMetricsRegistry, field string, source *orktypes.CrossSource) string {
	// Expected: cross.<crd>.metrics.<metric>
	// e.g.     cross.managed-database.metrics.queueDepth
	if registry == nil && source == nil {
		return ""
	}

	stripped := strings.TrimPrefix(field, "cross.")
	dotIdx := strings.Index(stripped, ".metrics.")
	if dotIdx < 0 {
		return ""
	}

	crd := stripped[:dotIdx]
	metricName := stripped[dotIdx+len(".metrics."):]
	metricField := "metrics." + metricName

	// Path 1: in-process registry
	if registry != nil {
		if m := registry.Get(crd); m != nil {
			return m.Get(metricField)
		}
	}

	// Path 2: HTTP endpoint fallback (cross-binary)
	if source != nil && source.Endpoint != "" {
		return fetchCrossMetricHTTP(source.Endpoint, source.Token, metricName)
	}

	return ""
}

const crossMetricHTTPTimeout = 5 * time.Second

// fetchCrossMetricHTTP calls the remote operator's /katalog/{crd} endpoint and
// extracts the named metric from the "metrics" key in the JSON response.
// This mirrors how readCross uses source.endpoint for CR observation.
func fetchCrossMetricHTTP(endpoint, token, metricName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), crossMetricHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}

	// Parse top-level "metrics" key from the /katalog/{crd} response.
	var response struct {
		Metrics map[string]interface{} `json:"metrics"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return ""
	}

	v, ok := response.Metrics[metricName]
	if !ok {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strings.TrimRight(strings.TrimRight(
			strings.Replace(fmt.Sprintf("%.4f", val), ".", ".", 1),
			"0"), ".")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// IsCrossMetricField returns true when the field path refers to another
// operatorbox's runtime metrics via the cross.*.metrics.* namespace.
func IsCrossMetricField(field string) bool {
	return strings.HasPrefix(field, "cross.") && strings.Contains(field, ".metrics.")
}
