---
title: "Installation"
weight: 1
description: "Install the Orkestra CLI and get your cluster ready."
---

## Prerequisites

Before installing Orkestra, make sure you have the following:

- **Kubernetes cluster** — v1.25 or later (local: [kind](https://kind.sigs.k8s.io/), [k3d](https://k3d.io/), or [minikube](https://minikube.sigs.k8s.io/))
- **kubectl** — configured and pointing at your cluster
- **Go 1.21+** — only needed to build from source; not needed to use Orkestra

{{< callout type="tip" title="Quick cluster setup" >}}
If you don't have a cluster, spin one up locally with:

```bash
kind create cluster --name orkestra-dev
```
{{< /callout >}}

## Install the CLI

### Using `go install` (recommended)

```bash
go install github.com/iAlexeze/orkestra/cmd/ork@latest
```

Verify the installation:

```bash
ork version
```

Expected output:

```
Orkestra v1.0.0  (commit: abc1234, built: 2025-01-01)
```

### Download a binary release

Pre-built binaries are available for Linux, macOS, and Windows on the [GitHub Releases](https://github.com/iAlexeze/orkestra/releases) page.

**Linux (amd64)**

```bash
curl -sSL https://github.com/iAlexeze/orkestra/releases/latest/download/ork_linux_amd64 \
  -o /usr/local/bin/ork && chmod +x /usr/local/bin/ork
```

**macOS (arm64)**

```bash
curl -sSL https://github.com/iAlexeze/orkestra/releases/latest/download/ork_darwin_arm64 \
  -o /usr/local/bin/ork && chmod +x /usr/local/bin/ork
```

**macOS (amd64)**

```bash
curl -sSL https://github.com/iAlexeze/orkestra/releases/latest/download/ork_darwin_amd64 \
  -o /usr/local/bin/ork && chmod +x /usr/local/bin/ork
```

### Build from source

```bash
git clone https://github.com/iAlexeze/orkestra.git
cd orkestra
make build
sudo mv bin/ork /usr/local/bin/ork
```

## Install the Orkestra runtime

The Orkestra runtime runs in-cluster and handles CRD registration, reconciliation, and the Control Center UI.

### Install via kubectl

```bash
kubectl apply -f https://github.com/iAlexeze/orkestra/releases/latest/download/install.yaml
```

This creates the `orkestra-system` namespace and deploys:

| Component | Purpose |
|---|---|
| `orkestra-controller` | Core reconciliation engine |
| `orkestra-apiserver` | Custom API extensions |
| `control-center` | Web UI for observability |
| `orkestra-webhook` | Validation & mutation webhooks |

### Verify the installation

```bash
kubectl get pods -n orkestra-system
```

All pods should be `Running` within about 30 seconds:

```
NAME                                    READY   STATUS    RESTARTS   AGE
orkestra-controller-7d8f9b6c4-x2pkj    1/1     Running   0          42s
orkestra-apiserver-5c6d7f8b9-m3nkl     1/1     Running   0          42s
control-center-6b7c8d9e0-p4qrs         1/1     Running   0          42s
```

{{< callout type="info" title="Namespace" >}}
All Orkestra system components run in the `orkestra-system` namespace. Your CRDs and operators can live in any namespace.
{{< /callout >}}

## Access Control Center

Control Center is the built-in observability dashboard. To access it locally:

```bash
kubectl port-forward -n orkestra-system svc/control-center 8080:80
```

Then open [http://localhost:8080](http://localhost:8080) in your browser.

## Uninstall

To remove Orkestra from your cluster:

```bash
kubectl delete -f https://github.com/iAlexeze/orkestra/releases/latest/download/install.yaml
```

{{< callout type="warning" title="CRD cleanup" >}}
Uninstalling Orkestra does **not** automatically delete your custom CRDs or any custom resources. You must delete those separately if desired.
{{< /callout >}}

## Next steps

With Orkestra installed, you're ready to write your first operator:

- [Quick Start](/docs/getting-started/quickstart/) — build a working operator in 5 minutes
- [Katalog Basics](/docs/basics/) — understand the core manifest structure
- [Examples](https://github.com/iAlexeze/orkestra/tree/main/examples) — real-world examples on GitHub
