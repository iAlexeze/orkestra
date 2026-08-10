# ork webhook

Inspect and locally test `gateway.webhooks` entries — no cluster required.

```bash
ork webhook list
ork webhook play --webhook pagerduty --body body.json
```

`--file` / `-f` defaults to `katalog.yaml` in both subcommands.

---

## ork webhook list

Print every entry declared under `gateway.webhooks`, across all four sources, as a table.

```bash
ork webhook list
ork webhook list -f my-katalog.yaml
```

**Output**

```text
SOURCE   NAME                  PATH                         ENABLED
generic  pagerduty             /webhooks/generic/pagerduty  true
github   payments-repo         /webhooks/github/payments    true   branch=main
gitlab   payments-repo-gitlab  /webhooks/gitlab/payments    true   branch=main
slack    platform-workspace    /webhooks/slack              true   commands=[/deploy]
```

---

## ork webhook play

Run a simulated webhook payload locally through the exact same engine `ork serve play` uses — target resolution, token check, CR construction, provenance stamping, admission validation, response payload evaluation. No cluster, no HTTP server, no real GitHub/GitLab/Slack account.

`--webhook` always identifies a real entry from `gateway.webhooks`, so its declared `branch`/`watch`/`commands` are exercised for real rather than re-typed on the command line. Signature/token verification is skipped — play mode tests the logic each source applies to its payload, not the HTTP transport.

`--source` is optional. Webhook entry names are unique across all four sources (`ork validate` enforces this), so it's resolved from `--webhook` automatically when omitted — pass `--source` only to disambiguate or to be explicit.

**Flags**

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--source` | `-s` | *(resolved from `--webhook`)* | `github`, `gitlab`, `slack`, or `generic` |
| `--webhook` | `-w` | *(required)* | Name of the configured `gateway.webhooks` entry to play |
| `--body` | `-b` | | Body file for `--source generic` (YAML or JSON) |
| `--command` | `-c` | | Slash command for `--source slack`, e.g. `/deploy` |
| `--text` | `-t` | | Command text for `--source slack`, e.g. `"servicerequest name=foo team=bar"` |
| `--event` | `-e` | | Push event JSON file for `--source github`/`gitlab` |
| `--fetch` | `-F` | | Simulated content-fetch override `<path>=<local-file>`, repeatable, for `--source github`/`gitlab` — capital `F`, since lowercase `-f` is already the global `--file` shorthand |
| `--simulate` | `-S` | | After play, hand each built CR to `ork simulate`; pass a `simulate.yaml` path to use assert mode |

### `--source generic`

The body file *is* the intent — no parsing needed.

```bash
ork webhook play -f katalog.yaml --webhook pagerduty --body body.json
```

### `--source slack`

`--text` is parsed exactly like a real slash command's text field: `"<target> key=value ..."`.

```bash
ork webhook play --source slack -f katalog.yaml --webhook platform-workspace \
  --command /deploy --text "servicerequest name=payments-api team=platform image=nginx"
```

`--command` is checked against the entry's declared `commands:` list before anything else runs.

### `--source github` / `--source gitlab`

`--event` is the real push event shape — see `intake.GitHubPushEvent` / `intake.GitLabPushEvent` for the exact fields read. Branch and watch-pattern filtering run for real against the event.

```bash
ork webhook play --source github -f katalog.yaml --webhook payments-repo \
  --event push-event.json \
  --fetch services/payments/intent.yaml=repo-example/services/payments/intent.yaml
```

Content can't be fetched from a real repo in play mode — there's no live GitHub/GitLab account to call. Pass `--fetch <matched-path>=<local-file>` once per path you expect the watch pattern to match, to supply what the Contents/Repository Files API would have returned. A matched path with no override is reported and skipped rather than failing the whole run — useful on its own for checking that a `watch:` pattern matches what you expect before wiring up `--fetch` at all.

A single push can match several files; each runs through the full chain independently.

**Output**

```text
▶  ork webhook play
  source:  github
  webhook: payments-repo
  path:    /webhooks/github/payments

→  Branch filter
   ✓ push on main matches watched branch
→  Watch pattern match
   ✓ 1 matched file(s): services/payments/intent.yaml

→  Content fetch — services/payments/intent.yaml
   ✓ read services/payments/intent.yaml (432 bytes) from repo-example/services/payments/intent.yaml

→  stage 1 · Target resolution
   ✓ kind=ServiceRequest  target=servicerequest  alias=(none)
→  stage 2 · Token check
   ✓ token payments-repo can create on ServiceRequest
→  stage 3 · CR construction
   ✓ name=payments-api  namespace=default
→  stage 4 · Provenance annotations
      ...
→  stage 5 · Admission validation
   ✓ passed — no violations
→  stage 6 · Response payload
      no serve.config.response declared — default CR response
   ✓ payload evaluated

✓ Webhook payload would be accepted
```

### `--simulate`

Hands each successfully built CR to `ork simulate`, the same handoff `ork serve play --simulate` does:

```bash
ork webhook play --source github -f katalog.yaml --webhook payments-repo \
  --event push-event.json \
  --fetch services/payments/intent.yaml=repo-example/services/payments/intent.yaml \
  --simulate
```

Omit the value for op-print mode, or pass a `simulate.yaml` path to use assert mode. For `github`/`gitlab`, a push matching several files runs `ork simulate` once per matched, fetched file.

---

## Related

- [`ork serve play`](13-serve.md#ork-serve-play) — the counterpart for CLI/CI-driven intents; `ork webhook play` shares its engine
- [`ork token`](15-token.md) — the equivalent local-testing tool for `gateway.api.auth.tokens`
- [Webhook Intake concept](../../concepts/self-service/09-webhook-intake.md)
- [Webhook credential verification](../../security/09-webhook-verification.md)
