package intake

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/gateway/api"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// slackWorkerPoolSize bounds concurrent background applies per Slack entry.
// Slack retries a slash command on a slow ack, so a burst must not translate
// into unbounded concurrent applies against the cluster.
const slackWorkerPoolSize = 8

// slackApplyTimeout bounds how long a single background apply may run,
// independent of the original HTTP request's lifecycle (which ends the
// moment the ack is written — using that request's context for work that
// continues after the handler returns would get it canceled immediately).
const slackApplyTimeout = 30 * time.Second

// NewSlackHandler returns the http.HandlerFunc for one Slack slash-command
// intake entry. Verifies the request signature, parses
// "<target> key=value ..." into a flat field map, acknowledges immediately
// (Slack requires a response within 3s), then applies on a bounded worker
// pool and posts the outcome back to response_url — the same
// ApplyTargetFields pipeline every other intake source and a direct
// POST /api/v1/apply call use.
func NewSlackHandler(
	src ResolvedSlackSource,
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
	slack SlackClient,
) http.HandlerFunc {
	pool := newWorkerPool(slackWorkerPoolSize)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed", "only POST requests are supported")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxIntakeBodyBytes))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request", "failed to read request body")
			return
		}

		timestamp := r.Header.Get("X-Slack-Request-Timestamp")
		signature := r.Header.Get("X-Slack-Signature")
		if !verifySlackSignature(src.SigningSecret, []byte(timestamp), body, signature, time.Now()) {
			logger.FromContext(r.Context()).Warn().
				Str("entry", src.Config.Name).
				Msg("intake: slack signature verification failed")
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid signature")
			return
		}

		// Slack sends application/x-www-form-urlencoded — parsed from the
		// bytes already read above, not from r.Body a second time (it's
		// drained, and re-reading it is what r.ParseForm() would try to do).
		values, err := url.ParseQuery(string(body))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request", "body is not valid form data")
			return
		}

		command := values.Get("command")
		text := values.Get("text")
		responseURL := values.Get("response_url")

		if !commandAllowed(src.Config.Commands, command) {
			writeSlackAck(w, fmt.Sprintf("Unknown command %q. Try: %s", command, strings.Join(src.Config.Commands, ", ")))
			return
		}

		fields, err := ParseSlackArgs(text)
		if err != nil {
			writeSlackAck(w, fmt.Sprintf("Invalid arguments: %v", err))
			return
		}

		target, _ := fields["target"].(string)
		if kat.LookupByTarget(target) == nil {
			writeSlackAck(w, fmt.Sprintf("Unknown target %q. Available: %s", target, strings.Join(kat.AvailableTargets(), ", ")))
			return
		}

		// Acknowledge immediately — Slack requires a response within 3s.
		writeSlackAck(w, fmt.Sprintf("Deploying %s... I'll update you when it's ready.", target))

		// Apply in the background, on a fresh context — r.Context() is
		// canceled the moment this handler returns, which is right after
		// the ack above.
		pool.Submit(context.Background(), func(_ context.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), slackApplyTimeout)
			defer cancel()

			resp, status := api.ApplyTargetFields(ctx, kube, kat, notes, src.Config.Name, fields, false)
			if status != http.StatusOK {
				if err := slack.PostMessage(responseURL, fmt.Sprintf(":x: %s", resp.Message)); err != nil {
					logger.FromContext(ctx).Warn().
						Str("entry", src.Config.Name).
						Err(err).
						Msg("intake: failed to post slack failure message")
				}
				return
			}
			if err := slack.PostMessage(responseURL, fmt.Sprintf(":rocket: Deployed. Poll: %s", resp.PollURL)); err != nil {
				logger.FromContext(ctx).Warn().
					Str("entry", src.Config.Name).
					Err(err).
					Msg("intake: failed to post slack success message")
			}
		})
	}
}

func commandAllowed(commands []string, command string) bool {
	for _, c := range commands {
		if c == command {
			return true
		}
	}
	return false
}

// writeSlackAck writes an immediate, synchronous Slack message response —
// the 200 OK a slash command needs within 3 seconds.
func writeSlackAck(w http.ResponseWriter, text string) {
	writeJSON(w, http.StatusOK, map[string]string{"text": text})
}
