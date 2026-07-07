# Best practices

---

## One E2E per Katalog

Each Katalog should have its own `e2e.yaml` in the same directory. Keep the scope tight: one Katalog, one CRD, one CR, a small set of checkpoints. When a test fails, the scope of the failure is obvious.

```text
my-operator/
  katalog.yaml
  crd.yaml
  cr.yaml
  e2e.yaml     ← here, not in a parent directory
```

When a Komposer combines several Katalogs, write a suite file at the root that imports each sub-test via `imports:`. The sub-tests stay individually runnable. The suite gives CI one entry point.

---

## Compose long expect lists with `include:`

As an operator matures, its `expect:` list grows. Resist leaving it all in one file — when checkpoint 7 of 15 fails, you re-read the whole thing to understand what phase you're in.

Split checkpoints by lifecycle phase and compose them with `include:`:

```yaml
# e2e.yaml — composed from phases
expect:
  - include: ./e2e/infra-ready.yaml   # resources created
  - include: ./e2e/behavior.yaml      # business logic verified
  - include: ./e2e/cleanup.yaml       # CR deleted, children gone
```

Each file uses `expect:` as its root key:

```yaml
# e2e/cleanup.yaml
expect:
  - name: Children removed after CR deletion
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        name: my-app
        namespace: default
        count: 0
```

`ork validate` expands all includes before reporting — the checkpoint list and count reflect the full run order.

**Put include files in a subfolder** (conventionally `e2e/`) to keep the root directory clean. The phase name is the filename: `infra-ready.yaml`, `failover.yaml`, `cleanup.yaml`. Each can be read, improved, and scaled independently without touching the others. Adding a new infrastructure assertion only means editing `infra-ready.yaml` — the other phases are unaffected.

---

## Prefer DSL over raw `commands[].run`

`commands[].run` is a raw shell string. It works but it is opaque — it has no type safety, `ork validate` cannot check it, and error messages are raw shell output. Prefer the typed DSL subcommands whenever the operation maps to a known kubectl command:

| Instead of | Use |
|-----------|-----|
| `kubectl get ... -o jsonpath=...` | `kubectl.get` with `field:` |
| `kubectl logs -l app=...` | `kubectl.logs` with `labelSelector:` |
| `kubectl delete pod $(kubectl get lease ...)` | `kubectl.delete` with `leaderElection:` |
| `kubectl exec <pod> -- cat /etc/config` | `kubectl.exec` |
| `kubectl port-forward ... & curl ... & kill` | `kubectl.port-forward` |

Reserve `commands[].run` for things that genuinely need shell: complex conditionals, multi-step sequences, tool-specific invocations.

---

## Use `leaderElection:` for HA operators

When your operator runs with `replicaCount > 1`, pod names change on every election. Never hardcode a pod name. Use `leaderElection:` on `kubectl.logs`, `kubectl.port-forward`, `kubectl.delete`, and `kubectl.exec` — Orkestra reads the Lease holder and targets the correct pod automatically.

```yaml
kubectl:
  # Assert the leader's log output
  logs:
    - leaderElection:
        lease: orkestra-konductor
        namespace: orkestra-system
      outputContains: "became konductor"

  # Kill the leader pod by name — without knowing it in advance
  delete:
    - leaderElection:
        lease: orkestra-konductor
        namespace: orkestra-system

  # Forward to the leader's HTTP endpoint
  port-forward:
    - leaderElection:
        lease: orkestra-konductor
      port: 8080
      path: /health
      jq: state
      equals: "healthy"
```

This is the typed alternative to the shell anti-pattern:

```bash
# fragile — breaks when the pod restarts
kubectl delete pod $(kubectl get lease my-lease -o jsonpath='{.spec.holderIdentity}') -n my-ns
```

---

## Always include a cleanup checkpoint

Every test should verify that child resources are cleaned up when the CR is deleted. Without this, the test passes even if a Deployment or Service leaked.

```yaml
- name: Cleanup verified
  after: cr-deleted
  timeout: 30s
  resources:
    - kind: Deployment
      name: my-app
      namespace: default
      count: 0
    - kind: Service
      name: my-app-svc
      namespace: default
      count: 0
    - kind: MyApp          # the CR itself
      name: my-app
      namespace: default
      count: 0
```

`count: 0` on the CR itself confirms the finalizer released and the object is fully gone.

---

## Add `onFailure` to your hardest checkpoints

When a test fails in CI you have no shell access — you rely entirely on what was printed. Without `onFailure`, a timeout failure shows you only the assertion that timed out. With it, you see pod logs, describe output, and events captured at the exact moment of failure.

Add per-expectation `onFailure` to checkpoints that are slow, involve async state, or interact with external dependencies. Collect the state most relevant to that specific assertion — not a generic dump:

```yaml
- name: Operator reaches Ready
  after: cr-applied
  timeout: 120s
  kubectl:
    get:
      - kind: MyApp
        name: my-app
        namespace: default
        field: .status.phase
        equals: Ready
  onFailure:
    kubectl:
      logs:
        - labelSelector: app=my-operator
          namespace: default
          since: 3m
      describe:
        - kind: MyApp
          name: my-app
          namespace: default
```

Use `spec.onFailure` as a global fallback — a broad cluster snapshot that runs once after all checkpoints complete, regardless of which one failed:

```yaml
spec:
  onFailure:
    kubectl:
      get:
        - kind: MyApp
          name: my-app
          namespace: default
      events:
        - kind: Deployment
          name: my-app
          namespace: default
    commands:
      - kubectl get pods -A -o wide
```

The two levels complement each other: per-expectation captures focused state at the moment of failure; spec-level captures the broad picture after the full run.

---

## Name checkpoints for the behavior, not the resource

```yaml
# bad — the resource type is already in the resources list
- name: Deployment check

# good — describes what the operator should have done
- name: App deployed and serving traffic
- name: Credentials not recreated on second apply
- name: New leader serves authoritative state after failover
- name: Children removed after CR deletion
```

The checkpoint name appears in pass/fail output. Make it answer "what behavior was verified?"

---

## Set realistic timeouts

Timeouts are per-checkpoint. Set them based on what that specific resource actually needs:

| Resource | Typical wait |
|----------|-------------|
| Namespace, ConfigMap, Secret | 10–15s |
| Service | 15–30s |
| Deployment with fast image | 60–90s |
| Deployment with slow pull | 120–180s |
| StatefulSet | 120–300s |
| In-cluster loop (e.g. CRD check every 90s) | 120s minimum |

Too short: flaky tests. Too long: slow CI. When a checkpoint depends on a background loop with a known tick interval, the timeout must exceed one full tick — not just the expected happy-path duration.

---

## Prefer `name:` over namespace-level any-match for cleanup checks

Any-match (`kind: Deployment, namespace: default, count: 0`) passes when there are zero Deployments in the namespace at all. That's almost never what you want — another test may have left a Deployment there. Name specific resources:

```yaml
# fragile — passes if anything cleans the namespace
- kind: Deployment
  namespace: default
  count: 0

# correct — asserts this exact resource is gone
- kind: Deployment
  name: my-app
  namespace: default
  count: 0
```

---

## Run validate before cluster work

```bash
ork validate -f e2e.yaml
```

Validate catches file path errors, missing `after:` values, invalid kubectl DSL, and broken `include:` references in milliseconds — without touching a cluster. There is no reason to provision a cluster before validation passes.

In CI, add validate as a separate step before the e2e step:

```yaml
- name: Validate E2E spec
  run: ork validate -f e2e.yaml

- name: Run E2E
  run: ork e2e -f e2e.yaml
```

---

## CI integration

`ork e2e` exits `0` on pass and `1` on any failure. It works with any CI system without configuration.

```yaml
# GitHub Actions
- name: Run E2E
  run: ork e2e -f e2e.yaml
```

For tests that require a multi-node cluster, use `--workers`:

```yaml
- name: Run E2E (HA)
  run: ork e2e -f e2e.yaml --workers 2
```

For parallel test jobs, pass `--cluster` with a unique name per job to avoid kind cluster name collisions:

```yaml
- name: Run E2E (shard ${{ matrix.shard }})
  run: ork e2e -f e2e.yaml --cluster ork-e2e-${{ matrix.shard }}
```

---

→ Back: [Suites and imports](03-suite-and-imports.md) | [Concept index](index.md)
