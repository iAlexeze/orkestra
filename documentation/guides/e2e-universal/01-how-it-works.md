# How It Works

Set `spec.custom.target: kubernetes` in any e2e file. This tells Orkestra: "you own the cluster and the assertions, but not the workload." Bundle generation and Orkestra runtime install are skipped. Everything else — cluster setup, setup manifests, the full assertion loop, and cleanup — runs unchanged.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: my-helm-chart-e2e

spec:
  custom:
    target: kubernetes

  setup:
    helm:
      - repo: https://charts.example.com
        chart: my-app
        namespace: my-app
        createNamespace: true
        version: v1.2.0
    wait:
      - kind: Deployment
        name: my-app
        namespace: my-app
        ready: true
        timeout: 120s

  cr: ./test-resource.yaml

  expect:
    - name: App is healthy
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          name: my-app
          namespace: my-app
          ready: true

    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: Deployment
          name: my-app
          namespace: my-app
          count: 0
```

Run it:

```bash
ork e2e -f e2e.yaml
```

An ephemeral kind cluster is created, your chart is installed into it, assertions run, and the cluster is torn down. Your actual cluster is never touched. Pass or fail. Minutes, not hours.

---

## Assertions

Assertions run in a polling loop until all pass or the checkpoint times out. Three blocks are available in any combination:

| Block | What it does |
|-------|-------------|
| `resources:` | Watch Kubernetes API state — existence, readiness, count |
| `commands:` | Run shell commands and assert stdout/stderr |
| `kubectl:` | Structured DSL — `get`, `logs`, `describe`, `exec`, `port-forward` |

The `kubectl:` block maps directly to kubectl subcommands you already know. Instead of constructing the command string yourself, you declare what you want to assert:

```yaml
kubectl:
  get:
    - kind: Deployment
      name: my-app
      field: .spec.replicas
      equals: "2"
  logs:
    - labelSelector: app=my-app
      outputContains: "server started"
  port-forward:
    - service: my-app
      port: 8080
      path: /healthz
      outputContains: ok
```

See the [kubectl block reference](../../reference/schema/04-e2e/07-kubectl.md) for all subcommands and fields.

---

## What this is not

`custom.target: kubernetes` does not add Orkestra runtime features to your workload. Orkestra is the test harness — cluster lifecycle, assertion polling, and cleanup. Your workload runs exactly as it would in production.

If you want Orkestra to manage your operator at runtime, start with `ork init --pack beginner`.

---

→ Next: [Use cases](02-use-cases.md)
