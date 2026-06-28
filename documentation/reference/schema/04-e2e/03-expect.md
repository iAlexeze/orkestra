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
| `name` | yes | Printed in the results table. |
| `after` | yes | Lifecycle phase that must have occurred. |
| `timeout` | yes | Maximum wait time (Go duration: `30s`, `2m`, `90s`). |
| `resources` | no | Resource state assertions, polled until passing. |
| `commands` | no | Shell command assertions, run in the same polling loop. |
| `kubectl` | no | Structured kubectl subcommand assertions. See [kubectl block](07-kubectl.md). |

---

## `after`

| Value | When it triggers |
|-------|-----------------|
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

---

## Full example — secret distribution

```yaml
expect:
  - name: CR created
    after: cr-applied
    timeout: 60s
    resources:
      - kind: SecretDistribution
        name: db-creds

  - name: Secret distributed to team-alpha
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Secret
        name: database-credentials
        namespace: team-alpha

  - name: Cleanup verified
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: SecretDistribution
        name: db-creds
        count: 0
      - kind: Secret
        name: database-credentials
        namespace: team-alpha
        count: 0
```

---

→ Back: [02-setup.md](02-setup.md) | [Schema index](index.md)
