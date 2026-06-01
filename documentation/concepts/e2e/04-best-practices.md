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

When a Komposer combines several Katalogs, write a suite file at the root that imports each sub-test. The sub-tests stay individually runnable. The suite gives CI one entry point.

---

## Use imports instead of one large E2E

Resist putting all assertions into a single `e2e.yaml`. A large file is hard to debug: when checkpoint 7 of 12 fails, you re-run the whole thing and wait through 1-6 again.

Separate concerns into focused files and compose them:

```yaml
# operator/e2e.yaml — suite
imports:
  - ./e2e-basic.yaml        # core resources created
  - ./e2e-once-secret.yaml  # once: semantics
  - ./e2e-cleanup.yaml      # finalizer and deletion order
```

Each file can be run in isolation during development (`ork e2e -f e2e-once-secret.yaml`) and run together in CI via the suite.

---

## Always include a `cr-deleted` cleanup checkpoint

Every test should verify that child resources are cleaned up when the CR is deleted. Without this, the test passes even if the Deployment or Service leaked.

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

## Set realistic timeouts

Timeouts are per-checkpoint. Set them based on what that specific resource actually needs:

| Resource | Typical wait |
|----------|-------------|
| Namespace, ConfigMap, Secret | 10–15s |
| Service | 15–30s |
| Deployment with fast image | 60–90s |
| Deployment with slow pull | 120–180s |
| StatefulSet | 120–300s |
| Custom operator (depends on logic) | 60–120s |

Too short: flaky tests. Too long: slow CI. A failing test with a 5-minute timeout is painful.

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

## Name checkpoints for the behavior, not the resource

```yaml
# bad — the resource type is already in the resources list
- name: Deployment check

# good — describes what the operator should have done
- name: App deployed and serving traffic
- name: Credentials not recreated on second apply
- name: Children removed after CR deletion
```

The checkpoint name appears in pass/fail output. Make it answer "what behavior was verified?"

---

## Run validate before cluster work

```bash
ork validate -f e2e.yaml
```

Validate catches file path errors, missing `after:` values, and invalid imports in milliseconds. There is no reason to provision a cluster before validation passes.

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

For parallel test jobs, pass `--cluster` with a unique name per job to avoid kind cluster name collisions:

```yaml
- name: Run E2E (shard ${{ matrix.shard }})
  run: ork e2e -f e2e.yaml --cluster ork-e2e-${{ matrix.shard }}
```

---

→ Back: [Suites and imports](03-suite-and-imports.md) | [Concept index](index.md)
