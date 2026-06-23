# Test Anything That Runs in Kubernetes

`ork e2e` is a Kubernetes behavioural verification tool. It spins up a real cluster,
installs things, watches the API server, and asserts GVK state. Nothing in that
pipeline requires the workload under test to be an Orkestra operator.

A Helm chart installs into the same API server. A `kubectl apply` produces the same
objects. The assertion layer does not care about the installation mechanism. It watches
GVKs. A `Deployment/my-app` is the same object whether it was created by an Orkestra
operator, a Helm chart, a controller-runtime reconciler, or `kubectl apply`.

---

## The insight

Today there is no way to look at a container image, a deployment manifest, or a Helm
chart and say it works. Signatures prove ownership. They say nothing about behaviour.
A signed chart with a broken init container is still signed. A signed operator that
silently drops events under load is still signed.

`ork e2e` built the verification layer in the hardest context — operators with
long-lived state, drift correction, complex status, and partial failure modes. If you
can prove an operator works correctly before distribution, you can prove a Helm chart
works. The machinery is the same. The e2e file is the contract. The cluster is the
witness.

---

## Set `custom.target: kubernetes`

Any e2e file with `spec.custom.target: kubernetes` tells Orkestra: "you own the
cluster and the assertions, but not the workload". Bundle generation and Orkestra's
own helm install/uninstall are skipped. Everything else — cluster setup, CRD apply,
setup manifests, CR apply, the full assertion loop, and cleanup — runs unchanged.

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

An ephemeral kind cluster is created, your chart is installed into it, assertions run,
and the cluster is torn down. Your actual cluster is never touched. Pass or fail.
Minutes, not hours.

---

## Use cases

### Helm charts

Write an `e2e.yaml` alongside your chart. Use `setup.helm` to install the chart
itself, then assert the resources it creates.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    helm:
      - chart: ./         # the chart itself
        namespace: default
    wait:
      - kind: Deployment
        name: my-app
        ready: true
        timeout: 120s
  expect:
    - name: App is healthy
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          name: my-app
          namespace: default
```

### Third-party operators

Install an operator you did not write — cert-manager, ArgoCD, Crossplane, FluxCD —
and assert the resources it manages.

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
        version: v1.14.0
        values:
          installCRDs: true
    wait:
      - kind: Deployment
        name: cert-manager-webhook
        namespace: cert-manager
        ready: true
        timeout: 120s
  cr: ./certificate-cr.yaml
  expect:
    - name: TLS Secret issued
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Secret
          name: my-tls-secret
          namespace: default
```

### Platform stacks

Install multiple tools together and assert they interact correctly.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    helm:
      - repo: https://argoproj.github.io/argo-helm
        chart: argo-cd
        namespace: argocd
        createNamespace: true
      - repo: https://charts.jetstack.io
        chart: cert-manager
        namespace: cert-manager
        createNamespace: true
        values:
          installCRDs: true
  cr: ./platform-resources.yaml
  expect:
    - name: ArgoCD Application synced
      after: cr-applied
      timeout: 180s
      commands:
        - run: kubectl get application my-app -n argocd -o jsonpath='{.status.sync.status}'
          outputContains: Synced
    - name: TLS Secret issued
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Secret
          name: my-app-tls
          namespace: default
```

---

## The GitHub Actions bridge

Add `e2e.yaml` to any repository and one step to any workflow:

```yaml
# .github/workflows/ci.yml
- uses: orkspace/orkestra-action@v1   # https://github.com/orkspace/orkestra-action
  with:
    validate: true
    e2e: true          # runs ork e2e before anything else
    # or point at an explicit file:
    e2e: ./e2e.yaml
```

The action installs `ork`, validates the spec, spins up an ephemeral kind cluster,
runs the assertions, and tears everything down. Your real cluster is never touched.
No other tooling required.

The path to "every published artifact has a verification badge" is: add `e2e.yaml`,
add one step to the workflow, done.

---

## What this is not

`custom.target: kubernetes` does not add Orkestra runtime features to your workload.
Orkestra is the test harness — the cluster lifecycle, assertion polling, and cleanup.
Your workload runs exactly as it would in production.

If you want Orkestra to manage your operator at runtime, start with
`ork init --pack from-controller-runtime`.

---

→ Schema reference: [spec.custom](../reference/schema/04-e2e/05-custom-target.md)
→ Example pack: `use-cases/custom-operator`
