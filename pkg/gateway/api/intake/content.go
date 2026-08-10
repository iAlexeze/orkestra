package intake

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

// Base URLs for the GitHub and GitLab REST APIs — package-level vars so
// tests can point them at an httptest server instead of the real APIs.
// Production code never overrides these.
var (
	githubAPIBaseURL = "https://api.github.com"
	gitlabAPIBaseURL = "https://gitlab.com/api/v4"
)

// fetchGitHubFileContent reads a file at ref via the GitHub Contents API.
// token is the entry's contentTokenRef — a separate, write-scope-capable
// credential from the one that verifies the webhook itself.
//
// Only public github.com is supported. GitHub Enterprise Server hosts its
// API at a different base path (https://{host}/api/v3/...) — not handled
// here; self-hosted GitHub is a documented limitation, not a silent gap.
func fetchGitHubFileContent(ctx context.Context, token, owner, repo, path, ref string) ([]byte, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		githubAPIBaseURL, url.PathEscape(owner), url.PathEscape(repo), escapeFilePath(path), url.QueryEscape(ref))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("building GitHub contents request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %q from GitHub: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub contents API returned status %d for %q", resp.StatusCode, path)
	}

	var out struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding GitHub contents response for %q: %w", path, err)
	}
	if out.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected content encoding %q from GitHub for %q", out.Encoding, path)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(out.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decoding base64 content for %q: %w", path, err)
	}
	return decoded, nil
}

// fetchGitLabFileContent reads a file at ref via the GitLab Repository Files
// API. projectID is the numeric project ID from the push event payload —
// avoids URL-encoding ambiguity a namespace/project path would introduce.
//
// Only public gitlab.com is supported. Self-hosted GitLab instances are a
// documented limitation, not a silent gap.
func fetchGitLabFileContent(ctx context.Context, token, projectID, path, ref string) ([]byte, error) {
	u := fmt.Sprintf("%s/projects/%s/repository/files/%s/raw?ref=%s",
		gitlabAPIBaseURL, url.PathEscape(projectID), escapeFilePath(path), url.QueryEscape(ref))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("building GitLab file request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %q from GitLab: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab repository files API returned status %d for %q", resp.StatusCode, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading GitLab file content for %q: %w", path, err)
	}
	return body, nil
}

// escapeFilePath URL-escapes a repo-relative path segment by segment,
// preserving the "/" separators the Contents/Repository Files APIs expect
// between directories — url.PathEscape alone would also escape those.
func escapeFilePath(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

// ParseIntentContent decodes a fetched file's bytes into a flat field map —
// the same target-mode intent shape POST /api/v1/apply accepts. Format is
// decided by the file extension, matching ork serve play's own
// intent.yaml/intent.json convention: ".json" decodes as JSON, anything
// else as YAML.
func ParseIntentContent(path string, data []byte) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("invalid JSON in %q: %w", path, err)
		}
		return raw, nil
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid YAML in %q: %w", path, err)
	}
	return raw, nil
}
