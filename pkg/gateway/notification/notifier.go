package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Event is a dispatch-ready notification unit.
// The runtime builds one per condition+team pair after the throttle check passes.
type Event struct {
	KatalogName string                 `json:"katalogName"`
	CRName      string                 `json:"crName"`
	CRNamespace string                 `json:"crNamespace"`
	GVK         string                 `json:"gvk"`
	CondKey     string                 `json:"condKey"`
	TeamName    string                 `json:"teamName"`
	Subject     string                 `json:"subject"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Notifier dispatches a resolved, throttle-checked notification event.
// DirectNotifier handles standalone (no-gateway) deployments.
// GatewayNotifier delegates to the gateway process via HTTP.
type Notifier interface {
	Dispatch(ctx context.Context, ev Event) error
}

// GatewayNotifier POSTs events to <endpoint>/notify.
// The gateway owns SMTP/Slack credential lookup and actual dispatch.
type GatewayNotifier struct {
	endpoint string
	client   *http.Client
}

func NewGatewayNotifier(endpoint string) *GatewayNotifier {
	return &GatewayNotifier{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (g *GatewayNotifier) Dispatch(ctx context.Context, ev Event) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("notification: marshal event: %w", err)
	}
	url := g.endpoint + "/notify"
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := g.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("gateway /notify returned %d", resp.StatusCode)
	}
	return fmt.Errorf("notification: gateway dispatch failed after 3 attempts: %w", lastErr)
}
