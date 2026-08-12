package intake

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/orkspace/orkestra/pkg/gateway/api"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// GitLabPushEvent is the subset of GitLab's push event payload this handler
// needs. https://docs.gitlab.com/user/project/integrations/webhook_events/#push-events
type GitLabPushEvent struct {
	Ref         string `json:"ref"`
	CheckoutSHA string `json:"checkout_sha"`
	Project     struct {
		ID int `json:"id"`
	} `json:"project"`
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
	} `json:"commits"`
}

// NewGitLabHandler returns the http.HandlerFunc for one GitLab push-event
// intake entry. Same pipeline as NewGitHubHandler — verify, filter by
// branch and watch pattern, fetch and apply each matched file — against
// GitLab's request verification (a static token, not an HMAC signature)
// and Repository Files API instead of GitHub's Contents API.
func NewGitLabHandler(
	src ResolvedGitSource,
	kube kubeclient.KubeClient,
	clusters *api.ClusterRegistry,
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

		if !verifyStaticToken(src.Secret, r.Header.Get("X-Gitlab-Token")) {
			logger.FromContext(r.Context()).Warn().
				Str("entry", src.Config.Name).
				Msg("intake: gitlab webhook token verification failed")
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}

		var event GitLabPushEvent
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

		projectID := strconv.Itoa(event.Project.ID)

		results := make([]PushApplyResult, 0, len(matched))
		for _, path := range matched {
			results = append(results, applyGitLabIntentFile(r.Context(), src, kube, clusters, kat, notes, projectID, path, event.CheckoutSHA))
		}

		writeJSON(w, http.StatusOK, PushResponse{Applied: results})
	}
}

func applyGitLabIntentFile(
	ctx context.Context,
	src ResolvedGitSource,
	kube kubeclient.KubeClient,
	clusters *api.ClusterRegistry,
	kat *katalog.Katalog,
	notes orktypes.NoteRegistry,
	projectID, path, sha string,
) PushApplyResult {
	data, err := fetchGitLabFileContent(ctx, src.ContentToken, projectID, path, sha)
	if err != nil {
		logger.FromContext(ctx).Warn().
			Str("entry", src.Config.Name).Str("path", path).Err(err).
			Msg("intake: gitlab content fetch failed")
		return PushApplyResult{Path: path, Message: err.Error()}
	}

	fields, err := ParseIntentContent(path, data)
	if err != nil {
		return PushApplyResult{Path: path, Message: err.Error()}
	}

	resp, status := api.ApplyTargetFields(ctx, kube, clusters, kat, notes, src.Config.Name, fields, false)
	result := PushApplyResult{
		Path:     path,
		Target:   fieldsTarget(fields),
		Accepted: status == http.StatusOK,
		Message:  resp.Message,
	}

	if src.Config.ReportStatus {
		state := applyState(result.Accepted, true)
		if err := reportGitLabCommitStatus(ctx, src.ContentToken, projectID, sha, state, statusDescription(result)); err != nil {
			logger.FromContext(ctx).Warn().
				Str("entry", src.Config.Name).Str("path", path).Err(err).
				Msg("intake: failed to report gitlab commit status")
		}
	}

	return result
}
