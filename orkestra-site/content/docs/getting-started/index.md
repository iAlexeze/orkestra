---
title: "Index"
weight: 24
---

# Getting Started

{{< callout type="tip" >}}
This guide gets you from zero to a running operator in under five minutes.
If you want to understand the concepts before diving in, read
[Why Orkestra](../blog/why-orkestra.md) first.
{{< /callout >}}

---

## Requirements

* A running Kubernetes cluster (local or remote)
* [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) configured and pointing at your cluster
* A valid `kubeconfig` (default location `~/.kube/config`)

!!! "kubernetes distribution"
    Orkestra works with any Kubernetes distribution — kind, minikube, k3s,
    EKS, GKE, AKS. No cluster-level changes are required before installing.

---

## Install Orkestra

### macOS (Homebrew)

```bash
brew tap orkspace/tap
brew install ork
```

### Linux / macOS (curl)

```bash
curl -sSL https://get.orkestra.sh | bash
```

### Options

```bash
# Pin to a specific version
curl -sSL https://get.orkestra.sh | ORK_VERSION=v1.0.0 bash

# Install to a custom directory
curl -sSL https://get.orkestra.sh | ORK_INSTALL_DIR=~/.local/bin bash
```

### Verify the binary (recommended)

{{< callout type="note" title="Security first" >}}
Every Orkestra release is GPG-signed. Verifying before running is good
practice, especially in CI or shared environments.
{{< /callout >}}

```bash
# Import the Orkestra public key (once)
curl -sSL https://github.com/orkspace/orkestra/releases/download/v1.0.0/orkestra-public-key.asc \
  | gpg --import

# Download binary and signature
curl -sSLO https://github.com/orkspace/orkestra/releases/download/v1.0.0/ork_linux_amd64.tar.gz
curl -sSLO https://github.com/orkspace/orkestra/releases/download/v1.0.0/ork_linux_amd64.tar.gz.asc

# Verify
gpg --verify ork_linux_amd64.tar.gz.asc ork_linux_amd64.tar.gz
# gpg: Good signature from "Orkestra Releases <releases@orkestra.io>"
```

### Confirm installation

```bash
ork version
```

---

## The Mental Model

Before building your first operator, it helps to understand what Orkestra
actually does.

```
CRD → Katalog → Orkestra → Kubernetes
```

* **CRD** — defines what your resource is (the schema). Kubernetes handles storage and validation.
* **Katalog** — defines how it should behave (the logic). This is your operator, in YAML.
* **Orkestra** — reconciles it. Watches CRs, resolves templates, calls the registry.
* **Kubernetes** — stores it and notifies Orkestra when things change.

{{< callout type="note" >}}
Orkestra operates on unstructured CRDs — the same `map[string]interface{}`
representation Kubernetes uses internally. You do not need Go types, code
generation, or scheme registration for the common case.
{{< /callout >}}

---

## Your First Operator

We will build an operator for a `Website` CRD. When a `Website` CR is
applied, Orkestra will create a Deployment and a Service for it — and keep
them in sync whenever the CR changes.

### Step 1 — Scaffold the project

```bash
ork init my-operator
cd my-operator
```

{{< callout type="tip" title="ork init" >}}
`ork init` creates a workspace with the Website example pre-configured.
Open `examples/website/website-katalog.yaml` to see what you are about
to run.
{{< /callout >}}

### Step 2 — Apply the CRD

```bash
kubectl apply -f examples/website/website-crd.yaml
```

This installs the `Website` CRD into your cluster. Orkestra does not
install CRDs — it manages the resources that CRs create.

{{< callout type="note" >}}
The CRD only needs to be applied once. After that, `kubectl get websites`
will work in any namespace.
{{< /callout >}}

---
### Step 3 — Start Orkestra

```bash
ork run --katalog examples/website/website-katalog.yaml
```

Orkestra starts, registers its informer for `Website` CRs, and waits. You
will see the health server come up and the leader election complete.

{{< callout type="tip" >}}
Open a second terminal and run `ork status` to see the live state of the
operator. You can also visit `localhost:8080/katalog/website` in a browser.
{{< /callout >}}

### Step 4 — Apply a CR

In a new terminal:

```bash
kubectl apply -f examples/website/website-cr.yaml
```

Watch Orkestra's output in the first terminal. You will see the reconcile
event arrive, templates resolve, and child resources created.

### Step 5 — Verify the results

```bash
kubectl get deployments
kubectl get services
```

A Deployment and Service named after your `Website` CR should appear.
Orkestra set owner references on both — deleting the `Website` CR will
cascade-delete them automatically.

{{< callout type="warning" >}}
Do not delete the child Deployment or Service manually to test drift
correction until you have run at least one successful reconcile.
Orkestra detects drift on the next reconcile cycle (configured via
`resync`) — not immediately.
{{< /callout >}}

### Step 6 — Explore the built-in endpoints

Orkestra exposes a health and observability API with no configuration:

```bash
# Is this CRD healthy?
curl localhost:8080/katalog/website/health | jq

# Full CRD info — workers, queue, reconcile stats
curl localhost:8080/katalog/website | jq

# All managed CRDs
curl localhost:8080/katalog | jq

# Prometheus metrics
curl localhost:8080/metrics
```

---

## What Just Happened?

When you applied the `Website` CR, Orkestra executed this sequence:

1. The API server notified Orkestra's informer of a new object
2. The object was enqueued in the `Website` workqueue
3. A worker picked it up and read the CR from the informer cache (zero API cost)
4. Finalizers were added to protect the CR from dirty deletion
5. Template expressions were resolved — `{{ .spec.image }}` became the actual image value
6. The Deployment and Service were created with owner references pointing to the CR
7. Both resources were marked `reconcile: true` — drift will be corrected on every cycle
8. A `Reconciled` Kubernetes event was emitted on the CR
9. Metrics were incremented: `controller_reconcile_total{result="success"}`

All of that from a Katalog entry and a CR. No code written.

You can see the events Orkestra emits by running:

```bash
kubectl describe website <name>
```

Look for the Events section at the bottom.

---

## Cleanup

```bash
# Delete the CR — cascades to Deployment and Service
kubectl delete -f examples/website/website-cr.yaml

# Stop Orkestra (Ctrl+C in the terminal running ork run)

# Remove the CRD if you no longer need it
kubectl delete -f examples/website/website-crd.yaml
```

{{< callout type="warning" title="safe cleanup" >}}
Deleting the `Website` CRD while CRs still exist will force-delete all
`Website` objects without running finalizers. Always delete CRs before
deleting their CRD in production.
{{< /callout >}}

---

## CLI Reference

| Command | Description |
|---------|-------------|
| `ork init <name>` | Scaffold a new operator project |
| `ork validate --katalog <path>` | Validate a Katalog or Komposer |
| `ork template --katalog <path>` | Preview merged, validated state |
| `ork template --katalog <path> --graph` | Show dependency graph |
| `ork run --katalog <path>` | Start the operator runtime |
| `ork status` | Show live health of all managed CRDs |
| `ork get <crd>` | List CRs of a given type |
| `ork describe <crd> <name>` | Describe a specific CR |
| `ork events <crd>` | Show Kubernetes events for a CRD |
| `ork version` | Print version information |

---

## Troubleshooting

**The informer is not seeing my CR**

Check that the CRD is installed and the `apiTypes` block in the Katalog
matches the group, version, kind, and plural exactly. Run:

```bash
ork validate --katalog <path>
```

**Orkestra starts but no resources are created**

Confirm the CR was applied and the informer has synced. Check
`localhost:8080/katalog/website` — `resourceCount` should be non-zero.
Also check `kubectl describe website <name>` for events.

**`ork status` cannot connect**

`ork status` expects the operator at `localhost:8080`. If Orkestra is
deployed in a cluster, port-forward first:

```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system
ork status
```

{{< callout type="tip" title="katalog validation" >}}
Run `ork validate --katalog <path>` before `ork run` to catch
configuration errors before they surface at runtime. It checks
apiTypes, enriches built-in Kinds, and validates the dependency graph.
{{< /callout >}}

---

## Next Steps

You have a working operator. Here is where to go next:

* **[Writing Your First Katalog](./writing-your-first-katalog.md)** — define your own CRDs and templates
* **[Komposer](../runtime-manual/concepts/komposer.md)** — compose Katalogs from files, Helm charts, and registries
* **[Deployment Guide](../guides/user-guide/deployment.md)** — run Orkestra in a cluster with Helm
* **[CLI Reference](../reference/cli/index.md)** — full documentation for every command
* **[Use Cases](../use-cases/index.md)** — real-world operator patterns

