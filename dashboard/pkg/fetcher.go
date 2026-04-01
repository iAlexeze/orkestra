package dashboard

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 5 * time.Second},
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Katalog Response (from /katalog)
// ─────────────────────────────────────────────────────────────────────────────

type KatalogResponse struct {
    Total        int                `json:"total"`
    TotalEnabled int                `json:"totalEnabled"`
    Healthy      bool               `json:"healthy"`
    Status       int                `json:"status"`
    CRDs         []CRDSummary       `json:"crds"`
    Name         string             `json:"name,omitempty"`
    Version      string             `json:"version,omitempty"`
    Author       string             `json:"author,omitempty"`
    Description  string             `json:"description,omitempty"`
    License      string             `json:"license,omitempty"`
}

type CRDSummary struct {
    Name            string  `json:"name"`
    Healthy         bool    `json:"healthy"`
    Started         bool    `json:"started"`
    Workers         int     `json:"workers"`
    WorkersActive   int     `json:"workersActive"`
    WorkersSource   string  `json:"workersSource"`
    QueueDepth      int     `json:"queueDepth"`
    MaxQueueDepth   int     `json:"maxQueueDepth"`
    ResourceCount   int     `json:"resourceCount"`
    ErrorRate       float64 `json:"errorRate"`
    Uptime          string  `json:"uptime"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CRD Health (from /katalog/{crd}/health)
// ─────────────────────────────────────────────────────────────────────────────

type CRDHealth struct {
    Name            string  `json:"name"`
    State           string  `json:"state"`           // "healthy", "degraded", "pending"
    Healthy         bool    `json:"healthy"`
    Started         bool    `json:"started"`
    StartedAt       string  `json:"startedAt"`
    Uptime          string  `json:"uptime"`
    QueueDepth      int     `json:"queueDepth"`
    ErrorRate       float64 `json:"errorRate"`
    ConsecutiveFails int    `json:"consecutiveFails"`
    TotalReconciles int     `json:"totalReconciles"`
    ResourceCount   int     `json:"resourceCount"`
    LastError       string  `json:"lastError"`
    LastReconcile   string  `json:"lastReconcile"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CRD Info (from /katalog/{crd})
// ─────────────────────────────────────────────────────────────────────────────

type CRDInfo struct {
    Name                string                 `json:"name"`
    Description         string                 `json:"description"`
    Mode                string                 `json:"mode"`
    GVK                 string                 `json:"gvk"`
    GVR                 string                 `json:"gvr"`
    Namespaced          bool                   `json:"namespaced"`
    Namespace           string                 `json:"namespace"`
    DependsOn           []string               `json:"dependsOn"`
    Workers             int                    `json:"workers"`
    WorkersActive       int                    `json:"workersActive"`
    WorkersSource       string                 `json:"workersSource"`
    Resync              string                 `json:"resync"`
    ResyncSource        string                 `json:"resyncSource"`
    QueueDepth          int                    `json:"queueDepth"`
    MaxQueueDepth       int                    `json:"maxQueueDepth"`
    MaxQueueDepthSource string                 `json:"maxQueueDepthSource"`
    ResourceCount       int                    `json:"resourceCount"`
    TotalReconciles     int                    `json:"totalReconciles"`
    Reconciler          map[string]interface{} `json:"reconciler"`
    Healthy             bool                   `json:"healthy"`
    Started             bool                   `json:"started"`
    ErrorRate           float64                `json:"errorRate"`
    Conversion          *ConversionStats       `json:"conversion"`
    Admission           *AdmissionStats        `json:"admission"`
}

type ConversionStats struct {
    Enabled     bool    `json:"enabled"`
    Total       int     `json:"total"`
    Success     int     `json:"success"`
    Failures    int     `json:"failures"`
    AvgLatencyMs float64 `json:"avgLatencyMs"`
    P95LatencyMs float64 `json:"p95LatencyMs"`
}

type AdmissionStats struct {
    WebhooksEnabled        bool    `json:"webhooksEnabled"`
    ValidationTotal        int     `json:"validationTotal"`
    ValidationAllowed      int     `json:"validationAllowed"`
    ValidationDenied       int     `json:"validationDenied"`
    ValidationWarned       int     `json:"validationWarned"`
    ValAvgLatencyMs        float64 `json:"valAvgLatencyMs"`
    ValP95LatencyMs        float64 `json:"valP95LatencyMs"`
    ValMaxLatencyMs        float64 `json:"valMaxLatencyMs"`
    MutationTotal          int     `json:"mutationTotal"`
    MutationApplied        int     `json:"mutationApplied"`
    MutationSkipped        int     `json:"mutationSkipped"`
    MutAvgLatencyMs        float64 `json:"mutAvgLatencyMs"`
    MutP95LatencyMs        float64 `json:"mutP95LatencyMs"`
    MutMaxLatencyMs        float64 `json:"mutMaxLatencyMs"`
}

// ─────────────────────────────────────────────────────────────────────────────
// CRD Detail combines info + health
// ─────────────────────────────────────────────────────────────────────────────

type CRDDetail struct {
    // From health endpoint
    State           string  `json:"state"`
    StartedAt       string  `json:"startedAt"`
    Uptime          string  `json:"uptime"`
    ConsecutiveFails int    `json:"consecutiveFails"`
    LastError       string  `json:"lastError"`
    LastReconcile   string  `json:"lastReconcile"`
    
 
    // Human readable
    StartedAgo       string `json:"startedAgo"`
    LastReconcileAgo string `json:"lastReconcileAgo"`

    // From info endpoint
    Name                string                 `json:"name"`
    Description         string                 `json:"description"`
    Mode                string                 `json:"mode"`
    GVK                 string                 `json:"gvk"`
    GVR                 string                 `json:"gvr"`
    Namespaced          bool                   `json:"namespaced"`
    Namespace           string                 `json:"namespace"`
    DependsOn           []string               `json:"dependsOn"`
    Workers             int                    `json:"workers"`
    WorkersActive       int                    `json:"workersActive"`
    WorkersSource       string                 `json:"workersSource"`
    Resync              string                 `json:"resync"`
    ResyncSource        string                 `json:"resyncSource"`
    QueueDepth          int                    `json:"queueDepth"`
    MaxQueueDepth       int                    `json:"maxQueueDepth"`
    MaxQueueDepthSource string                 `json:"maxQueueDepthSource"`
    ResourceCount       int                    `json:"resourceCount"`
    TotalReconciles     int                    `json:"totalReconciles"`
    Reconciler          map[string]interface{} `json:"reconciler"`
    Healthy             bool                   `json:"healthy"`
    Started             bool                   `json:"started"`
    ErrorRate           float64                `json:"errorRate"`
    Conversion          *ConversionStats       `json:"conversion"`
    Admission           *AdmissionStats        `json:"admission"`
}

// ─────────────────────────────────────────────────────────────────────────────
// API Methods
// ─────────────────────────────────────────────────────────────────────────────

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

func (c *Client) FetchCRDHealth(name string) (*CRDHealth, error) {
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

func (c *Client) FetchCRDInfo(name string) (*CRDInfo, error) {
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

func (c *Client) FetchCRDDetail(name string) (*CRDDetail, error) {
    health, err := c.FetchCRDHealth(name)
    if err != nil {
        return nil, fmt.Errorf("fetching health: %w", err)
    }

    info, err := c.FetchCRDInfo(name)
    if err != nil {
        return nil, fmt.Errorf("fetching info: %w", err)
    }

    // Format times for frontend
    startedAt := parseTime(health.StartedAt)
    lastReconcile := parseTime(health.LastReconcile)

    // Merge health into info
    detail := &CRDDetail{
        // From health
        State:            health.State,
        StartedAt:        health.StartedAt,
        Uptime:           health.Uptime,
        ConsecutiveFails: health.ConsecutiveFails,
        LastError:        health.LastError,
        LastReconcile:    health.LastReconcile,

        // Human-friendly
        StartedAgo:       timeAgo(startedAt),
        LastReconcileAgo: timeAgo(lastReconcile),

        // From info
        Name:                info.Name,
        Description:         info.Description,
        Mode:                info.Mode,
        GVK:                 info.GVK,
        GVR:                 info.GVR,
        Namespaced:          info.Namespaced,
        Namespace:           info.Namespace,
        DependsOn:           info.DependsOn,
        Workers:             info.Workers,
        WorkersActive:       info.WorkersActive,
        WorkersSource:       info.WorkersSource,
        Resync:              info.Resync,
        ResyncSource:        info.ResyncSource,
        QueueDepth:          info.QueueDepth,
        MaxQueueDepth:       info.MaxQueueDepth,
        MaxQueueDepthSource: info.MaxQueueDepthSource,
        ResourceCount:       info.ResourceCount,
        TotalReconciles:     info.TotalReconciles,
        Reconciler:          info.Reconciler,
        Healthy:             info.Healthy,
        Started:             info.Started,
        ErrorRate:           info.ErrorRate,
        Conversion:          info.Conversion,
        Admission:           info.Admission,
    }

    return detail, nil
}