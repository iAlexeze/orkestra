# Startup Sequencing

## Dependency order

OperatorBoxes start in the topological order defined by their `dependsOn` declarations. The dependency graph is validated at Katalog load time — cycles are fatal.

```yaml
database-backed-app:
  dependsOn:
    managed-database: healthy   # wait until managed-database has reconciled successfully
```

Supported conditions:

| Condition | Meaning |
|---|---|
| `started` | Workers are running and the queue is accepting events |
| `healthy` | Running with zero consecutive reconcile failures |

The `DependencyKordinator` resolves the graph and starts each operatorBox when its declared condition is met.

---

## Relationship to the Katalog

A Katalog is a collection of operatorBox declarations. The Katalog is the schema; the operatorBox is the runtime instance of that schema.

One Katalog can declare many operatorBoxes. Multiple Katalogs can run in one Orkestra instance. Each operatorBox is isolated regardless of which Katalog declared it.

The Control Center shows one panel per operatorBox: health state, reconcile rate, queue depth, worker utilization, and the last error.
