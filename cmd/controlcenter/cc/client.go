package controlcenter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for one Orkestra runtime instance.
// All methods are safe for concurrent use.
type Client struct {
	baseURL    string
	httpClient *http.Client // field name used throughout — do not rename
}

// NewClient creates a Client targeting the given base URL.
func NewClient(baseURL string, _ time.Duration, _ string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Katalog endpoints
// ─────────────────────────────────────────────────────────────────────────────

// FetchKatalog calls GET /katalog.
func (c *Client) FetchKatalog() (*KatalogResponse, error) {
	return getJSON[KatalogResponse](c, "/katalog")
}

// ─────────────────────────────────────────────────────────────────────────────
// CRD endpoints
// ─────────────────────────────────────────────────────────────────────────────

// FetchCRDDetail fetches health + info for a single CRD and merges them.
func (c *Client) FetchCRDDetail(name string) (*CRDDetail, error) {
	health, err := getJSON[CRDHealth](c, "/katalog/"+name+"/health")
	if err != nil {
		return nil, fmt.Errorf("fetching health: %w", err)
	}

	info, err := getJSON[CRDInfo](c, "/katalog/"+name)
	if err != nil {
		return nil, fmt.Errorf("fetching info: %w", err)
	}

	detail := &CRDDetail{
		Name:                     info.Name,
		Description:              info.Description,
		Mode:                     info.Mode,
		GVK:                      info.GVK,
		GVR:                      info.GVR,
		Namespaced:               info.Namespaced,
		Namespace:                info.Namespace,
		DependsOn:                info.DependsOn,
		HasUnhealthyDependencies: health.HasUnhealthyDependencies,
		Dependencies:             health.Dependencies,
		Workers:                  info.Workers,
		WorkersActive:            int(info.WorkersActive),
		WorkersIdle:              int(info.WorkersIdle),
		WorkersProcessing:        int(info.WorkersProcessing),
		WorkerDetails:            info.WorkerDetails,
		WorkersSource:            info.WorkersSource,
		Resync:                   info.Resync,
		ResyncSource:             info.ResyncSource,
		QueueDepth:               info.QueueDepth,
		MaxQueueDepth:            info.MaxQueueDepth,
		ResourceCount:            info.ResourceCount,
		TotalReconciles:          info.TotalReconciles,
		Reconciler:               info.Reconciler,
		Healthy:                  health.Healthy,
		Started:                  health.Started,
		Pending:                  health.Pending,
		ErrorRate:                health.ErrorRate,
		Conversion:               info.Conversion,
		Admission:                info.Admission,
		State:                    health.State,
		StartedAt:                health.StartedAt,
		Uptime:                   health.Uptime,
		ConsecutiveFails:         health.ConsecutiveFails,
		LastError:                health.LastError,
		LastReconcile:            health.LastReconcile,
		RBAC:                     info.RBAC,
		RBACCount:                info.RBAC.TotalRules,
	}

	detail.StartedAgo = humanDuration(health.StartedAt)
	detail.LastReconcileAgo = humanDuration(health.LastReconcile)

	return detail, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CR endpoints
// ─────────────────────────────────────────────────────────────────────────────

// FetchCRList calls GET /katalog/{crd}/cr.
func (c *Client) FetchCRList(instanceURL, crdName string) (*CRListResponse, error) {
	url := fmt.Sprintf("%s/katalog/%s/cr", instanceURL, strings.ToLower(crdName))
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching CR list for %s: %w", crdName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return &CRListResponse{CRD: crdName, Items: []CRSummary{}}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CR list: HTTP %d for %s", resp.StatusCode, crdName)
	}

	var result CRListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding CR list for %s: %w", crdName, err)
	}
	return &result, nil
}

// FetchCRDetail calls GET /katalog/{crd}/cr[/{namespace}]/{name}.
func (c *Client) FetchCRDetail(instanceURL, crdName, namespace, name string) (*CRDetailResponse, error) {
	var path string
	if namespace != "" {
		path = fmt.Sprintf("%s/katalog/%s/cr/%s/%s", instanceURL, strings.ToLower(crdName), namespace, name)
	} else {
		path = fmt.Sprintf("%s/katalog/%s/cr/%s", instanceURL, strings.ToLower(crdName), name)
	}

	resp, err := c.httpClient.Get(path)
	if err != nil {
		return nil, fmt.Errorf("fetching CR detail %s/%s: %w", namespace, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("CR %q not found in namespace %q", name, namespace)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CR detail: HTTP %d", resp.StatusCode)
	}

	var result CRDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding CR detail: %w", err)
	}
	return &result, nil
}

// FetchCREvents calls GET /katalog/{crd}/cr[/{namespace}]/{name}/events.
// Best-effort — returns empty on any error.
func (c *Client) FetchCREvents(instanceURL, crdName, namespace, name string) (*CREventsResponse, error) {
	var path string
	if namespace != "" {
		path = fmt.Sprintf("%s/katalog/%s/cr/%s/%s/events", instanceURL, strings.ToLower(crdName), namespace, name)
	} else {
		path = fmt.Sprintf("%s/katalog/%s/cr/%s/events", instanceURL, strings.ToLower(crdName), name)
	}

	resp, err := c.httpClient.Get(path)
	if err != nil {
		return &CREventsResponse{Name: name, Events: []CREvent{}}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &CREventsResponse{Name: name, Events: []CREvent{}}, nil
	}

	var result CREventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &CREventsResponse{Name: name, Events: []CREvent{}}, nil
	}
	return &result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// Health Checker
func (c *Client) CheckHealth() error {
	url := strings.TrimSuffix(c.baseURL, "/") + "/health"

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// getJSON is a generic GET → JSON decode helper.
func getJSON[T any](c *Client, path string) (*T, error) {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return nil, fmt.Errorf("HTTP %d from %s%s", resp.StatusCode, c.baseURL, path)
	}

	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", path, err)
	}
	return &v, nil
}

func humanDuration(rfc3339 string) string {
	if rfc3339 == "" || rfc3339 == "not started" || rfc3339 == "no reconciles yet" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}
