package intake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// statusContext is the fixed context/name every reported status uses —
// lets a caller distinguish Orkestra's status from CI's on the same commit.
const statusContext = "orkestra/gateway"

// reportGitHubCommitStatus posts the apply outcome as a commit status.
// token is the entry's contentTokenRef — see GitWebhookConfig.ReportStatus.
// Best-effort: a failure here is logged by the caller, never lets a failed
// status post turn a successful apply into a rejected webhook delivery.
func reportGitHubCommitStatus(ctx context.Context, token, owner, repo, sha, state, description string) error {
	u := fmt.Sprintf("%s/repos/%s/%s/statuses/%s",
		githubAPIBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(sha))

	body, err := json.Marshal(map[string]string{
		"state":       state,
		"description": description,
		"context":     statusContext,
	})
	if err != nil {
		return fmt.Errorf("marshaling GitHub status body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building GitHub status request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting GitHub commit status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub statuses API returned status %d", resp.StatusCode)
	}
	return nil
}

// reportGitLabCommitStatus posts the apply outcome as a pipeline/commit
// status. state is one of GitLab's accepted values: "pending", "running",
// "success", "failed", "canceled".
func reportGitLabCommitStatus(ctx context.Context, token, projectID, sha, state, description string) error {
	u := fmt.Sprintf("%s/projects/%s/statuses/%s?state=%s&name=%s&description=%s",
		gitlabAPIBaseURL, url.PathEscape(projectID), url.PathEscape(sha),
		url.QueryEscape(state), url.QueryEscape(statusContext), url.QueryEscape(description),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return fmt.Errorf("building GitLab status request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting GitLab commit status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitLab statuses API returned status %d", resp.StatusCode)
	}
	return nil
}

// applyState maps an apply outcome to each source's status vocabulary.
// GitHub: "success" | "failure". GitLab: "success" | "failed".
func applyState(accepted bool, gitlab bool) string {
	if accepted {
		return "success"
	}
	if gitlab {
		return "failed"
	}
	return "failure"
}
