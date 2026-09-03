# Writing a test

If you have not yet written an E2E from scratch, start with [Writing Your First E2E](../../getting-started/06-writing-your-first-e2e.md). It walks through every step — file layout, validate, run, and debugging a failing checkpoint.

This page is a quick reference for what each field does and the most common patterns.

---

## Minimum viable E2E

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E

metadata:
  name: my-operator-e2e

spec:
  katalog: ./katalog.yaml
  crd:     ./crd.yaml
  cr:      ./cr.yaml

  expect:
    - name: Deployment ready
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          namespace: default
          ready: true
```

Four required fields. Everything else is optional.

---

## Field quick reference

| Field | Required | What it controls |
|-------|----------|-----------------|
| `spec.katalog` | yes | Katalog under test |
| `spec.crd` | yes | CRD to install before the operator starts |
| `spec.cr` | yes | CR to apply at the `cr-applied` phase |
| `spec.cluster` | no | Cluster name and provider. Defaults to `kind` / `ork-e2e` |
| `spec.setup` | no | Prerequisite files, charts, and wait conditions |
| `spec.expect` | yes | Ordered list of checkpoints |
| `imports` | no | Other E2E files to run after this one |

→ [Full spec reference](../../reference/schema/04-e2e/01-spec.md)

---

## Assertions

**Resource existence:**
```yaml
resources:
  - kind: Deployment
    name: hello-website
    namespace: default
```
Passes when the resource exists. Polled until timeout.

**Resource readiness:**
```yaml
resources:
  - kind: Deployment
    namespace: default
    ready: true
```
Passes when `availableReplicas == replicas`. Omit `name` to match any Deployment in the namespace.

**Cleanup assertion (`count: 0`):**
```yaml
resources:
  - kind: Deployment
    name: hello-website
    namespace: default
    count: 0
```
Passes when the resource does not exist. Use in `after: cr-deleted` checkpoints to verify finalizer cleanup ran.

**Command assertion:**
```yaml
commands:
  - run: "kubectl get secret -n platform db-creds -o name"
    exitCode: 0
  - run: "kubectl exec -n default deploy/api-server -- wget -qO- localhost/health"
    outputContains: "ok"
```
Run alongside resource checks in the same polling loop. `outputContains` checks combined stdout+stderr.

---

## Lifecycle triggers

Every checkpoint has an `after:` field:

| Value | When |
|-------|------|
| `cr-applied` | CR has been applied. The reconciler has started. Resources may not yet exist. |
| `cr-deleted` | CR has been deleted. Finalizers have run. Child resources should be cleaning up. |

The CR is applied or deleted exactly once — when the first checkpoint with that `after:` value is reached. All subsequent checkpoints with the same `after:` use the same lifecycle event.

---

## Annotated example — once: semantics

Testing that a Secret is created on first apply but not recreated on second apply:

```yaml
spec:
  katalog: ./katalog.yaml
  crd:     ./crd.yaml
  cr:      ./cr.yaml

  expect:
    - name: Secret created on first apply
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Secret
          name: app-creds
          namespace: default

    - name: Secret still exists after re-apply
      after: cr-applied
      timeout: 10s
      commands:
        - run: >
            kubectl apply -f cr.yaml &&
            kubectl get secret -n default app-creds -o jsonpath='{.metadata.uid}'
          outputContains: ""   # any uid means the secret survived

    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: Secret
          name: app-creds
          namespace: default
          count: 0
```

---

## Validate before running

```bash
ork validate -f e2e.yaml
```

Catches missing files, unknown `after` values, empty checkpoints, and invalid imports — before touching a cluster. Always validate first.

---

→ Next: [Suites and imports](03-suite-and-imports.md)
