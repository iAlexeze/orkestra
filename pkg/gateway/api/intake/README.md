# pkg/gateway/api/intake

`intake` implements `gateway.webhooks` — four inbound, push-based delivery sources (GitHub, GitLab, Slack, generic JSON) that resolve through the exact same target-mode pipeline `POST /api/v1/apply` does. Where `pkg/gateway/api` is a REST surface a caller *pulls* against, `intake` is a set of listeners a caller's own platform *pushes* into.

```
POST /webhooks/github/{entry}    ← GitHub push event, X-Hub-Signature-256
POST /webhooks/gitlab/{entry}    ← GitLab push event, X-Gitlab-Token
POST /webhooks/slack             ← Slack slash command, X-Slack-Signature
POST /webhooks/generic/{entry}   ← any JSON caller, X-Signature-256
```

Every handler ends at the same call: `api.ApplyTargetFields(ctx, kube, kat, notes, tokenName, fields, dryRun)`. The source's only job is verifying the request is genuine and turning its payload into a flat field map. Past that point there is no "webhook" concept left in the code path — it's the identical `BuildCRFromTarget` → provenance stamping → `serve.tokens` check → SSA patch pipeline a direct API caller gets.

## Enabling

Off by default. Requires `gateway.api.enabled: true` — `ork validate` rejects `gateway.webhooks` without it:

```yaml
gateway:
  api:
    enabled: true
  webhooks:
    github:
      - name: payments-repo
        enabled: true
        path: /webhooks/github/payments
        branch: main
        watch:
          - "services/*/intent.yaml"
        secretRef: { name: ork-payments-github-secret, key: secret }
        contentTokenRef: { name: ork-payments-github-app-token, key: token }
```

## What each source can do

| Source | Handler | Verifies via | Payload → fields |
|--------|---------|---------------|-------------------|
| GitHub | `NewGitHubHandler` (`github.go`) | `X-Hub-Signature-256` (HMAC) | fetch matched file via Contents API, parse as YAML/JSON |
| GitLab | `NewGitLabHandler` (`gitlab.go`) | `X-Gitlab-Token` (static) | fetch matched file via Repository Files API, parse as YAML/JSON |
| Slack | `NewSlackHandler` (`slack.go`) | `X-Slack-Signature` (HMAC, ±5min) | parse `"<target> key=value ..."` from slash command text |
| Generic | `NewGenericHandler` (`generic.go`) | `X-Signature-256` (HMAC) | body *is* the field map |

Registration and credential lifecycle live in `server.go` (`Server`, `NewIntakeServer`) — a separate type from `api.APIServer`, composed by the caller (`cmd/internal/gateway.go`) rather than nested, since `intake` already imports `api` and the reverse would cycle. See [docs/01-overview.md](docs/01-overview.md#why-a-separate-package).

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand why intake is a separate package, and the push/pull distinction | [docs/01-overview.md](docs/01-overview.md) |
| Understand the four verification schemes and why each differs | [docs/02-verification.md](docs/02-verification.md) |
| Understand GitHub/GitLab: push parsing, watch matching, content fetch, status reporting | [docs/03-git-sources.md](docs/03-git-sources.md) |
| Understand Slack: ack timing, the worker pool, background apply | [docs/04-slack.md](docs/04-slack.md) |
| Understand local testing — `ork webhook play` internals | [docs/05-testing.md](docs/05-testing.md) |
