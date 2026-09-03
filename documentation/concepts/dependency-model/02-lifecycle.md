# Lifecycle

## How Orkestra builds the dependency graph

When Orkestra starts, it reads all CRD entries and their `dependsOn` declarations and builds a directed acyclic graph (DAG) — each node is a CRD, each edge points from a dependency to its dependent.

```text
database ──┐
           │
           ▼
     application
```

From the graph, Orkestra computes a topological sort — the startup order:

1. `database` starts first (no dependencies)
2. `application` starts second (waits for `database`)

For each CRD, Orkestra creates a ready channel — a signal that fires when the CRD meets its declared condition. Dependents block on their dependencies' channels before starting workers.

---

## Missing CRDs — wait, retry, self-heal

If a CRD declared in `dependsOn` is not installed in the cluster when Orkestra starts, Orkestra does not fail. Instead:

1. Orkestra starts a background retry loop for the missing CRD
2. All CRDs that depend on the missing one wait, blocking their ready channels
3. Healthy CRDs (those with no missing dependencies) start normally
4. When the missing CRD is eventually installed, Orkestra detects it, starts its informer and workers, and opens the ready channel

The dependents then start automatically. No restart needed.

**Example — database CRD missing at startup:**

```text
1. Orkestra sees database CRD is missing
2. application depends on database → application waits
3. Orkestra starts its background retry loop for database
```

You install the database CRD:

```bash
kubectl apply -f database-crd.yaml
```

```text
4. Orkestra detects the database CRD appears
5. Orkestra starts database workers and informer
6. database ready channel fires (condition: started)
7. application unblocks and starts its workers
8. application reconciles any existing Application CRs
```

The system self-heals without a restart.

---

## Circular dependency detection

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

```text
dependency cycle detected involving a → b → a
```

Use `ork validate --file katalog.yaml` to catch cycles before applying to a cluster.

---

## Shutdown order

When Orkestra shuts down, it stops CRDs in the reverse of startup order:

```text
Startup:   database → application
Shutdown:  application → database
```

Dependents stop before their dependencies. This ensures no running reconciler holds a reference to a dependency that has already been torn down.

---

## CLI visualization

Preview the dependency graph before running:

```bash
ork template --file my-katalog.yaml --graph
```

```text
Dependency Graph:
database
application
  └─ database
```

For a three-tier system:

```bash
ork template --file platform.yaml --graph
```

```text
Dependency Graph:
project
managednamespace
  └─ project
application
  └─ project
  └─ managednamespace
```
