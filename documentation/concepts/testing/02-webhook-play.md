# Webhook Play

`ork webhook play` simulates a webhook payload locally — the full intake path for a GitHub push, a GitLab push, a Slack slash command, or a generic HTTP body — with no live account, no HTTP server, and no cluster.

It runs the same engine as `ork serve play`, so the same six delivery stages run. The difference is the entry point: `ork webhook play` starts from a webhook payload, which means source-specific logic runs first — branch filters, watch patterns, command parsing, and content fetching.

---

## Why it exists

`ork serve play` tests delivery from a flat intent file. That covers the CLI path, the CI pipeline path, and any caller that constructs an intent directly.

Webhooks have a layer before the delivery chain. A GitHub push event carries a branch, a set of changed files, and a ref — none of which is in the intent file. The Gateway evaluates them: does the branch match the watched branch? Do any changed paths match the `watch:` pattern? Which files need to be fetched? Only after that does delivery begin.

You cannot test that logic with `ork serve play`. `ork webhook play` tests it.

---

## Basic usage

```bash
ork webhook play --webhook <name> [source-specific flags]
```

`--webhook` identifies a real entry from `gateway.webhooks` in your Katalog. The source is resolved from the entry automatically — all webhook names are unique across sources (`ork validate` enforces this). Pass `--source` explicitly only to disambiguate or to be self-documenting.

---

## By source

### Generic

```bash
ork webhook play --webhook pagerduty --body body.json
```

The body file is the intent — no parsing. It is passed directly into the delivery chain.

### Slack

```bash
ork webhook play --webhook platform-workspace \
  --command /deploy \
  --text "servicerequest name=payments-api team=platform image=nginx"
```

`--command` is checked against the entry's declared `commands:` list before anything else runs. `--text` is parsed as `<target> key=value ...` — the same parse the real slash command handler applies.

### GitHub / GitLab

```bash
ork webhook play --webhook payments-repo \
  --event push-event.json \
  --fetch services/payments/intent.yaml=./intent.yaml
```

`--event` is a real push event JSON file. Branch filtering and watch pattern matching run against it for real.

In play mode there is no live GitHub or GitLab account to call. Pass `--fetch <matched-path>=<local-file>` for each path you expect the watch pattern to match — it supplies what the Contents API would have returned. A matched path with no override is reported and skipped rather than failing the whole run. That means you can run without `--fetch` first to check that your `watch:` pattern matches the files in the event.

A push matching several files runs the full delivery chain once per matched, fetched file.

---

## Output

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
   ✓ read services/payments/intent.yaml (432 bytes) from ./intent.yaml

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
   ✓ payload evaluated

✓ Webhook payload would be accepted
```

Each stage that fails exits non-zero with a clear message pointing to which stage failed and why.

---

## Chaining into reconciliation

`--simulate` hands each built CR to `ork simulate` — the same handoff `ork serve play --simulate` does:

```bash
ork webhook play --webhook payments-repo \
  --event push-event.json \
  --fetch services/payments/intent.yaml=./intent.yaml \
  --simulate
```

Omit the value to run in op-print mode (print resources without assertions), or pass a `simulate.yaml` path for assert mode. When a push matches multiple files, `ork simulate` runs once per matched, fetched file.

This is the full path from a webhook push to reconciled child resources, locally, in milliseconds.

---

## What `--simulate` does not replace

`--simulate` runs the Orkestra reconciler — the same one that runs in production. It does not run the Kubernetes admission webhook. A rule that uses `unique` or `external` (the two operators skipped in local admission) will not fire here.

Use `ork e2e` for cases that need real webhook TLS, real admission, or real pod scheduling.

---

## Where to go next

- [Testing in Orkestra](index.md) — full testing overview
- [Gate and local gateway](01-gate.md) — admission rule testing
- [Webhook intake concept](../self-service/09-webhook-intake.md)
- [`ork webhook play` CLI reference](../../reference/cli/16-webhook.md)
- [`ork serve play` CLI reference](../../reference/cli/13-serve.md#ork-serve-play)
