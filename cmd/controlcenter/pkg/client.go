package controlcenter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client is an HTTP client for a single Orkestra runtime instance
type Client struct {
	baseURL         string
	client          *http.Client
	refreshInterval time.Duration
	mu              sync.RWMutex
}

// NewClient creates a new client
func NewClient(baseURL string, refreshInterval time.Duration, logLevel string) *Client {
	return &Client{
		baseURL:         baseURL,
		refreshInterval: refreshInterval,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchKatalog retrieves the Katalog from the instance
func (c *Client) FetchKatalog() (*KatalogResponse, error) {
	resp, err := c.client.Get(c.baseURL + "/katalog")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result KatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FetchCRDDetail retrieves detailed information for a specific CRD
func (c *Client) FetchCRDDetail(name string) (*CRDDetail, error) {
	// Fetch health and info separately
	health, err := c.fetchCRDHealth(name)
	if err != nil {
		return nil, fmt.Errorf("fetching health: %w", err)
	}

	info, err := c.fetchCRDInfo(name)
	if err != nil {
		return nil, fmt.Errorf("fetching info: %w", err)
	}

	// Merge health and info
	detail := &CRDDetail{
		Name:             info.Name,
		Description:      info.Description,
		Mode:             info.Mode,
		GVK:              info.GVK,
		GVR:              info.GVR,
		Namespaced:       info.Namespaced,
		Namespace:        info.Namespace,
		DependsOn:        info.DependsOn,
		Workers:          info.Workers,
		WorkersActive:    info.WorkersActive,
		WorkersIdle:      info.WorkersIdle,
		WorkersProcessing: info.WorkersProcessing,
		WorkerDetails:    info.WorkerDetails,
		WorkersSource:    info.WorkersSource,
		Resync:           info.Resync,
		ResyncSource:     info.ResyncSource,
		QueueDepth:       info.QueueDepth,
		MaxQueueDepth:    info.MaxQueueDepth,
		ResourceCount:    info.ResourceCount,
		TotalReconciles:  info.TotalReconciles,
		Reconciler:       info.Reconciler,
		Healthy:          health.Healthy,
		Started:          health.Started,
		Pending:          health.Pending,
		ErrorRate:        health.ErrorRate,
		Conversion:       info.Conversion,
		Admission:        info.Admission,
		State:            getState(health),
		StartedAt:        health.StartedAt,
		Uptime:           health.Uptime,
		ConsecutiveFails: health.ConsecutiveFails,
		LastError:        health.LastError,
		LastReconcile:    health.LastReconcile,
	}

	// Format times
	if health.StartedAt != "" && health.StartedAt != "not started" {
		if t, err := time.Parse(time.RFC3339, health.StartedAt); err == nil {
			duration := time.Since(t)
			if duration < time.Minute {
				detail.StartedAgo = fmt.Sprintf("%ds ago", int(duration.Seconds()))
			} else if duration < time.Hour {
				detail.StartedAgo = fmt.Sprintf("%dm ago", int(duration.Minutes()))
			} else {
				detail.StartedAgo = fmt.Sprintf("%dh ago", int(duration.Hours()))
			}
		}
	}

	if health.LastReconcile != "" && health.LastReconcile != "no reconciles yet" {
		if t, err := time.Parse(time.RFC3339, health.LastReconcile); err == nil {
			duration := time.Since(t)
			if duration < time.Minute {
				detail.LastReconcileAgo = fmt.Sprintf("%ds ago", int(duration.Seconds()))
			} else if duration < time.Hour {
				detail.LastReconcileAgo = fmt.Sprintf("%dm ago", int(duration.Minutes()))
			} else {
				detail.LastReconcileAgo = fmt.Sprintf("%dh ago", int(duration.Hours()))
			}
		}
	}

	return detail, nil
}

func (c *Client) fetchCRDHealth(name string) (*CRDHealth, error) {
	resp, err := c.client.Get(c.baseURL + "/katalog/" + name + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CRDHealth
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) fetchCRDInfo(name string) (*CRDInfo, error) {
	resp, err := c.client.Get(c.baseURL + "/katalog/" + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result CRDInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func getState(health *CRDHealth) string {
	if health.Healthy {
		return "healthy"
	}
	if health.Started {
		return "started"
	}
	if health.Pending {
		return "pending"
	}
	return "degraded"
}
