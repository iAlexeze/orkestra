# E2E

An `E2E` is a declarative end-to-end test for a Katalog. It tells Orkestra exactly what to apply, which cluster to use, and what the expected state is after each step.

```text
Motif     — smallest reusable unit
    ↓
Katalog   — operator declaration
    ↓
Komposer  — platform declaration
    ↓
E2E       — end-to-end test for a Katalog
```

`ork validate` checks the E2E structure. `ork e2e` runs it.

---

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E

metadata:
  name: hello-website-e2e
  description: >
    Deploy a single nginx website, verify the Deployment and Service come up,
    then delete the CR and verify cleanup.

spec:
  katalog: ./katalog.yaml     # path to the Katalog under test
  crd: ./crd.yaml             # CRD to apply before the test run
  cr: ./cr.yaml               # CR to apply when the test starts

  cluster:
    provider: kind            # kind | existing
    name: ork-e2e             # cluster name (created if missing)
    reuse: false              # keep the cluster after the test

  # Optional — apply prerequisite resources before the operator starts.
  setup:
    apply:
      - ./prereqs/image-pull-secret.yaml
      - ./prereqs/test-configmap.yaml
    wait:
      - kind: Secret
        name: image-pull-secret
        namespace: default
        timeout: 10s

  expect:
    - name: Deployment created
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

---

## `metadata`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Test identifier. Used in output and CI reports. |
| `description` | no | What this test exercises. |

---

## `spec.katalog`

Path to the `kind: Katalog` file to test. Relative to the E2E file's location.

```yaml
spec:
  katalog: ./katalog.yaml
```

---

## `spec.crd`

Path to the CRD YAML to apply before the test run. If the CRD is already installed in the target cluster, this is a no-op.

```yaml
spec:
  crd: ./crd.yaml
```

---

## `spec.cr`

Path to the Custom Resource YAML to apply when the test starts. Orkestra applies this file at the start of the `cr-applied` phase.

```yaml
spec:
  cr: ./cr.yaml
```

---

## `spec.setup`

Applies prerequisite manifests to the cluster after the CRD is installed but before the Katalog operator starts and the CR is applied. Use it to create Secrets, ConfigMaps, Namespaces, or any other resources the operator needs in order to reconcile successfully.

```yaml
spec:
  setup:
    apply:
      - ./prereqs/image-pull-secret.yaml
      - ./prereqs/test-configmap.yaml
    wait:
      - kind: Secret
        name: image-pull-secret
        namespace: default
        timeout: 10s
```

`setup` is optional. Omit it entirely when the test has no prerequisites.

### `setup.apply`

An ordered list of YAML file paths to apply, relative to the E2E file's location. Files are applied in declaration order. Each file may contain multiple YAML documents separated by `---`.

```yaml
setup:
  apply:
    - ./prereqs/namespace.yaml
    - ./prereqs/pull-secret.yaml
    - ./prereqs/seed-data.yaml
```

`apply` runs after the cluster is ready and the CRD is installed. It runs before the Katalog operator starts watching and before the CR is applied.

### `setup.wait`

An optional ordered list of resources to wait for after `apply` completes. The test blocks until every listed resource exists and satisfies the `ready` constraint (if set). If any wait times out, the test fails before the CR is applied.

```yaml
setup:
  wait:
    - kind: Secret
      name: image-pull-secret
      namespace: default
      timeout: 10s
    - kind: Namespace
      name: staging
      timeout: 10s
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind to wait for. |
| `name` | yes | Exact resource name. |
| `namespace` | no | Namespace. Cluster-scoped resources (Namespace, ClusterRole) omit this. |
| `ready` | no | When `true`, waits for the resource to reach a ready state, not just existence. |
| `timeout` | no | Per-resource timeout. Defaults to `30s`. |

---

## `spec.cluster`

Controls where the test runs.

```yaml
spec:
  cluster:
    provider: kind        # kind | existing
    name: ork-e2e         # cluster name
    reuse: false          # keep the cluster after the test completes
```

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `provider` | no | `kind` | `kind` creates a local kind cluster. `existing` uses the current kubeconfig context. |
| `name` | no | `ork-e2e` | Cluster name. For `kind`, this is the kind cluster name. For `existing`, used only in test output. |
| `reuse` | no | `false` | When `true`, the cluster is kept after the test completes — useful during development to inspect state. When `false`, the cluster is deleted on test completion. |

---

## `spec.expect`

An ordered list of assertion checkpoints. Each checkpoint waits up to `timeout` for all `resources` conditions to pass.

```yaml
spec:
  expect:
    - name: Deployment created
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          namespace: default
          ready: true
```

### `after`

The lifecycle phase that must have occurred before this checkpoint runs.

| Value | Description |
|-------|-------------|
| `cr-applied` | The CR has been applied to the cluster and the initial reconcile has completed. |
| `cr-deleted` | The CR has been deleted and the finalizer cleanup loop has completed. |

### `timeout`

How long to wait for the checkpoint to pass before the test fails. Accepts Go duration strings: `30s`, `2m`, `90s`.

### `resources`

List of resource assertions. All must pass for the checkpoint to pass.

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind: `Deployment`, `Service`, `Pod`, `ConfigMap`, etc. |
| `name` | no | Exact resource name. Omit to match any resource of this kind in the namespace. |
| `namespace` | no | Namespace to check. Defaults to `default`. |
| `ready` | no | When `true`, waits for the resource to be in a ready/available state (Deployment: `availableReplicas == replicas`, Pod: `Ready` condition). |
| `count` | no | Expected number of resources matching `kind`/`name`/`namespace`. `0` means the resource must not exist — used to verify cleanup. |

---

## Lifecycle phases

```text
cluster ready
    ↓
CRD applied
    ↓
setup.apply  ← prerequisite manifests applied (if spec.setup declared)
    ↓
setup.wait   ← block until setup resources exist/ready (if setup.wait declared)
    ↓
Katalog loaded — operator starts watching
    ↓
CR applied  ← "cr-applied" checkpoints run here
    ↓
Expect checkpoints evaluated
    ↓
CR deleted  ← "cr-deleted" checkpoints run here
    ↓
Expect checkpoints evaluated
    ↓
Cluster torn down (if reuse: false)
```

---

## Multiple CRs

To test more than one CR, declare additional `cr:` paths or use `ork e2e` with a directory:

```yaml
spec:
  katalog: ./katalog.yaml
  crd: ./crd.yaml
  cr: ./cr.yaml          # primary CR

  expect:
    - name: Primary CR ready
      after: cr-applied
      timeout: 60s
      resources:
        - kind: MyApp
          name: my-app
          ready: true
```

---

## Validation

`ork validate` checks the E2E structure without running the test:

```bash
ork validate -f my-e2e.yaml
```

Validation catches: missing required fields, unknown `after` values, unknown `provider` values, unreachable file paths for `katalog`, `crd`, `cr`, and `setup.apply` entries.

---

## See also

- [Katalog schema](../02-katalog/01-katalog.md) — the Katalog being tested
- [Motif schema](../01-motif/index.md) — reusable Motifs imported by the Katalog
- [Schema index](../index.md) — full schema reference
