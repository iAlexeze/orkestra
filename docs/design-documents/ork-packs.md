# Design Document: Unified Example Makefile & Advanced Capability Packs

## Problem Statement

Orkestra examples currently rely on scattered `cleanup.sh` scripts and manual Kind cluster creation. This is inconsistent and makes advanced scenarios (multi‑cluster, dependency ordering, autoscaling) hard to demonstrate. We need a unified, maintainable way to:

- Spin up Kind clusters (single or multiple)
- Clean up resources reliably
- Demonstrate cross‑operator communication, dependencies, and autoscaling in a hands‑on manner

## Solution Overview

- **Single source `Makefile`** placed in the root `examples/` directory.
- During `ork init` baking, this `Makefile` is **copied into every generated pack** (beginner, advanced, security, use‑cases, etc.) to provide consistent targets.
- A helper script `examples/setup-kind.sh` (currently living in scripts/setup-kind.sh) is also copied to manage cluster lifecycle.
- Example READMEs now reference `make cluster`, `make clean`, etc., instead of `chmod +x cleanup.sh`.
- New advanced packs (cross‑operator communication, dependencies, autoscaling) use `make cluster` to create multiple clusters where required.

## Design Goals

1. **Consistency** – Every example uses the same commands to create/destroy clusters.
2. **Maintainability** – Changes to cluster setup are made in one place.
3. **Self‑contained** – Each example remains runnable without external scripts (the Makefile includes all needed targets).
4. **Demo‑ready** – Advanced packs can spin up two Kind clusters, install Orkestra, and run cross‑cluster tests with a single `make` invocation.
5. **Educational** – The Makefile targets are simple enough to understand, showing how Orkestra works in multi‑cluster environments.

## Components

### 1. `examples/Makefile` (template)

- **`make cluster`** – Creates a Kind cluster named `orkestra-playground` (or uses `CLUSTER_NAME` env var).  
  Checks for Kind; if not installed, exits with a helpful message.
- **`make clean`** – Deletes the Kind cluster and removes all local temporary files.
- **`make test`** – Runs `ork validate` on the example’s `katalog.yaml`.
- **`make e2e`** – Runs `ork e2e` using the example’s `e2e.yml` (if present).
- **`make multi-cluster`** (advanced) – Creates two clusters (`cluster-a`, `cluster-b`) for cross‑cluster examples.

### 2. `examples/setup-kind.sh` (helper script)

- Handles `create`, `delete`, and `list` operations.
- Used by the Makefile to avoid repetition.

### 3. Integration with `ork init`

When `ork init my-operator --pack beginner` runs, the CLI copies:

- The pack’s example files (`katalog.yaml`, `crd.yaml`, etc.)
- The shared `Makefile` (from `examples/Makefile`) into the pack’s root.
- The shared `setup-kind.sh` into `examples/` (relative path) or directly into the pack.

The result: every generated project has a working `Makefile` that matches the user’s environment.

## New Advanced Packs

To demonstrate Orkestra’s deeper capabilities, we introduce three new pack families. Each will be placed under `examples/advanced/cross-operator/`, `examples/advanced/dependencies/`, `examples/advanced/autoscale/`.

### A. Cross‑Operator Communication Pack

Shows how one CRD can read the status of another CRD (or of resources created by it) and react accordingly.

| Example | Description | Cluster Setup |
|---------|-------------|----------------|
| `01-in-binary` | Two CRDs (`Producer`, `Consumer`) in the same Orkestra runtime. Consumer waits for Producer’s `status.endpoint` before creating its own child resources. | Single cluster, single runtime |
| `02-cross-binary` | Same logic, but Producer and Consumer run in **different Orkestra deployments** (different namespaces, still same cluster). Demonstrates cross‑binary communication via HTTP API. | Single cluster, two Orkestra runtimes |
| `03-cross-cluster` | Producer in cluster‑A, Consumer in cluster‑B. Consumer uses the Producer’s external endpoint (e.g., via Ingress or LoadBalancer). | Two Kind clusters (`make multi-cluster`) |

**Makefile targets:**  
- `make cluster` (single) for 01  
- `make dual-cluster` for 03 (creates two clusters)

### B. Dependency Pack

Shows startup ordering and condition‑based readiness.

| Example | Description |
|---------|-------------|
| `01-in-binary` | CRD `App` declares `dependsOn: Database`. Orkestra starts reconciling `App` only after `Database` is healthy. |
| `02-cross-binary` | `Database` runs in a separate Orkestra deployment (e.g., security‑sensitive). `App` waits for a remote health check. |
| `03-cross-cluster` | `Database` runs in a different cluster; `App` polls its external status endpoint. |

### C. Autoscale Pack

Shows how Orkestra can scale reconcile workers based on metrics.

| Example | Description |
|---------|-------------|
| `01-based-on-own-metrics` | CRD scales its own workers when queue depth exceeds a threshold. |
| `02-sibling-in-binary` | CRD `A` scales based on the queue depth of CRD `B` (same runtime). |
| `03-sibling-in-cluster` | CRD `A` scales based on metrics from a sibling in the same cluster but a different Orkestra deployment. |
| `04-external-friend` | CRD scales based on metrics from a resource in another cluster (e.g., a Prometheus federation or another Orkestra operator). |

For these examples, the Makefile will also start a Prometheus instance (in Kind) to feed metrics.

## How This Exposes Orkestra’s True Capabilities

| Capability | Demonstration |
|------------|----------------|
| **Native cross‑CRD communication** | `cross-operator/01-in-binary` shows `status` fields being read in real time. |
| **Distributed operator communication** | `cross-operator/02-cross-binary` shows the HTTP API surfaces (`/katalog/crd/…`). |
| **Multi‑cluster operation** | `cross-operator/03-cross-cluster` proves Orkestra can work across cluster boundaries. |
| **Dependency ordering without Go** | `dependencies/01-in-binary` uses declarative `dependsOn` in the Katalog. |
| **Autoscaling at the controller level** | `autoscale/*` shows worker pools scaling dynamically – a unique feature to Orkestra. |

By packaging these as runnable examples, users no longer need to read verbose documentation. They just `make cluster`, `make e2e`, and see Orkestra’s advanced behaviour in action.

## Implementation Roadmap

| Phase | Deliverable | Effort |
|-------|-------------|--------|
| 1 | Create `examples/Makefile` and `setup-kind.sh`; update `ork init` to copy them. | 1 week |
| 2 | Convert existing beginner/advanced packs to use `make cluster` and `make clean` (remove `cleanup.sh` files). | 1 week |
| 3 | Implement `cross-operator` pack (3 examples). | 2 weeks |
| 4 | Implement `dependencies` pack. | 1 week |
| 5 | Implement `autoscale` pack. | 2 weeks |
| 6 | Update documentation (READMEs, website) to reference `make` commands. | 1 week |

## Risks & Mitigations

| Risk | Mitigation |
|------|-------------|
| Kind not installed | Makefile checks for `kind` and exits with install instructions. |
| Port conflicts for multi‑cluster tests | Use dedicated ports for each cluster (e.g., 6443 for cluster‑A, 6444 for cluster‑B). |

## Conclusion

A centralised Makefile and helper script will bring consistency to all Orkestra examples. More importantly, it will enable powerful new example packs that showcase Orkestra’s unique strengths: cross‑operator communication, declarative dependencies, and controller‑level autoscaling. These examples will turn abstract features into tangible, copy‑pastable experiences – making Orkestra’s value immediately clear. 🚀