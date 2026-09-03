# 02 — Dependency Graph

The dependency graph is an immutable DAG built at boot from all `dependsOn` declarations in the Katalog.

## Structure

```
Node  — one CRD
Edge  — dep → dependent   ("A must start before B" means edge A → B)
```

Nodes track `InDegree` (how many CRDs this CRD depends on) and `OutDegree` (how many CRDs depend on this CRD). Leaf nodes (no dependencies) have `InDegree = 0` and start first.

## Construction

```go
g := NewDependencyGraph(katalog)
```

`NewDependencyGraph` exits the process if a declared dependency does not exist in the enabled CRD set. Missing dependencies are caught earlier in `validateDependsOn`, so this is a safety net.

## Startup order

Computed once, then cached. Uses Kahn's algorithm with sorted queues for determinism — the order is the same across restarts.

```go
g.StartupOrder()   // CRDs with no deps first, then their dependents
g.ShutdownOrder()  // reverse of startup — drain dependents before deps
```

The runtime uses `ShutdownOrder()` to stop CRDs in reverse-dependency order, ensuring no CRD is stopped while something still depends on it.

## Validation

Two passes before the graph is used:

1. **Existence** — every name in `dependsOn` must be an enabled CRD key.
2. **Cycle detection** — DFS over the graph. A back-edge (node already on the current stack) means a cycle. The offending name is included in the error.

Both passes run inside `validateDependsOn()` before `NewDependencyGraph` is ever called.

## Example

```yaml
spec:
  crds:
    queue:
      ...
    worker:
      dependsOn:
        queue:
          condition: healthy
    processor:
      dependsOn:
        worker:
          condition: started
```

Graph edges: `queue → worker → processor`

```
StartupOrder():  [queue, worker, processor]
ShutdownOrder(): [processor, worker, queue]
```

## Inspecting at runtime

```go
node := g.GetNode("worker")
node.InDegree   // 1 (depends on queue)
node.OutDegree  // 1 (processor depends on it)

g.GetEdges()    // map: dep → []dependent
```

→ Next: [03-deletion-protection.md](03-deletion-protection.md)
