# Core spec fields

The four required inputs that tell `ork e2e` what to test and where to run.

---

## `spec.katalog`

Path to the `kind: Katalog` file under test. Relative to the E2E file's location.

```yaml
spec:
  katalog: ./katalog.yaml
```

Optional when `spec.custom.target` is set — see [05-custom-target.md](05-custom-target.md).

---

## `spec.crd`

Path to the CRD YAML to apply before the test run. If the CRD is already installed in the target cluster, this is a no-op.

```yaml
spec:
  crd: ./crd.yaml
```

---

## `spec.crdFiles`

Additional CRD paths applied alongside `spec.crd`. Use when the Katalog covers multiple CRDs stored in separate files.

```yaml
spec:
  crdFiles:
    - ./crd-app.yaml
    - ./crd-route.yaml
```

All paths in `crd` and `crdFiles` are applied. The effective set is `[crd] + crdFiles`. Either field may be omitted; at least one CRD path is required unless the CRD is already installed.

---

## `spec.cr`

Path to the Custom Resource YAML to apply at the start of the `cr-applied` phase.

```yaml
spec:
  cr: ./cr.yaml
```

---

## `spec.crFiles`

Additional CR paths applied at the start of the `cr-applied` phase alongside `spec.cr`. Use when the test exercises multiple CRDs from the same Katalog.

```yaml
spec:
  crFiles:
    - ./cr-app.yaml
    - ./cr-route.yaml
```

All paths in `cr` and `crFiles` are applied together. Deletion at `cr-deleted` covers all applied CRs. The effective set is `[cr] + crFiles`.

---

## `spec.cluster`

Controls where the test runs.

```yaml
spec:
  cluster:
    provider: kind        # kind | existing
    name: ork-e2e
    reuse: false
```

| Field | Default | Description |
|-------|---------|-------------|
| `provider` | `kind` | `kind` creates a local kind cluster. `existing` uses the current kubeconfig context. |
| `name` | `ork-e2e` | Cluster name. For `kind`, the kind cluster name. For `existing`, used in output only. |
| `reuse` | `false` | When `true`, keeps the cluster after the test — useful for debugging failed runs. |

---

---

## `spec.custom`

Declares the target runtime when Orkestra is not the operator under test. Bundle generation and Orkestra helm install/uninstall are skipped. Everything else runs unchanged.

```yaml
spec:
  custom:
    target: kubernetes
  crd: ./crd.yaml
  cr: ./cr.yaml
  setup:
    helm:
      - repo: https://charts.example.com
        chart: my-operator
```

`target` must be `kubernetes`. `container` is reserved and not yet supported.

See [05-custom-target.md](05-custom-target.md) for full documentation and use cases.

---

## `spec.notes`

Declares user-defined note functions available as template expressions in `when:` and `anyOf:` conditions on `expect` entries. Uses the same syntax as `notes:` in a Katalog — a list of named Go template expressions evaluated against the current context.

```yaml
spec:
  notes:
    functions:
      - name: inBusinessHours
        expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'
```

Notes defined here are available by name in any `when:` or `anyOf:` condition on an `expect` entry:

```yaml
expect:
  - name: Feature enabled during business hours
    after: cr-applied
    timeout: 30s
    when:
      - field: '{{ inBusinessHours }}'
        equals: "true"
    kubectl:
      get:
        - kind: Deployment
          name: my-app
          namespace: default
          field: .metadata.annotations.feature-enabled
          equals: "true"
```

All built-in note functions (`weekday`, `timeInWindow`, etc.) are available inside note expressions. Notes may call each other.

| Field | Required | Description |
|-------|----------|-------------|
| `functions` | no | List of named note functions. |
| `functions[].name` | yes | Name used to call the note in template expressions: `{{ myNote }}`. |
| `functions[].expression` | yes | Go template string evaluated at runtime. Result is always a string. |
| `functions[].description` | no | Human-readable description of the note. |

---

## `spec.onFailure`

Declares kubectl operations and shell commands to run and print to the terminal when any expectation fails. Uses the same `kubectl:` DSL as `expect` entries — assertion fields are present on the structs but never evaluated. Output is always printed. Teardown still runs after `onFailure` completes.

```yaml
spec:
  onFailure:
    kubectl:
      logs:
        - labelSelector: app=my-operator
          namespace: default
          since: 2m
      describe:
        - kind: Deployment
          name: my-operator
          namespace: default
      events:
        - kind: Pod
          name: my-operator-pod
          namespace: default
    commands:
      - kubectl get pods -A -o wide
```

| Field | Description |
|-------|-------------|
| `kubectl` | Accepts the full `kubectl:` DSL (`get`, `logs`, `describe`, `events`, `exec`). Assertion fields are ignored — output is printed. |
| `commands` | List of shell strings run via `sh -c`. Output is printed to the terminal. |

`onFailure` is spec-level — it runs once when the test has at least one failing expectation, not per-expectation. It is skipped entirely when all expectations pass.

To run diagnostics immediately when a specific checkpoint fails, add `onFailure:` directly on that `expect` entry — see [Per-expectation onFailure](03-expect.md#per-expectation-onfailure).

→ See [07-kubectl.md](07-kubectl.md) for the full `kubectl:` field reference.

---

## `imports`

A top-level list of other E2E files to run after this test completes. Used to compose test suites without a cluster-per-test overhead. See [04-imports.md](04-imports.md) for the full field reference.

```yaml
imports:
  - ./auth-e2e.yaml
  - ./rbac-e2e.yaml
  - path: ./infra-e2e.yaml
    freshCluster: true     # this one gets its own cluster
```

---

→ Next: [02-setup.md](02-setup.md) — prerequisite resources before the operator starts
