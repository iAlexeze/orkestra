// pkg/reconciler/run_git_helper.go

package reconciler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	// "github.com/ialexeze/orkestra/pkg/logger"
)

type GitOperationResult struct {
	Operation string // "clone" | "fetch"
	Commit    string
	Changed   string // "true" | "false"
	Path      string
	Error     string
}

// executeGitOperation performs a minimal Git checkout/update.
//
// Behavior:
//   - If the working directory does not exist → clone
//   - If it exists → fetch + fast-forward
//   - Reads HEAD commit hash
//   - Detects whether commit changed since last reconcile
//
// No panics. Always returns a result struct.
func executeGitOperation(ctx context.Context, repo, branch, path string) GitOperationResult {
	// log := logger.FromContext(ctx)

	// Ensure working directory exists
	if err := os.MkdirAll(path, 0o755); err != nil {
		return GitOperationResult{
			Operation: "clone",
			Error:     fmt.Sprintf("mkdir: %v", err),
			Path:      path,
		}
	}

	// Determine if repo already exists
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)

	operation := "fetch"
	if os.IsNotExist(err) {
		operation = "clone"
	}

	var cmd *exec.Cmd

	if operation == "clone" {
		cmd = exec.CommandContext(ctx, "git", "clone", "--branch", branch, repo, path)
	} else {
		cmd = exec.CommandContext(ctx, "git", "-C", path, "fetch", "origin", branch)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return GitOperationResult{
			Operation: operation,
			Error:     fmt.Sprintf("%s: %v (%s)", operation, err, out),
			Path:      path,
		}
	}

	// Fast-forward if fetch
	if operation == "fetch" {
		ff := exec.CommandContext(ctx, "git", "-C", path, "merge", "--ff-only", "origin/"+branch)
		if out, err := ff.CombinedOutput(); err != nil {
			return GitOperationResult{
				Operation: "fetch",
				Error:     fmt.Sprintf("fast-forward: %v (%s)", err, out),
				Path:      path,
			}
		}
	}

	// Read current commit
	rev := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "HEAD")
	commitBytes, err := rev.CombinedOutput()
	if err != nil {
		return GitOperationResult{
			Operation: operation,
			Error:     fmt.Sprintf("rev-parse: %v (%s)", err, commitBytes),
			Path:      path,
		}
	}

	commit := strings.TrimSpace(string(commitBytes))

	// Detect change by comparing to last commit file
	lastFile := filepath.Join(path, ".orkestra_last_commit")
	changed := "true"

	if prev, err := os.ReadFile(lastFile); err == nil {
		if strings.TrimSpace(string(prev)) == commit {
			changed = "false"
		}
	}

	// Write new commit
	_ = os.WriteFile(lastFile, []byte(commit), 0o644)

	return GitOperationResult{
		Operation: operation,
		Commit:    commit,
		Changed:   changed,
		Path:      path,
		Error:     "",
	}
}
