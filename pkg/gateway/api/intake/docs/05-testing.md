# 05 — Local Testing

`ork webhook play` (`cmd/cli/webhook_play.go`) runs a simulated payload through the same chain a real delivery does, without a cluster, an HTTP server, or a real GitHub/GitLab/Slack account. It exists because this package's own test suite proves the *code* works — `ork webhook play` lets a platform team prove their *configuration* works, before wiring up a real delivery surface.

## What's skipped, and why that's fine

Signature/token verification never runs in play mode — there's no real secret exchange to test outside a live deployment, and the four functions in `verify.go` are already covered by `verify_test.go`. Everything downstream of verification runs for real: branch filtering, watch-pattern matching, command checking, argument parsing, and the full target-mode chain (`runCreateUpdateChain`, shared with `ork serve play` — see `cmd/cli/play_chain.go`). That downstream logic is where a misconfiguration actually lives — a typo'd `watch` pattern, a branch that doesn't match, a `serve.tokens` entry that doesn't list the webhook's name.

## `--webhook` resolves a real entry; `--source` doesn't have to

`--webhook <name>` always looks up a real `gateway.webhooks` entry from the loaded Katalog — its declared `Branch`, `Watch`, `Commands` are exercised as configured, never re-typed on the command line, so play mode can't silently drift from what's actually deployed.

`--source` is optional. `pkg/katalog`'s `LookupWebhookSource` (in `lookup.go`, alongside every other O(1) index-based lookup — `LookupByKind`, `LookupByTarget`, etc.) resolves it from `--webhook` alone, via a `webhookNameIndex` built in `BuildLookupIndexes()` the same way every other index is. This only works because entry names are unique *across all four sources*, not just within one — `validateGatewayWebhooks` (`pkg/katalog/validate_webhooks.go`) enforces that at `ork validate` time via a single `seenNames` map spanning github/gitlab/slack/generic, not four separate per-source checks. Pass `--source` explicitly to disambiguate or to be explicit; it's never required.

## Content fetch is simulated via `--fetch`, never real

GitHub/GitLab play mode can't call the real Contents/Repository Files API — there's no live account. `--fetch <matched-path>=<local-file>` supplies what that API call would have returned, once per path the watch pattern is expected to match. A matched path with no override is reported and skipped rather than failing the whole run — useful on its own, since it proves the watch pattern matched without needing a `--fetch` value ready yet.

## `--simulate` hands off exactly like `ork serve play`

`playSimulate` (`cmd/cli/play_chain.go`) is the shared handoff both commands use — `ork webhook play --simulate` and `ork serve play --simulate` both end up calling `playRunSimulate`. For GitHub/GitLab, a push matching several files runs the simulate handoff once per matched, fetched file — each is an independent CR, so each gets its own reconciliation check.

→ Back: [01-overview.md](01-overview.md)
