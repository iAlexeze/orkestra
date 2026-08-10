package intake

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/orkspace/orkestra/pkg/gateway/api"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// GitHubPushEvent is the subset of GitHub's push event payload this handler
// needs. https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
type GitHubPushEvent struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
	} `json:"commits"`
}

// NewGitHubHandler returns the http.HandlerFunc for one GitHub push-event
// intake entry. Verifies the request signature, filters by branch and watch
// pattern, then for each matched changed file: fetches its content via the
// entry's contentTokenRef, parses it as a target-mode intent, and applies it
// through the same pipeline every other intake source uses.
//
// Always acks 200 once past signature verification — GitHub retries
// deliveries that return non-2xx, and a downstream apply rejection isn't a
// delivery failure. Per-file outcomes are reported in the response body.
func NewGitHubHandler(
	src ResolvedGitSource,
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
) http.HandlerFunc {
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

		if !verifyHMACSHA256(src.Secret, body, r.Header.Get("X-Hub-Signature-256")) {
			logger.FromContext(r.Context()).Warn().
				Str("entry", src.Config.Name).
				Msg("intake: github webhook signature verification failed")
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid signature")
			return
		}

		var event GitHubPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request", "body is not a valid push event")
			return
		}

		branch := strings.TrimPrefix(event.Ref, "refs/heads/")
		if branch != src.Config.Branch {
			writeJSON(w, http.StatusOK, PushResponse{Message: "branch not watched"})
			return
		}

		var groups [][]string
		for _, c := range event.Commits {
			groups = append(groups, c.Added, c.Modified)
		}
		matched := MatchedWatchFiles(src.Config.Watch, CollectChangedFiles(groups...))
		if len(matched) == 0 {
			writeJSON(w, http.StatusOK, PushResponse{Message: "no watched files changed"})
			return
		}

		owner := event.Repository.Owner.Login
		repo := event.Repository.Name

		results := make([]PushApplyResult, 0, len(matched))
		for _, path := range matched {
			results = append(results, applyGitHubIntentFile(r.Context(), src, kube, kat, notes, owner, repo, path, event.After))
		}

		writeJSON(w, http.StatusOK, PushResponse{Applied: results})
	}
}

func applyGitHubIntentFile(
	ctx context.Context,
	src ResolvedGitSource,
	kube kubeclient.KubeClient,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
	owner, repo, path, sha string,
) PushApplyResult {
	data, err := fetchGitHubFileContent(ctx, src.ContentToken, owner, repo, path, sha)
	if err != nil {
		logger.FromContext(ctx).Warn().
			Str("entry", src.Config.Name).Str("path", path).Err(err).
			Msg("intake: github content fetch failed")
		return PushApplyResult{Path: path, Message: err.Error()}
	}

	fields, err := ParseIntentContent(path, data)
	if err != nil {
		return PushApplyResult{Path: path, Message: err.Error()}
	}

	resp, status := api.ApplyTargetFields(ctx, kube, kat, notes, src.Config.Name, fields, false)
	result := PushApplyResult{
		Path:     path,
		Target:   fieldsTarget(fields),
		Accepted: status == http.StatusOK,
		Message:  resp.Message,
	}

	if src.Config.ReportStatus {
		state := applyState(result.Accepted, false)
		if err := reportGitHubCommitStatus(ctx, src.ContentToken, owner, repo, sha, state, statusDescription(result)); err != nil {
			logger.FromContext(ctx).Warn().
				Str("entry", src.Config.Name).Str("path", path).Err(err).
				Msg("intake: failed to report github commit status")
		}
	}

	return result
}
