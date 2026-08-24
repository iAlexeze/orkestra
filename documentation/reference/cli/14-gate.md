# ork gate

| Command | Description |
|---------|-------------|
| [`ork gate`](#ork-gate-1) | Evaluate admission rules locally against a CR |
| [`ork gate run`](#ork-gate-run) | Start the gateway locally for Serve layer testing |

---

## `ork gate`

Evaluate admission rules locally against a CR — no cluster, no webhook server required.

In a cluster deployment, `validation.rules` are enforced by the admission webhook when a CR is applied via SSA. `ork gate` runs the same evaluation logic in-process from a Katalog and a CR file, giving you the same deny/warn output without any cluster access.

```bash
ork gate -f katalog.yaml --cr cr.yaml
```

`--file` defaults to `katalog.yaml` and `--cr` defaults to `cr.yaml` — both are optional when your files use the standard names.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Path(s) to katalog.yaml (repeatable) |
| `--cr` | | `cr.yaml` | CR file to evaluate |

## Examples

```bash
# Standard layout — no flags needed
ork gate

# Explicit paths
ork gate -f my-operator.yaml --cr cr-staging.yaml

# Evaluate against the gateway fixture
ork gate -f pkg/gateway/api/fixture/katalog.yaml \
  --cr pkg/gateway/api/fixture/crs/app-staging.yaml
```

## Output

### Pass

```text
▶  ork gate
  cr:      cr.yaml
  katalog: katalog.yaml

◆  PlatformResource  (apifixture)
  ✓ 9/9 rules passed
```

### Deny

```text
▶  ork gate
  cr:      cr.yaml
  katalog: katalog.yaml

◆  PlatformResource  (apifixture)
  ✗ spec.workloadType  spec.workloadType must be one of: app, cert, monitoring

  ✗ 8/9 rules passed · ✗ 1 denial(s)

admission denied
```

### Warn

```text
▶  ork gate
  cr:      cr.yaml
  katalog: katalog.yaml

◆  PlatformResource  (apifixture)
  ⚠ spec.productionApproval  production deploys should include a productionApproval ticket

  ✓ 8/9 rules passed · ⚠ 1 warning(s)
```

Deny exits non-zero. Warn exits zero — warnings are advisory only.

## Limitations

Two operators are skipped in local mode and noted in the output:

| Operator | Why skipped | Real behavior |
|----------|-------------|---------------|
| `unique` | Requires a live informer cache | Checked against all existing CRs in the cluster |
| `external` | Requires a real endpoint | Calls an HTTP service before admission |

Both are clearly noted in the output so you know they were not checked.

## How it works

`ork gate` runs the same admission evaluation as the real webhook — the same rules, the same `when:` conditions, the same `or:` logic. A rule guarded by `when: workloadType=cert` does not fire for an `app` CR, just as it wouldn't in the cluster.

`unique:` and `external:` rules are the only exceptions (see Limitations above).

Multi-document CR files are supported. Each document is matched to a CRD by kind.

## Relationship to `ork simulate`

`ork simulate` runs the reconciler against a fake cluster. `ork gate` runs admission rules against a CR. They are complementary:

- `ork gate` — answers "would this CR be admitted?"
- `ork simulate` — answers "what would the reconciler produce from this CR?"

Both require no cluster. Both use the same Katalog as the real gateway and runtime.

## Relationship to `ork serve play`

`ork serve play` runs the full gateway chain — it builds a CR from a flat intent file and evaluates admission rules as stage 5 of that chain. `ork gate` accepts a pre-built CR directly, which is useful when:

- You have a CR from another source (kubectl, GitOps, a script).
- You want to check admission rules in isolation from the delivery surface.
- You are debugging a webhook denial and want to reproduce it locally.

---

## `ork gate run`

Start the gateway locally for Serve layer testing — plain HTTP, no TLS, no admission webhooks.

This is the local equivalent of the production `ork gate` binary (which requires a pod). It starts the Gateway API and intake handlers on HTTP so you can run `ork serve apply` against it without a Helm deployment. Admission and conversion webhooks are disabled because TLS is not available in local mode.

```bash
ork gate run -f katalog.yaml
```

`--file` defaults to `katalog.yaml` in the current directory.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--file` | `-f` | Path(s) to katalog files (repeatable; defaults to `katalog.yaml`) |

### Output

```text
⎈  ork gate run
  local mode — admission and conversion webhooks are disabled (TLS not available)
  gateway API: http://localhost:8080
```

### Typical workflow

```bash
# Terminal 1 — start the local gateway
ork gate run -f katalog.yaml

# Terminal 2 — apply an intent against it
ork serve apply -f intent.yaml -t my-token

# Or with --verbose to see the raw response
ork serve apply -f intent.yaml -t my-token --verbose
```

### How it differs from production

| | `ork gate` (production) | `ork gate run` (local) |
|-|------------------------|------------------------|
| Build tag | `gateway` | `!runtime && !gateway` |
| Transport | HTTPS + TLS | HTTP only |
| Admission webhooks | Yes | No |
| Conversion webhooks | Yes | No |
| Health server | Yes | Yes |
| Gateway API + intake | Yes | Yes |

---

## Related

- [`ork simulate`](05-simulate.md) — reconciler simulation, no cluster needed
- [`ork serve play`](13-serve.md#ork-serve-play) — full gateway chain from an intent file
- [`ork serve apply`](13-serve.md#ork-serve-apply) — apply an intent against a live or local gateway
- [`validation.rules` schema reference](../schema/02-katalog/20-serve.md) — rule operators and actions
