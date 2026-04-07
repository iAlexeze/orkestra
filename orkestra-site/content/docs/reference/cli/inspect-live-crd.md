---
title: "Inspect Live Crd"
weight: 82
---

# Orkestra Inspect Commands

The inspect commands let you observe and interact with Orkestra-managed
resources directly from the terminal. No need to construct `kubectl` commands
or know the API group of a CRD — just use the name you declared in the Katalog.

All inspect commands read from the cluster — they require `KUBECONFIG` or
`--kubeconfig` to be set.

---

## Commands

| Command | Description |
|---------|-------------|
| `ork get <crd>` | List all CRs of a CRD type |
| `ork describe <crd> <name>` | Describe a specific CR with spec, status, and events |
| `ork reconcile <crd> [name]` | Trigger reconciliation for one or all CRs |
| `ork reconcile all` | Trigger reconciliation for every Orkestra-managed resource |
| `ork status` | Show health and stats of the running operator |
| `ork events <crd> [name]` | Show Kubernetes events for a CRD type or specific CR |

---

## Cluster access

All inspect commands use the standard Kubernetes client resolution:

1. `--kubeconfig <path>` flag
2. `KUBECONFIG` environment variable
3. `~/.kube/config` (default)
4. In-cluster config (when running inside a pod)

```bash
# Explicit kubeconfig
ork get website --kubeconfig ~/.kube/production.yaml

# Environment variable
KUBECONFIG=~/.kube/production.yaml ork get website

# Default (~/.kube/config)
ork get website
```

---

## CRD name resolution

All inspect commands accept the CRD name in any of these forms:

```bash
ork get websites          # plural
ork get website           # singular
ork get Website           # Kind (case-insensitive)
```

If multiple CRDs match the name across different API groups, Orkestra
lists the candidates and asks you to be more specific:

```
"website" matches multiple CRDs — be more specific:
  websites.demo.orkestra.io (v1alpha1)
  websites.other.io (v1beta1)
```

---

## `ork get`

List all Custom Resources of a given CRD type.

```bash
ork get <crd> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --namespace` | List resources in a specific namespace |
| `-A, --all-namespaces` | List resources across all namespaces |

**Examples:**

```bash
# List all Website CRs
ork get websites

# Output:
# NAME           STATUS    AGE
# landing-page   Ready     2d
# my-api         Ready     5h
# my-blog        Ready     3d

# List across all namespaces
ork get website -A

# Output:
# NAMESPACE    NAME           STATUS    AGE
# default      landing-page   Ready     2d
# production   my-api         Ready     5h
# staging      my-blog        Ready     3d

# Cluster-scoped CRDs (namespace column omitted automatically)
ork get platformnamespace

# Output:
# NAME                   STATUS    AGE
# payments-production    Ready     7d
# payments-staging       Ready     7d
# platform-development   Ready     2h
```

**Status extraction:**

Orkestra inspects the CR's `.status` in this order:
- `status.phase` — e.g. `Ready`, `Pending`, `Failed`
- `status.state` — alternative phase field
- `status.conditions[0]` — e.g. `Ready=True`
- Falls back to `Unknown` if no status fields are present

---

## `ork describe`

Show the full details of a named CR — spec, status, and recent events.

```bash
ork describe <crd> <name> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --namespace` | Namespace of the resource |

**Example:**

```bash
ork describe website my-blog
```

```
── Metadata ──────────────────────────────────────────────────────────────────
Name:                my-blog
Namespace:           default
Kind:                Website
Group:               demo.orkestra.io
Version:             v1alpha1
Age:                 3d
Labels:
  app:               my-blog
  managed-by:        orkestra
Finalizers:
  - finalizer.demo.orkestra.io/website

── Spec ──────────────────────────────────────────────────────────────────────
image:               nginx:1.25
replicas:            2
port:                80
serviceType:         ClusterIP

── Status ────────────────────────────────────────────────────────────────────
phase:               Ready
readyReplicas:       2
message:             Deployment and Service created

── Events ────────────────────────────────────────────────────────────────────

TYPE       REASON           OBJECT             MESSAGE                         AGE
● Normal   Reconciled       default/my-blog    Successfully reconciled Web...  5m
● Normal   FinalizerAdded   default/my-blog    Added finalizers to Website...  3d
```

---

## `ork reconcile`

Trigger reconciliation by patching the `orkestra.konductor.io/reconcile-at`
annotation. The Orkestra informer detects the metadata change and re-queues
the object on the next reconcile loop.

This is non-destructive — only the annotation is changed, never the spec.

### Single CR

```bash
ork reconcile website my-blog
ork reconcile website my-blog -n production
```

```
Triggering reconcile for Website/my-blog...  ✓
```

### All CRs of a type

```bash
ork reconcile website
```

```
Triggering reconcile for all Website resources...

  default/landing-page                          ✓
  default/my-api                                ✓
  default/my-blog                               ✓
  production/my-api                             ✓

✓  Triggered 4 resources
```

### All Orkestra-managed resources

```bash
ork reconcile all
```

```
[1/3] Website
  default/landing-page                          ✓
  default/my-api                                ✓
  default/my-blog                               ✓
  Waiting 3s before next CRD...

[2/3] PlatformNamespace
  payments-production                           ✓
  payments-staging                              ✓
  Waiting 3s before next CRD...

[3/3] Application
  default/payments-app                          ✓

──────────────────────────────────────────────────
✓  Reconciled 6 resources across 3 CRDs
```

**Flags for `ork reconcile all`:**

| Flag | Description |
|------|-------------|
| `--sleep` | Pause between CRD types (default: `3s`) |
| `--dry-run` | Print what would be reconciled without making changes |

```bash
ork reconcile all --sleep 5s
ork reconcile all --dry-run
```

**When to use reconcile all:**

- After updating a Katalog and restarting the operator — to immediately
  reconcile all existing CRs rather than waiting for the next resync
- After a brief operator downtime — to catch up on any changes that
  accumulated during the outage
- During debugging — to force a fresh reconcile pass and observe the logs

---

## `ork status`

Show the health and reconcile statistics of a running Orkestra operator.
Connects to the operator's health API.

```bash
ork status [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--url` | Operator URL (default: `http://localhost:8080`) |
| `--timeout` | Request timeout (default: `5s`) |

**Port-forwarding a cluster deployment:**

```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system
ork status
```

**Example output:**

```
Orkestra Status
Operator:            my-platform-operator
Health:              healthy
CRDs:                5 total, 5 enabled
Uptime:              2d 4h 32m

CRD                    WORKERS   QUEUE   HEALTH   RECONCILES   ERR%   RESOURCES
website                2/2       0       ●        1,247        0.0%   3
platformnamespace      2/2       0       ●        412          0.0%   6
application            4/4       3       ●        8,891        0.2%   12
database               2/2       0       ●        201          0.0%   4
cache                  2/2       0       ●        89           0.0%   2
```

**Columns:**

- `WORKERS` — active workers / configured workers
- `QUEUE` — current workqueue depth (non-zero means backlog building up)
- `HEALTH` — green ● healthy / red ● degraded
- `RECONCILES` — total reconcile count since operator started
- `ERR%` — percentage of reconciles that returned an error
- `RESOURCES` — live CR count from the informer cache

**Connect to a remote operator:**

```bash
# Via Ingress
ork status --url https://orkestra.platform.myorg.io

# Via port-forward to a specific pod
kubectl port-forward pod/orkestra-7d9b4c8f6d-xkj2p 8080:8080 -n orkestra-system
ork status
```

---

## `ork events`

List Kubernetes events for a CRD type or a specific CR. Useful for debugging
reconcile failures without having to parse operator logs.

```bash
ork events <crd> [name] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-n, --namespace` | Namespace (namespaced CRDs only) |
| `--tail` | Number of most recent events to show (default: `25`, `0` = all) |

**Examples:**

```bash
# All events for all Website CRs
ork events website

# Events for a specific CR
ork events website my-blog

# Last 50 events
ork events website --tail 50

# All events (no limit)
ork events website --tail 0
```

**Example output:**

```
Events for all Website resources:

TYPE         REASON              OBJECT               MESSAGE                  AGE
● Normal     Reconciled          default/my-blog      Successfully reconci...  5m
● Normal     Reconciled          default/landing-...  Successfully reconci...  5m
● Warning    ReconcileError      default/my-api       failed to create dep...  12m
● Normal     FinalizerAdded      default/my-blog      Added finalizers to ...  3d
● Normal     Deleting            default/old-site     Deleting Website def...  7d
```

**Event types from Orkestra:**

| Event | Type | When |
|-------|------|------|
| `Reconciled` | Normal | Successful reconcile |
| `ReconcileError` | Warning | Reconcile returned an error |
| `FinalizerAdded` | Normal | Finalizer added to the CR |
| `FinalizerRemovalError` | Warning | Failed to remove finalizer during deletion |
| `Deleting` | Normal | CR deletion is in progress |
| `Deleted` | Normal | CR deletion completed |
| `FinalizerAdded` | Normal | Finalizer was added |

---

## Combining with kubectl

The inspect commands are designed to complement `kubectl`, not replace it.
Use Orkestra's inspect commands when you want to work with CRDs by name
without remembering their API groups. Use kubectl when you need advanced
filtering, patching, or output formats.

```bash
# Orkestra — quick overview by CRD name
ork get website

# kubectl — patch a specific field
kubectl patch website my-blog --type merge -p '{"spec":{"replicas":3}}'

# Orkestra — trigger immediate reconcile after the patch
ork reconcile website my-blog

# kubectl — watch the Deployment respond
kubectl get deployment my-blog -w
```

---

## Kubeconfig for Kubernetes deployments

When Orkestra itself is deployed in the cluster and you are running `ork`
commands from your local machine, you need a kubeconfig with access to the
cluster. The inspect commands use the same resolution as `kubectl`.

```bash
# Use the cluster's kubeconfig
export KUBECONFIG=~/.kube/production.yaml

# Or per-command
ork get website --kubeconfig ~/.kube/production.yaml

# Or via kubectl context
kubectl config use-context my-production-cluster
ork get website
```
