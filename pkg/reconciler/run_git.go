// pkg/reconciler/run_git.go
//
// Git hook dispatch — runs before external calls and resource groups in
// runTemplateReconcile.
//
// The Git hook performs a minimal clone/fetch of the declared repository,
// computes the current commit hash, detects whether the commit changed since
// the previous reconcile, and injects results into the resolver context via
// resolver.WithGit().
//
// Call sequence in runTemplateReconcile:
//
//  1. resolver = NewResolver(ctx, obj)
//  2. resolver = resolver.WithCross(ReadCross(...))     ← cross-CRD first
//  3. resolver, err = runGit(ctx, gvk, resolver, ...)   ← Git second
//  4. resolver, err = runExternal(ctx, ...)             ← HTTP third
//  5. resolver, err = runDocker(ctx, ...)               ← Docker fourth
//  6. runDeployments, runServices, ... etc.
//
// Results in template context:
//
//		.git.commit     → HEAD commit hash
//		.git.changed    → "true" if commit changed since last reconcile
//		.git.path       → working directory path
//		.git.error      → error message if operation failed
//		.git.called     → "true" if Git hook executed
//	 .git.succeeded	→ "true" if Git operation succeded
//
// When: conditions can gate on these:
//
//	when:
//	  - field: git.changed
//	    equals: "true"
//
// The Git hook never panics. It always returns a result map.
package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// runGit executes the declared Git hook and returns a new resolver with
// .git.* injected.
//
// Returns the enriched resolver. The original resolver is unchanged.
// Returns an error only if the Git operation fails and continueOnError=false
// (future extension — currently Git always continues).
func runGit(
	ctx context.Context,
	gvk string,
	resolver *orktmpl.Resolver,
	spec *orktypes.GitHookSpec,
) (*orktmpl.Resolver, error) {
	if spec == nil {
		return resolver, nil
	}

	log := logger.FromContext(ctx)

	// Resolve template expressions in repo, branch, and path
	repo, err := resolver.Resolve(spec.Repo)
	if err != nil {
		return resolver, fmt.Errorf("git.repo: %w", err)
	}
	branch, err := resolver.Resolve(spec.Branch)
	if err != nil {
		return resolver, fmt.Errorf("git.branch: %w", err)
	}
	path, err := resolver.Resolve(spec.Path)
	if err != nil {
		return resolver, fmt.Errorf("git.path: %w", err)
	}

	log.Debug().
		Str("repo", repo).
		Str("branch", branch).
		Str("path", path).
		Msg("git: starting operation")

		// Execute the Git operation
	start := time.Now()
	result := executeGitOperation(ctx, repo, branch, path)
	duration := time.Since(start).Seconds()

	// Record metrics
	metrics.RecordGitOperation(
		gvk,
		repo,
		result.Operation,
		result.Error,
		duration,
	)

	withError := "false"
	// Log success/failure
	if result.Error != "" {
		withError = "true"

		log.Warn().
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Str("error", result.Error).
			Msg("git: operation failed")
	} else {
		log.Debug().
			Str("repo", repo).
			Str("branch", branch).
			Str("path", path).
			Str("commit", result.Commit).
			Str("changed", result.Changed).
			Msg("git: operation succeeded")
	}

	// Inject into resolver
	gitMap := map[string]interface{}{
		"commit":    result.Commit,
		"changed":   result.Changed,
		"path":      result.Path,
		"error":     result.Error,
		"called":    "true",
		"withError": withError,
	}

	resolver = resolver.WithGit(gitMap)
	return resolver, nil
}
