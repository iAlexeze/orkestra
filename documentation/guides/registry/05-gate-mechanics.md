# Gate Mechanics

Gates are the mechanism that turns a YAML file into a trusted artifact. Every pattern pushed to the registry carries a record of what was verified — and when. Consumers read that record before importing.

---

## The gate sequence

```text
ork push ./
  │
  ├─ 1. Validate   — schema, types, required fields, no local imports
  ├─ 2. Simulate   — reconciler assertions, in-memory, <1s
  ├─ 3. E2E        — real cluster, kind, 2–5 min
  └─ 4. Push       — artifact published with quality annotations
```

Simulate runs before E2E. A failing simulate blocks E2E — no point spending minutes on a cluster when the reconciler is already broken.

**Which gates run** depends on which files are present:

| Files present | What runs |
|---|---|
| `simulate.yaml` | Simulate only |
| `e2e.yaml` | E2E only |
| Both | Simulate, then E2E |
| Neither | Validation only — artifact published immediately |

---

## The quality annotation

Every pushed artifact carries these OCI annotations regardless of gate result:

```text
io.orkestra.simulate.status     passed | no-assertion | skipped
io.orkestra.simulate.assertions 2
io.orkestra.simulate.duration   14ms
io.orkestra.simulate.tested_at  2026-06-14T10:00:00Z

io.orkestra.e2e.status          passed | skipped
io.orkestra.e2e.duration        48s
io.orkestra.e2e.tested_at       2026-06-14T10:00:00Z
io.orkestra.e2e.runner          local | github-actions | gitlab-ci
```

`ork inspect` reads these annotations. The quality signal is immutable — it cannot be changed after the push without pushing a new artifact.

---

## Gate statuses in ork inspect

```text
Simulate:    ✓ Verified · 2 assertions · 14ms · tested 2 days ago
E2E:         ✓ Verified · 48s · tested 2 days ago

Simulate:    ⊘ Skipped (pushed with --force or --no-simulate)
E2E:         ⊘ Skipped (pushed with --force or --no-e2e)

Simulate:    ⚠ No assertions (simulate.yaml present, no expect: block)
E2E:         - Not verified (no e2e.yaml in artifact)
```

A consumer who sees `⊘ Skipped` knows the gates were bypassed. They can decide whether to accept a draft for review, or wait for a fully-verified artifact.

---

## Skipping gates

### --force

Bypasses both gates. Use when sharing a draft for review before the tests are written:

```bash
ork push --force ./
```

The gates were bypassed — that fact is baked into the artifact. `ork inspect` shows `⊘ Skipped` for both.

### --no-e2e

Runs simulate but skips E2E. Useful in CI environments without Docker-in-Docker:

```bash
ork push --no-e2e ./
```

The artifact carries `simulate: ✓ Verified` and `e2e: ⊘ Skipped`. Consumers know the reconciler was verified but cluster behavior was not.

### --no-simulate

Skips simulate but runs E2E. Unusual — simulate is fast and catches more:

```bash
ork push --no-simulate ./
```

---

## The draft workflow

The gate failure itself is information worth sharing. This is the 03-katalog-cache story:

You have a Katalog with a failing simulate assertion — it asserts a ConfigMap that the Motif doesn't yet create. You want to share the draft with a teammate before implementing the ConfigMap.

```bash
# Run simulate — see the failure
ork simulate
# ✗ configmaps/my-cache-config not found (expected cycle 1 create)
# 3/4 assertions passed, 1 failed — FAIL

# Push the draft anyway — gates are skipped and recorded
ork push --force --no-e2e ./
# ✓ Pushed: oci://ghcr.io/myorg/katalogs/cache-operator:v1.0.0
#   Simulate: ⊘ Skipped
#   E2E:      ⊘ Skipped
```

Your teammate inspects the artifact:

```bash
ork inspect cache-operator:v1.0.0
# Simulate:    ⊘ Skipped
# E2E:         ⊘ Skipped
```

They know it's a draft. When you implement the ConfigMap and the assertion passes, you push cleanly. The new artifact carries `✓ Verified`.

---

## Simulate assertions are the published contract

The simulate assertions are not just your tests — they are the behavioral specification you publish alongside the artifact. A consumer who runs:

```bash
ork pull cache-operator:v1.0.0 -o ./local
ork simulate -f ./local/simulate.yaml
```

...is running the same assertions you ran before publishing. The assertions describe what the operator claims to do. A pattern that ships with passing assertions has proven its behavior. One without has made a promise.

---

## Gates in CI

```yaml
# .github/workflows/publish.yml
- name: Push pattern
  run: |
    export ORK_REGISTRY=ghcr.io/myorg/katalogs
    ork push webapp-operator:${{ github.ref_name }} ./
```

In GitHub Actions, `io.orkestra.e2e.runner` is recorded as `github-actions`. Consumers can see the pattern was verified in CI.

For CI runners without Docker-in-Docker:

```bash
# Run simulate (no Docker needed), skip E2E
ork push --no-e2e webapp-operator:${{ github.ref_name }} ./
```

The artifact carries `simulate: ✓ Verified` and makes clear that E2E was deferred.

---

## Try it

```bash
ork init --pack registry-guide
cd 03-katalog-cache

# See the simulate gate fail
ork simulate

# Push as a draft — gates skipped, recorded in artifact
ork push --force --no-e2e ./
ork inspect cache-operator:v1.0.0

# Fix the assertion (or the motif), then push cleanly
ork push ./
ork inspect cache-operator:v1.0.0
```

→ Next: [Composing Patterns](06-composing.md)
