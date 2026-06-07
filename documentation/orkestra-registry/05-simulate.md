# Simulate

`simulate.yaml` is unit testing for your reconciler — without writing Go. It runs the same reconcile loop that executes in production against a fake in-memory cluster, asserts which resources were created and in which cycle, and produces a pass or fail in under a second. No cluster, no Helm, no Docker.

When `simulate.yaml` is present in a pattern directory, `ork registry push` runs it automatically before the E2E gate and blocks publication if any assertion fails.

> For the complete field reference — `spec`, `expect`, `ops`, `crds` — see **[Simulate schema](../concepts/simulate/02-simulate-kind.md)**.

---

## How it gates publication

```bash
# Simulate runs automatically if simulate.yaml is present — before E2E
ork registry push postgres:v14 ./patterns/postgres/

# Skip the gate
ork registry push postgres:v14 ./patterns/postgres/ --no-simulate

# Skip both gates
ork registry push postgres:v14 ./patterns/postgres/ --force
```

The result is baked into the OCI artifact as annotations:

```text
io.orkestra.simulate.status     passed | no-assertion | skipped
io.orkestra.simulate.duration   4ms
io.orkestra.simulate.tested_at  2026-05-24T10:00:00Z
```

`ork registry info` shows this alongside the E2E status. Consumers can see at a glance what guarantees a pattern carries before importing it.

---

## Simulate status in `ork registry info`

```text
ork registry info postgres:v14

postgres:v14
  Registry:    ghcr.io/orkspace/orkestra-registry/patterns/katalogs
  Kind:        Katalog
  Digest:      sha256:abc123def456...
  Pushed:      2026-05-24T10:00:00Z
  Size:        14.2 KB

  Description: A declarative PostgreSQL deployment operator

  Simulate:    ✓ Verified · 4ms · tested 2h ago
  E2E:         ✓ Verified · 45s · tested 2h ago
```

Three possible simulate statuses:

| Status | Meaning |
|--------|---------|
| `✓ Verified` | `expect:` blocks present and all assertions passed |
| `⚠ No assertions` | `simulate.yaml` present but no `expect:` — ran clean, nothing asserted |
| `~ Skipped` | `--no-simulate` or `--force` used at push time |

Simulate status is not shown in `ork registry list` — it appears in `ork registry info` only, alongside E2E.

---

## simulate.yaml — pattern shape

A minimal simulate spec alongside a Katalog pattern:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: postgres-sim
  description: Verify StatefulSet and headless Service created in cycle 1

spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  cycles: 5

  expect:
    steady: true
    noErrors: true
    ops:
      - cycle: 1
        verb: create
        resource: statefulsets
        name: my-postgres
      - cycle: 1
        verb: create
        resource: services
        name: my-postgres-headless
```

**`expect:` is optional.** Without it, `simulate.yaml` runs in op-print mode — confirms the reconciler exits clean, records `no-assertion` in the annotation, and never blocks a push. Add `expect:` when you know the resource sequence and want it enforced.

---

## Running simulate locally

Start by scaffolding a `simulate.yaml` from a live reconcile run — no manual rule writing:

```bash
ork simulate init
```

This runs the reconciler once, captures the cycle-1 create operations, and writes a `simulate.yaml` pre-filled with `expect:` rules. Then run it:

```bash
ork simulate
```

Use `--debug-ops` to see every op recorded across all cycles — useful when you want to add rules beyond what `init` captured, or understand what an unfamiliar operator does:

```bash
ork simulate --debug-ops
```

```text
  [debug-ops] 5 total ops recorded across all cycles:
  [debug-ops]   cycle=1   verb=create   resource=deployments          name=my-postgres
  [debug-ops]   cycle=1   verb=create   resource=services             name=my-postgres
  [debug-ops]   cycle=1   verb=create   resource=persistentvolumeclaims  name=my-postgres-data
  [debug-ops]   cycle=2   verb=update   resource=statuses             name=my-postgres
  [debug-ops]   cycle=3   verb=update   resource=statuses             name=my-postgres
```

Copy any ops you want to assert on into the `expect:` block in your `simulate.yaml`.

---

## Simulate vs E2E

Simulate and E2E answer different questions. Together they give you two quality layers with complementary failure modes.

| | Simulate | E2E |
|---|---|---|
| Cluster required | no | yes (kind) |
| Speed | <1s | 2–5 min |
| Runs in CI without Docker | yes | no |
| What is asserted | reconciler ops by cycle | Kubernetes resources in a live cluster |
| Catches | template errors, wrong resource names, wrong cycle order | webhook failures, RBAC gaps, real scheduling issues |

The push order reflects this: simulate runs first (instant feedback), E2E runs after (integration truth). A failing simulate blocks E2E — no point spending minutes on a cluster when the reconciler is already broken.

---

## Simulate in CI

Because simulate requires no cluster or Docker, it runs anywhere — including CI runners without Docker-in-Docker access. Use `--no-e2e` to skip the cluster gate in environments that can't provision kind, while still enforcing reconciler correctness via simulate:

```bash
ork registry push postgres:v14 ./patterns/postgres/ --no-e2e
```

The artifact carries `io.orkestra.simulate.status=passed` — consumers know the reconciler was verified even if the full cluster test was deferred.

---

## Testing a multi-operator pattern

When a pattern contains several Katalogs, a simulate aggregator at the root runs all sub-simulations together:

```text
multi-tenancy/
  komposer.yaml
  simulate.yaml                    ← aggregator
  01-basic-namespacing/
    katalog.yaml  crd.yaml  cr.yaml  simulate.yaml
  02-cross-access-control/
    katalog.yaml  crd.yaml  cr.yaml  simulate.yaml
```

The aggregator:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: multi-tenancy-sim

imports:
  - ./01-basic-namespacing/simulate.yaml
  - ./02-cross-access-control/simulate.yaml
```

`ork registry push` discovers `simulate.yaml` at the pattern root and runs it. Each sub-simulation runs independently; one failure stops the push.

`ork validate -f simulate.yaml` checks the spec and confirms all imported files exist on disk before any simulation runs:

```text
Validating Simulate...

  ✓ metadata.name: multi-tenancy-sim
  ✓ imports: ./01-basic-namespacing/simulate.yaml (found)
  ✓ imports: ./02-cross-access-control/simulate.yaml (found)

2 import(s) valid
```

→ Full field reference: **[simulate.yaml schema](../concepts/simulate/02-simulate-kind.md)**
