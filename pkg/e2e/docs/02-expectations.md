# Expectations

Each entry in `spec.expect` is an expectation — a named assertion that must pass within a timeout.

## Triggers

| `after` value | When the check runs |
|--------------|---------------------|
| `cr-applied` | After `cr.yaml` is applied to the cluster |
| `cr-deleted` | After the CR is deleted from the cluster |

The CR is applied exactly once (on the first `cr-applied` expectation) and deleted exactly once (on the first `cr-deleted` expectation). All subsequent expectations with the same `after` value reuse that state.

## Resource checks

Assert that a Kubernetes resource exists (or doesn't):

```yaml
resources:
  - kind: Deployment        # any kind — built-in or custom
    name: my-deploy         # omit to match any resource of this kind
    namespace: default      # omit for cluster-scoped resources
    ready: true             # optional — checks availableReplicas > 0
    count: 0                # optional — assert exact count (0 = must not exist)
```

**Checking that the CR itself was created** is recommended as the first expectation — it confirms the CRD is registered and working before checking child resources:

```yaml
- name: CR created
  after: cr-applied
  timeout: 60s
  resources:
    - kind: MyApp
      name: my-cr
      namespace: default
```

**Checking cleanup** should include both the CR and all child resources:

```yaml
- name: Cleanup verified
  after: cr-deleted
  timeout: 30s
  resources:
    - kind: MyApp
      name: my-cr
      namespace: default
      count: 0
    - kind: Deployment
      name: my-deploy
      namespace: default
      count: 0
```

## Command checks

Run arbitrary shell commands and assert exit code or output:

```yaml
commands:
  - run: kubectl delete crd apps.security.orkestra.io
    exitCode: 1
    outputContains: "denied the request"
```

## Polling

The verifier polls every 3 seconds until all conditions in an expectation pass or the timeout expires. A `kubectl` error on a `count: 0` check is treated as "nothing exists" and passes — this handles the window between CR deletion and CRD deregistration.
