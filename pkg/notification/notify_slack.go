package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/orkspace/orkestra/pkg/types"
)

type slackPayload struct {
	Text string `json:"text"`
}

func sendSlackNotification(
	_ context.Context,
	webhook string,
	channels []string,
	operatorName, teamName string,
	cond types.Condition,
	data map[string]interface{},
) error {
	text := formatSlackMessage(operatorName, teamName, cond, data)
	payload := slackPayload{Text: text}

	b, _ := json.Marshal(payload)

	for range channels {
		// For now, we ignore per-channel routing and just post to the webhook.
		// TODO: extend this later with blocks, attachments, etc.
		_, _ = http.Post(webhook, "application/json", bytes.NewReader(b))
	}

	return nil
}

func formatSlackMessage(operatorName, teamName string, cond types.Condition, _ map[string]interface{}) string {
	return "*Orkestra notification*\n" +
		"*Team:* " + teamName + "\n" +
		"*Operator:* " + operatorName + "\n" +
		"*Field:* `" + cond.Field + "`\n" +
		"*Notify:* `" + strings.Join(cond.Notify, ",") + "`\n" +
		"*Status:* condition evaluated to *true*."
}
