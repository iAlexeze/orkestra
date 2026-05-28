# Writing Your First E2E Test

An **E2E** is a declarative end-to-end test for a Katalog. You describe what to apply, which cluster to use, and what the expected state is — Orkestra runs the test and tells you pass or fail.

No test framework. No Go. One YAML file.

```bash
ork validate -f e2e.yaml   # validate the structure
ork e2e -f e2e.yaml        # run the test
```

---

## What you will build

A test that:
1. Applies your Katalog's CRD and CR to a fresh kind cluster
2. Waits for the Deployment and Service to be created and ready
3. Deletes the CR and verifies that the Deployment and Service are cleaned up

---

## Step 1 — Write the E2E file

Create `e2e.yaml` in the same directory as your `katalog.yaml`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E

metadata:
  name: hello-website-e2e
  description: >
    Deploy a single nginx website. Verify the Deployment and Service come up.
    Delete the CR and verify cleanup.

spec:
  katalog: ./katalog.yaml
  crd: ./crd.yaml
  cr: ./cr.yaml

  cluster:
    provider: kind
    name: ork-e2e
    reuse: false

  expect:
    - name: Deployment created and ready
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          namespace: default
          ready: true

    - name: Service created
      after: cr-applied
      timeout: 30s
      resources:
        - kind: Service
          namespace: default

    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: Deployment
          name: hello-website
          namespace: default
          count: 0
        - kind: Service
          name: hello-website-svc
          namespace: default
          count: 0
        - kind: Website
          name: hello-website
          namespace: default
          count: 0
```

Four things drive the test:

- `spec.katalog` — the Katalog to run
- `spec.crd` / `spec.cr` — what to apply to the cluster
- `spec.setup` — optional prerequisite manifests applied before the operator starts
- `spec.expect` — checkpoints that must pass, in order

---

## Step 2 — Validate the structure

```bash
ork validate -f e2e.yaml
```

This checks all required fields, resolves the file paths for `katalog`, `crd`, and `cr`, and verifies that `after` values and `provider` values are recognised. No cluster is needed.

---

## Step 3 — Run the test

```bash
ork e2e -f e2e.yaml
```

What happens:

```text
1.  kind cluster "ork-e2e" created (if it doesn't exist)
2.  CRD applied from ./crd.yaml
3.  setup.apply runs (if declared) — prerequisite manifests applied
4.  setup.wait runs (if declared) — blocks until prerequisites exist
5.  Katalog loaded — operator starts watching
6.  CR applied from ./cr.yaml
7.  Checkpoint "Deployment created and ready" — waits up to 60s
8.  Checkpoint "Service created" — waits up to 30s
9.  CR deleted
10. Checkpoint "Cleanup verified" — waits up to 30s
11. Test passes
12. kind cluster deleted (reuse: false)
```

Pass output:

```text
✓  Deployment created and ready    (12s)
✓  Service created                 (2s)
✓  Cleanup verified                (5s)

PASS  hello-website-e2e  (19s)
```

---

## Adding setup prerequisites

When your operator needs a Secret, ConfigMap, or Namespace to exist before it can reconcile, use `spec.setup`:

```yaml
spec:
  katalog: ./katalog.yaml
  crd: ./crd.yaml
  cr: ./cr.yaml

  setup:
    apply:
      - ./prereqs/image-pull-secret.yaml   # applied before the operator starts
      - ./prereqs/test-configmap.yaml
    wait:
      - kind: Secret
        name: image-pull-secret
        namespace: default
        timeout: 10s

  expect:
    ...
```

`setup.apply` runs after the CRD is installed but before the Katalog operator starts watching. Files are applied in order.

`setup.wait` blocks until each listed resource exists. The test fails immediately if a wait times out — the CR is never applied, and the failure message names the blocking resource.

---

## Keeping the cluster for debugging

Set `reuse: true` to keep the cluster running after the test:

```yaml
cluster:
  provider: kind
  name: ork-e2e
  reuse: true
```

Then inspect the cluster manually:

```bash
kubectl get all -n default
kubectl describe deployment hello-website
```

Delete it when you're done:

```bash
kind delete cluster --name ork-e2e
```

---

## Testing against an existing cluster

Use `provider: existing` to run against your current kubeconfig context:

```yaml
cluster:
  provider: existing
  name: my-staging-cluster   # used in test output only, not in kubeconfig
  reuse: true                # existing clusters are never deleted by ork e2e
```

---

## What `count: 0` means

```yaml
resources:
  - kind: Deployment
    name: hello-website
    namespace: default
    count: 0
```

`count: 0` asserts that the resource does not exist. This is how you verify that Orkestra's cleanup logic (finalizers + `onDelete` templates) removed the child resources when the CR was deleted.

Without this, a test could pass even if the Deployment leaked after CR deletion.

---

## Adding more checkpoints

Checkpoints run in order. Add as many as you need:

```yaml
expect:
  - name: CR in Pending phase
    after: cr-applied
    timeout: 10s
    resources:
      - kind: Website
        name: hello-website
        namespace: default
        ready: false

  - name: Deployment created
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Deployment
        namespace: default
        ready: true

  - name: CR in Ready phase
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Website
        name: hello-website
        namespace: default
        ready: true
```

---

## Project layout

```text
my-operator/
  katalog.yaml
  crd.yaml
  cr.yaml
  e2e.yaml          ← the test
```

---

## CI integration

```yaml
# .github/workflows/e2e.yml
- name: Run E2E tests
  run: ork e2e -f e2e.yaml
```

`ork e2e` exits with code `0` on pass and `1` on any failure, making it compatible with any CI system.

---

## Next steps

- **[E2E schema reference](../reference/schema/04-e2e/index.md)** — complete field reference for every E2E option
- **[Writing Your First Katalog](./03-writing-your-first-katalog.md)** — build the Katalog this test exercises
- **[Writing Your First Motif](./02-writing-your-first-motif.md)** — reusable building blocks for your operators
