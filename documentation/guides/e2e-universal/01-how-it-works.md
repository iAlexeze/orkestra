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

## What this is not

`custom.target: kubernetes` does not add Orkestra runtime features to your workload. Orkestra is the test harness — cluster lifecycle, assertion polling, and cleanup. Your workload runs exactly as it would in production.

If you want Orkestra to manage your operator at runtime, start with `ork init --pack beginner`.

---

→ Next: [Use cases](02-use-cases.md)
