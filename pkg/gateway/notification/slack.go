// pkg/notification/slack.go
//
// Slack notification dispatch via incoming webhooks.
// One HTTP POST per channel per send. Channels are notified in parallel.
//
// Message format: plain text with Markdown-like formatting that Slack renders.
// The message template is evaluated against the full resolver data map before
// dispatch so it can reference .spec.*, .metadata.*, .status.*, metrics.*, etc.
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/orkspace/orkestra/pkg/logger"
)

const slackHTTPTimeout = 5 * time.Second

// slackPayload is the JSON body sent to a Slack incoming webhook.
type slackPayload struct {
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string `json:"color"` // "good", "warning", "danger"
	Fields []struct {
		Title string `json:"title"`
		Value string `json:"value"`
		Short bool   `json:"short"`
	} `json:"fields,omitempty"`
}

// sendSlackNotification fans out a notification to multiple Slack channels
// using the given webhook URL. Channels are notified in parallel with a
// 5-second per-request timeout. Errors per channel are logged but do not
// prevent other channels from receiving the notification.
func sendSlackNotification(
	ctx context.Context,
	webhookURL string,
	channels []string,
	katalogName string,
	teamName string,
	message string,
	severity string, // "info", "warning", "danger"
) error {
	if webhookURL == "" {
		return fmt.Errorf("slack: no webhook URL configured for team %q", teamName)
	}

	color := slackColor(severity)

	payload := slackPayload{
		Text: message,
		Attachments: []slackAttachment{
			{
				Color: color,
				Fields: []struct {
					Title string `json:"title"`
					Value string `json:"value"`
					Short bool   `json:"short"`
				}{
					{Title: "Katalog", Value: katalogName, Short: true},
					{Title: "Team", Value: teamName, Short: true},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshaling payload: %w", err)
	}

	// Fan out to all channels in parallel — channels are Slack channel names
	// for routing; the actual routing is done by the webhook itself.
	// We send once per webhook (the channel list is informational for the
	// payload) unless the team has per-channel webhooks in future.
	var wg sync.WaitGroup
	var sendErr error
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := postSlack(ctx, webhookURL, body); err != nil {
			mu.Lock()
			sendErr = err
			mu.Unlock()
			logger.Warn().
				Err(err).
				Str("team", teamName).
				Strs("channels", channels).
				Msg("notification: slack send failed")
		} else {
			logger.Debug().
				Str("team", teamName).
				Strs("channels", channels).
				Msg("notification: slack sent")
		}
	}()

	wg.Wait()
	return sendErr
}

func postSlack(ctx context.Context, webhookURL string, body []byte) error {
	reqCtx, cancel := context.WithTimeout(ctx, slackHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func slackColor(severity string) string {
	switch severity {
	case "danger":
		return "danger"
	case "warning":
		return "warning"
	default:
		return "good"
	}
}
