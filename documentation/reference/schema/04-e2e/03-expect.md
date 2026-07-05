# Expect

`spec.expect` is an ordered list of assertion checkpoints. Each checkpoint declares a lifecycle trigger (`after:`), a timeout, and a set of resource or command assertions. All assertions in a checkpoint must pass for the checkpoint to pass.

---

## Checkpoint structure

```yaml
expect:
  - name: Deployment created and ready
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Deployment
        namespace: default
        ready: true
    commands:
      - run: "kubectl get deploy -n default -o name"
        outputContains: "hello-website"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes* | Printed in the results table. Not required when `include:` is set. |
| `after` | yes | Lifecycle phase that must have occurred. |
| `timeout` | yes | Maximum wait time (Go duration: `30s`, `2m`, `90s`). |
| `wait` | no | Duration to sleep before the polling loop starts (Go duration: `5s`, `30s`). Useful when the previous step triggers an async operation that needs time to propagate before assertions are meaningful. |
| `resources` | no | Resource state assertions, polled until passing. |
| `commands` | no | Shell command assertions, run in the same polling loop. |
| `kubectl` | no | Structured kubectl subcommand assertions. See [kubectl block](07-kubectl.md). |
| `onFailure` | no | Diagnostic kubectl and shell commands to run and print when this specific checkpoint fails. See [Per-expectation onFailure](#per-expectation-onfailure). |
| `include` | no | Path to a YAML file containing a bare list of checkpoints to expand in place. See [Composing expectations](#composing-expectations-with-include). |

---

## `after`

| Value | When it triggers |
|-------|-----------------|
| `setup-complete` | After all setup steps finish, before the CR is applied. Use for Kubernetes workloads that are not operators — where setup is the thing under test, not a CR lifecycle. |
| `cr-applied` | After the CR has been applied and the initial reconcile has started. |
| `cr-deleted` | After the CR has been deleted and finalizer cleanup has run. |

---

## `resources`

A list of Kubernetes resource state checks. All must pass for the checkpoint to pass.

```yaml
resources:
  - kind: Deployment
    name: hello-website
    namespace: default
    ready: true

  - kind: Service
    name: hello-website-svc
    namespace: default

  - kind: Website
    name: hello-website
    namespace: default
    count: 0     # must not exist (cleanup check)
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind: `Deployment`, `Service`, `Pod`, `Secret`, etc. |
| `name` | no | Exact name. Omit to match any resource of this kind in the namespace. |
| `namespace` | no | Namespace. Defaults to `default`. |
| `ready` | no | `true` waits for available/ready condition. Deployment: `availableReplicas == replicas`. Pod: `Ready` condition true. |
| `count` | no | Exact expected count. `0` asserts the resource does not exist — use in `cr-deleted` checkpoints to verify cleanup. |

---

## `commands`

Shell commands run in the same polling loop as `resources`. Useful for assertions that go beyond resource existence — health endpoints, data validation, connectivity checks.

```yaml
commands:
  - run: "kubectl exec -n default deploy/hello-website -- wget -qO- localhost:80"
    exitCode: 0
    outputContains: "nginx"

  - run: "kubectl get secret -n platform database-credentials -o name"
    exitCode: 0
```

| Field | Required | Description |
|-------|----------|-------------|
| `run` | yes | Shell command executed via `sh -c`. |
| `exitCode` | no | Expected exit code. Default `0` (success). Set non-zero to assert the command must fail — useful for admission webhook rejection tests. |
| `outputContains` | no | The combined stdout+stderr must contain this substring. |
| `outputNotContains` | no | The combined stdout+stderr must not contain this substring. |
| `equals` | no | Output (trimmed) must exactly match this string. |
| `notEquals` | no | Output must not exactly match this string. |
| `greaterThan` | no | Output (trimmed, parsed as a number) must be greater than this value. |
| `lessThan` | no | Output (trimmed, parsed as a number) must be less than this value. |

---

## Per-expectation `onFailure`

A checkpoint can declare its own `onFailure:` block. When that specific checkpoint fails, its `onFailure` diagnostics run immediately — before the next checkpoint is evaluated. This lets you capture cluster state at the moment of failure, when it is most informative.

```yaml
expect:
  - name: Both probes reach Ready status
    after: cr-applied
    timeout: 120s
    kubectl:
      get:
        - kind: E2EProbe
          name: my-probe-server
          namespace: default
          field: .status.phase
          equals: Ready
    onFailure:
      kubectl:
        get:
          - kind: E2EProbe
            name: my-probe-server
            namespace: default
        describe:
          - kind: Deployment
            name: my-probe-server
            namespace: default
      commands:
        - kubectl get pods -n default -o wide
```

| Field | Description |
|-------|-------------|
| `kubectl` | Accepts the full `kubectl:` DSL (`get`, `logs`, `describe`, `events`, `exec`). Assertion fields are ignored — output is printed. |
| `commands` | List of shell strings run via `sh -c`. Output is printed to the terminal. |

The per-expectation `onFailure` is complementary to `spec.onFailure`. The difference:

| | When it runs |
|---|---|
| `expect[].onFailure` | Immediately after that checkpoint fails — cluster state reflects the failure context |
| `spec.onFailure` | Once at the end, after all expectations complete — useful for a global summary |

→ See [spec.onFailure in 01-spec.md](01-spec.md#speconfailure) for the spec-level variant.

---

## Composing expectations with `include:`

Large test suites can be split across files. An `include:` entry is replaced in place by the checkpoints in the referenced file — position in the list determines where the expanded checkpoints appear in the run order. Place an `include:` at the top for setup checks, in the middle for shared assertions, or at the end for cleanup. The file uses `expect:` as its root key, mirroring the field it slots into.

```yaml
# e2e.yaml
expect:
  - include: ./infra-ready.yaml     # expands here — runs first
  - name: Operator-specific check
    after: cr-applied
    timeout: 30s
    resources:
      - kind: MyOperator
        name: my-resource
  - include: ./cleanup.yaml         # expands here — runs last
```

```yaml
# infra-ready.yaml
expect:
  - name: CRD registered
    after: cr-applied
    timeout: 30s
    kubectl:
      get:
        - kind: CustomResourceDefinition
          name: myoperators.example.com
          field: status.conditions[0].type
          equals: Established
```

Paths are resolved relative to the `e2e.yaml` that contains the `include:` entry. Nested includes (a file that itself includes another) are not supported.

Use `ork validate` to confirm the expansion and see where each checkpoint landed:

```text
● my-operator-e2e

    CRD registered                   after: cr-applied    timeout: 30s
    Operator-specific check          after: cr-applied    timeout: 30s
    Cleanup verified                 after: cr-deleted    timeout: 60s

────────────────────────────────────────────────────────────
3 expectation(s) valid
```

The checkpoint list is the fully-expanded run order — what you see is what runs.

---

→ See [08-complete-example.md](08-complete-example.md) for a full E2E file exercising every subcommand.

→ Back: [02-setup.md](02-setup.md) | [Schema index](index.md)
