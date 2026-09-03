# 02 — Verification

## Four schemes, implemented in `verify.go`

No shared middleware verifies every request the way `api`'s `auth()` wrapper does for Bearer tokens — the four sources don't share a scheme, so each handler calls its own verification function directly, at the top of its own `ServeHTTP`/closure body, before touching the payload.

| Source | Function | Header | Mechanism |
|--------|----------|--------|-----------|
| GitHub | `verifyHMACSHA256` | `X-Hub-Signature-256` | HMAC-SHA256 over the raw body, `sha256=<hex>` |
| Generic | `verifyHMACSHA256` | `X-Signature-256` | same function, GitHub's convention reused so callers don't learn a second scheme |
| GitLab | `verifyStaticToken` | `X-Gitlab-Token` | constant-time string comparison — GitLab's webhook auth is a shared secret, not a signature |
| Slack | `verifySlackSignature` | `X-Slack-Signature` | HMAC-SHA256 over `v0:{timestamp}:{body}`, ±5 minute replay window |

## Constant-time, always

Every comparison — `subtle.ConstantTimeCompare` in all three functions — never short-circuits on the first mismatched byte. A naive `==` string comparison leaks timing information proportional to how many leading bytes matched, which is enough for a patient attacker to recover a signature or token byte by byte. This applies uniformly, including to `verifyStaticToken`'s GitLab comparison, which might look low-stakes enough to skip — it isn't.

## Why not OIDC

`gateway.api.auth.tokens` supports `githubOIDC`/`gitlabOIDC` — a CI job requests a short-lived JWT from its own provider and presents it as a Bearer token. That's not available here, and not because it's unimplemented: GitHub, GitLab, and Slack's own webhook *delivery* infrastructure doesn't support OIDC as a transport. They sign a body or send a static token, full stop. OIDC authenticates the opposite direction — a caller proving its own identity to the gateway — not a platform proving a delivery is genuine.

## Slack's replay window

`verifySlackSignature` takes `now time.Time` as an explicit parameter rather than reading the wall clock internally:

```go
func verifySlackSignature(secret string, timestamp, body []byte, signature string, now time.Time) bool
```

This is the only verification function with a time-based check, and passing `now` in lets tests exercise the ±5 minute boundary deterministically instead of racing real time. Every other verification function is pure — same inputs, same output, no clock involved.

## Two-credential split for GitHub and GitLab

`secretRef` (verified here) and `contentTokenRef` (used in [03-git-sources.md](03-git-sources.md) to fetch file content) are deliberately separate credentials, checked at different points in the pipeline. A leaked `secretRef` lets an attacker forge webhook *deliveries* — POST a fake push event that passes signature verification. It grants no read access to the repository. `contentTokenRef` is the only credential that can actually see inside the repo, and it's never involved in verifying that a request is genuine.

→ Next: [03-git-sources.md](03-git-sources.md)
