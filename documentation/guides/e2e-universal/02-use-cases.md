# Use Cases

## Helm charts

Ship an `e2e.yaml` alongside your chart. Use `setup.helm` with `chart: ./` to install
the chart itself, add per-entry `wait:` to confirm it is ready, then assert the
resources it creates. No `crd:` or `cr:` needed — everything goes through setup.

> **Real example:** Orkestra dogfoods this pattern for its own Helm chart —
> [`charts/orkestra/e2e.yaml`](https://github.com/orkspace/orkestra/blob/main/charts/orkestra/e2e.yaml)

All expectations use `after: setup-complete` (or omit `after:`, which means the same
thing) since there is no CR lifecycle to wait for.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    helm:
      - chart: ./
        release: my-app
        namespace: my-app
        createNamespace: true
        values:
          replicaCount: 2
        wait:
          - kind: Deployment
            name: my-app
            namespace: my-app
            ready: true
            timeout: 120s
  expect:
    - name: Deployment is ready
      timeout: 30s
      resources:
        - kind: Deployment
          name: my-app
          namespace: my-app
          ready: true
    - name: Service is created
      timeout: 30s
      resources:
        - kind: Service
          name: my-app
          namespace: my-app
```

When you need manifests applied before the chart (CRDs, namespaces, config), use
`setup.apply` with per-entry waits to confirm each step before moving on:

```yaml
setup:
  apply:
    - path: ./fixtures/crd.yaml
      wait:
        - kind: CustomResourceDefinition
          name: myresources.example.com
          timeout: 30s
    - path: ./fixtures/config.yaml
  helm:
    - chart: ./
      release: my-app
      namespace: my-app
      createNamespace: true
      wait:
        - kind: Deployment
          name: my-app
          namespace: my-app
          ready: true
          timeout: 120s
```

---

## Third-party operators

Install an operator you did not write — cert-manager, ArgoCD, Crossplane, FluxCD —
and assert the resources it manages. Apply the CRD and CR via `setup.apply` so the
operator sees the CR on first boot, then use `after: setup-complete` for infrastructure
checks.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    apply:
      - path: ./fixtures/crd.yaml
        wait:
          - kind: CustomResourceDefinition
            name: certificates.cert-manager.io
            timeout: 30s
      - ./fixtures/cr-certificate.yaml
    helm:
      - repo: https://charts.jetstack.io
        chart: cert-manager
        namespace: cert-manager
        createNamespace: true
        version: v1.14.0
        values:
          installCRDs: true
        wait:
          - kind: Deployment
            name: cert-manager-webhook
            namespace: cert-manager
            ready: true
            timeout: 120s
  expect:
    - name: TLS Secret issued
      timeout: 60s
      resources:
        - kind: Secret
          name: my-tls-secret
          namespace: default
```

---

## Structured assertions with kubectl DSL

The `kubectl:` block replaces ad-hoc shell commands with structured subcommands that map directly to how people already use kubectl. It supports `get`, `logs`, `describe`, `exec`, and `port-forward`. All assertions run in the same polling loop as `resources:`.

```yaml
expect:
  - name: Deployment is healthy and correctly configured
    after: cr-applied
    timeout: 90s

    resources:
      - kind: Deployment
        name: my-service
        namespace: default
        ready: true

    kubectl:
      # assert a field value
      get:
        - kind: Deployment
          name: my-service
          field: .spec.template.spec.containers[0].resources.requests.cpu
          equals: 200m

      # assert a log line appeared
      logs:
        - labelSelector: app=my-service
          namespace: default
          since: 30s
          outputContains: "server started"
          outputNotContains: FATAL

      # assert HTTP endpoint via port-forward + curl (auto-installed if missing)
      port-forward:
        - service: my-service
          namespace: default
          port: 8080
          path: /healthz
          outputContains: ok
```

See [kubectl block reference](../../reference/schema/04-e2e/07-kubectl.md) for all fields and subcommands.

---

## Platform stacks

Install multiple tools together and assert they interact correctly. Per-entry `wait:`
on each helm install ensures the stack comes up in order.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    helm:
      - repo: https://charts.jetstack.io
        chart: cert-manager
        namespace: cert-manager
        createNamespace: true
        values:
          installCRDs: true
        wait:
          - kind: Deployment
            name: cert-manager-webhook
            namespace: cert-manager
            ready: true
            timeout: 120s
      - repo: https://argoproj.github.io/argo-helm
        chart: argo-cd
        namespace: argocd
        createNamespace: true
        wait:
          - kind: Deployment
            name: argocd-server
            namespace: argocd
            ready: true
            timeout: 180s
  expect:
    - name: ArgoCD Application synced
      timeout: 60s
      commands:
        - run: kubectl get application my-app -n argocd -o jsonpath='{.status.sync.status}'
          outputContains: Synced
    - name: TLS Secret issued
      timeout: 60s
      resources:
        - kind: Secret
          name: my-app-tls
          namespace: default
```

---

## Lifecycle events quick reference

| `after:` value | When to use |
|----------------|-------------|
| `setup-complete` (default) | Non-operator tests, infrastructure checks, anything without a CR lifecycle |
| `cr-applied` | After `spec.cr` is applied — operator reconciliation assertions |
| `cr-deleted` | After `spec.cr` is deleted — cleanup assertions |

Omitting `after:` is the same as writing `after: setup-complete`.

---

→ Next: [CI integration](03-ci.md)
