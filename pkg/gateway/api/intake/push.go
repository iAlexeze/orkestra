package intake

import "fmt"

// PushApplyResult is the outcome of applying one matched, changed intent
// file from a single push event.
type PushApplyResult struct {
	Path     string `json:"path"`
	Target   string `json:"target,omitempty"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message,omitempty"`
}

// PushResponse is returned by the GitHub and GitLab push-event handlers.
// Message is set for a no-op response (branch not watched, nothing
// matched); Applied is set once at least one matched file was processed.
//
// Always 200 once past signature verification — see NewGitHubHandler and
// NewGitLabHandler for why a downstream apply rejection isn't itself a
// delivery failure worth retrying.
type PushResponse struct {
	Message string            `json:"message,omitempty"`
	Applied []PushApplyResult `json:"applied,omitempty"`
}

// fieldsTarget reads "target" out of a parsed intent file's fields, for
// including in the response — best-effort, empty when absent or not a string.
func fieldsTarget(fields map[string]interface{}) string {
	target, _ := fields["target"].(string)
	return target
}

// statusDescription is the short text posted back as a commit/pipeline
// status description — GitHub and GitLab both cap this field, so it stays
// to the headline outcome, not the full violation list.
func statusDescription(result PushApplyResult) string {
	if result.Accepted {
		return fmt.Sprintf("orkestra: applied %s", result.Path)
	}
	if result.Message != "" {
		return fmt.Sprintf("orkestra: rejected — %s", result.Message)
	}
	return "orkestra: rejected"
}
