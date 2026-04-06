# How Dependencies Work

This document explains how Orkestra manages dependencies between CRDs — automatically, without any code.

---

## The Problem

When you have multiple CRDs that depend on each other, you need them to start in the right order.

Imagine a web application that needs a database before it can run:

- The **Database CRD** must be ready before the **Application CRD** starts
- If the database CRD is missing, the application should wait
- When the database appears, the application should start automatically

Traditional operators do not handle this. You have to write code to check if dependencies exist, wait for them, and retry when they appear.

Orkestra does this for you.

---

## Declaring Dependencies

In your Katalog, you declare dependencies with `dependsOn`. Three formats are supported.

### Format 1 — Simple list (backwards-compatible)

The most common format. Each entry is a CRD name. The condition defaults to `started`.

```yaml
crds:
  database:
    apiTypes:
      group: database.myorg.io
      version: v1alpha1
      kind: Database
      plural: databases

  application:
    apiTypes:
      group: app.myorg.io
      version: v1alpha1
      kind: Application
      plural: applications
    dependsOn:
      - database      # Application depends on Database (condition: started)
```

### Format 2 — Key-value map (name → condition string)

When you need different readiness conditions per dependency:

```yaml
crds:
  application:
    apiTypes:
      group: app.myorg.io
      version: v1alpha1
      kind: Application
      plural: applications
    dependsOn:
      database: started     # wait until database informer is started
      cache: ready          # wait until cache is reporting ready
```

Supported condition values:

| Condition | Meaning |
|-----------|---------|
| `started` | The dependency's informer has synced and workers have launched. Default when using the simple list form. |
| `ready` | The dependency is started and its health endpoint reports `ready: true`. |
| `healthy` | The dependency is started, ready, and reports no degradation. |

### Format 3 — Full map (name → condition object)

The most explicit form. Useful when tooling or schema validation requires structured objects:

```yaml
crds:
  application:
    apiTypes:
      group: app.myorg.io
      version: v1alpha1
      kind: Application
      plural: applications
    dependsOn:
      database:
        condition: started
      cache:
        condition: ready
```

All three formats are interchangeable. Mix them freely across different CRD entries in the same Katalog.

---

## Condition Semantics

### `started`

The dependency CRD's informer has performed its initial list and the worker pool is running. The dependency is not necessarily processing CRs successfully — only that it has started. This is the minimum signal and the default when no condition is specified.

Use `started` when the downstream CRD only needs the upstream CRD to exist and be registered, not necessarily healthy.

### `ready`

The dependency has passed `started` and additionally reports `ready: true` on its health endpoint (`/katalog/{crd}/health`). This means at least one reconcile cycle has succeeded.

Use `ready` when the downstream CRD creates resources that depend on the upstream having successfully reconciled at least one CR.

### `healthy`

The dependency has passed `ready` and additionally has no active degradation — zero consecutive reconcile failures, queue depth below the degrade threshold, no error rate spike.

Use `healthy` for the strictest ordering guarantee, typically in production environments where partial readiness is unacceptable.

---

## How Orkestra Builds the Dependency Graph

When Orkestra starts, it reads all CRD entries and their `dependsOn` declarations. It builds a directed acyclic graph (DAG) in which each node is a CRD and each edge points from a dependency to its dependent.

```
database ──┐
           │
           ▼
     application
```

From the graph, Orkestra computes a topological sort — the startup order:

1. `database` starts first (no dependencies)
2. `application` starts second (waits for `database`)

For each CRD, Orkestra creates a ready channel — a signal that fires when the CRD meets its declared condition. Dependents block on the channels of all their declared dependencies before starting workers.

---

## Missing CRDs — Wait, Retry, Self-Heal

If a CRD declared in `dependsOn` is not installed in the cluster when Orkestra starts, Orkestra does not fail. Instead:

1. Orkestra starts a background retry loop for the missing CRD
2. All CRDs that depend on the missing one wait, blocking their ready channels
3. Healthy CRDs (those with no missing dependencies) start normally
4. When the missing CRD is eventually installed, Orkestra detects it, starts its informer and workers, and opens the ready channel

The dependents then start automatically. No restart needed.

**Example — database CRD missing at startup:**

```
1. Orkestra sees database CRD is missing
2. application depends on database → application waits
3. Orkestra starts its background retry loop for database
```

You install the database CRD:

```bash
kubectl apply -f database-crd.yaml
```

```
4. Orkestra detects the database CRD appears
5. Orkestra starts database workers and informer
6. database ready channel fires (condition: started)
7. application unblocks and starts its workers
8. application reconciles any existing Application CRs
```

The system self-heals without a restart.

---

## Circular Dependency Detection

Orkestra detects circular dependencies at startup and refuses to run:

```yaml
crds:
  a:
    dependsOn:
      - b
  b:
    dependsOn:
      - a
```

Error:

```
dependency cycle detected involving a → b → a
```

Fix your dependencies before running. Use `ork validate --katalog katalog.yaml` to catch cycles before applying to a cluster.

---

## Shutdown Order

When Orkestra shuts down, it stops CRDs in the **reverse** of startup order:

```
Startup:   database → application
Shutdown:  application → database
```

Dependents stop before their dependencies. This ensures no running reconciler holds a reference to a dependency that has already been torn down.

---

## CLI Visualization

Preview the dependency graph before running:

```bash
ork template --katalog my-katalog.yaml --graph
```

Output:

```
Dependency Graph:
database
application
  └─ database
```

For a three-tier system:

```bash
ork template --katalog platform.yaml --graph
```

```
Dependency Graph:
project
managednamespace
  └─ project
application
  └─ project
  └─ managednamespace
```

---

## Summary

| What | How Orkestra Handles It |
|------|------------------------|
| Declare dependencies | `dependsOn` in three formats: list, key-value map, or full map |
| Startup order | Automatic — dependencies first, computed via topological sort |
| Condition granularity | `started`, `ready`, or `healthy` per dependency |
| Missing CRDs | Wait, retry in background, activate when they appear |
| CRD deletion | Workers stop, informer recovers when recreated |
| Shutdown order | Reverse of startup |
| Circular dependencies | Detected at validate time and rejected at startup |

**You declare the dependencies. Orkestra handles the order.**
