package intake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackClient posts a message back to Slack after an async apply completes.
// An interface so the handler is testable without reaching the real Slack API.
type SlackClient interface {
	PostMessage(responseURL, text string) error
}

// HTTPSlackClient posts to a Slack response_url — the standard mechanism
// for delayed slash-command responses. responseURL is single-use and
// expires a few minutes after the original command, per Slack's own limits.
type HTTPSlackClient struct {
	Client *http.Client
}

// NewHTTPSlackClient returns a client with a sane default timeout.
func NewHTTPSlackClient() *HTTPSlackClient {
	return &HTTPSlackClient{Client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *HTTPSlackClient) PostMessage(responseURL, text string) error {
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshaling slack message: %w", err)
	}

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodPost, responseURL, bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("building slack response request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("posting to slack response_url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack response_url returned status %d", resp.StatusCode)
	}
	return nil
}
