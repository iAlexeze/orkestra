# 03 — GitHub and GitLab

`github.go` and `gitlab.go` are near-mirrors of each other — same pipeline, different wire shapes and API conventions. Both share `watch.go` (glob matching), `content.go` (fetch + parse), and `status.go` (optional outcome reporting).

## The pipeline

```
verify signature/token
  → decode push event
  → extract branch from ref, reject if it doesn't match Branch
  → union added+modified paths across every commit (dedup, preserve order)
  → filter through Watch glob patterns (empty Watch matches everything)
  → for each matched path:
      fetch content via contentTokenRef
      parse as YAML/JSON by extension
      api.ApplyTargetFields(...)
      if ReportStatus: post outcome back as a commit/pipeline status
  → respond 200 with per-path results
```

A single push can match several files — a monorepo with `watch: ["services/*/intent.yaml"]` might see three services change in one push. Each matched path runs the full chain independently; one rejection doesn't block the others.

## `removed` is excluded on purpose

`CollectChangedFiles` (`watch.go`) unions `added` and `modified` across every commit — never `removed`. There's no content left to fetch for a deleted path, so processing it would mean designing delete semantics for a push-triggered surface, which hasn't happened yet. A deleted intent file today does nothing; it's not silently mis-handled, it's simply not in scope. See `documentation/concepts/self-service/09-webhook-intake.md` for the user-facing version of this note.

## Watch patterns use `gobwas/glob`, not `filepath.Match`

`watch.go`'s `MatchedWatchFiles` compiles each pattern with `glob.Compile(pattern, '/')` — `gobwas/glob` supports `**` for recursive matching, which Go's standard `path/filepath.Match` does not. A pattern that fails to compile is skipped rather than erroring the whole request; `ork validate` is where a malformed pattern should be caught, not a live delivery mid-flight.

`Watch` is `[]string` — plain glob strings, the same shape GitHub Actions' own `on.push.paths` filter uses. It was originally `[]WebhookWatchPattern{Path string}`, a single-field wrapper struct with no other fields and no plan to grow one; that was premature abstraction, corrected before anything shipped depending on the richer shape.

## Fetching content — why per-file, not a clone

`fetchGitHubFileContent` calls the Contents API (`GET /repos/{owner}/{repo}/contents/{path}?ref={sha}`, base64-decoded); `fetchGitLabFileContent` calls the Repository Files API (`GET /projects/{id}/repository/files/{path}/raw?ref={sha}`, raw bytes, no envelope). Both take a `ref`/`sha` from the push event so the fetch is pinned to the exact commit that triggered it, not whatever `HEAD` happens to be by the time the request completes.

ArgoCD and Flux both treat a push webhook as a pure "resync now" signal — actual content always comes from a full git clone, since an arbitrary manifest tree could touch anything. A target-mode intent file is a handful of flat key-value lines matched by explicit `watch` patterns; a per-file REST read is lighter than shipping git-clone infrastructure into the gateway for that. The tradeoff: a very broad `watch` pattern matching many files in one push means many API calls, not one clone. Not optimized for, and worth knowing before writing an overly broad pattern.

`githubAPIBaseURL`/`gitlabAPIBaseURL` (`content.go`) are package-level vars, not hardcoded inline — tests point them at an `httptest` server; production never overrides them. They also mark the current scope limit: `api.github.com` and `gitlab.com/api/v4` only. GitHub Enterprise Server's API lives at a different base path (`/api/v3`); self-hosted GitLab isn't reachable either. Neither is handled today — a documented limitation, not a silent gap.

## Status reporting is opt-in and reuses `contentTokenRef`

`GitWebhookConfig.ReportStatus` defaults to `false`. When true, `applyGitHubIntentFile`/`applyGitLabIntentFile` best-effort POST the outcome as a commit status (GitHub) or pipeline status (GitLab) via `reportGitHubCommitStatus`/`reportGitLabCommitStatus` in `status.go`, using the same `contentTokenRef` that read the file — assumed to also carry write access for this. A status-report failure is logged, never turned into a failure of the apply itself; the CR was already applied (or rejected) by the time status reporting runs.

`applyState(accepted, gitlab bool)` maps a boolean outcome to each platform's own vocabulary — GitHub wants `"success"`/`"failure"`; GitLab wants `"success"`/`"failed"`. Small, but the two platforms genuinely disagree on the string.

## Why every response is 200

GitHub and GitLab retry a delivery that returns a non-2xx status. A downstream apply rejection — denied token, failed admission rule, missing field — isn't a delivery failure worth retrying; it's a legitimate answer to "should this be applied," which the platform has no way to fix by resending the same push. Every handler here acks `200` once signature/token verification passes and reports per-path outcomes in the JSON body (`PushResponse`/`PushApplyResult` in `push.go`) instead. Only verification failures (`401`) and malformed requests (`400`/`405`) are non-2xx — the layer GitHub/GitLab's retry logic is actually meant to catch.

→ Next: [04-slack.md](04-slack.md)
