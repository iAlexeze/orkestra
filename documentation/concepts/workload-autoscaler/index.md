# Workload Autoscaler

The Workload Autoscaler scales the Deployments your operator manages based on time, external queue depth, sibling operator metrics, or any combination. It is declared directly on the Deployment:



```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      replicas: "{{ .spec.replicas }}"
      autoscale:
        min: 2
        max: 20
        cooldown: 3m
        scaleUp:
          conditions:
            when:
              - field: external.queue.pendingJobs
                greaterThan: "100"
          increment: 2
        scaleDown:
          conditions:
            when:
              - field: external.queue.pendingJobs
                lessThan: "10"
          decrement: 1
```

---

## What the workload autoscaler does

- Evaluates `scaleUp` and `scaleDown` conditions on every reconcile cycle
- Patches `spec.replicas` on the Deployment when conditions are met
- Enforces `min` and `max` boundaries
- Respects a `cooldown` period between scale events to prevent oscillation
- Works with any source the Resolver reads — `external:`, `cross:`, time notes, user-defined notes

---

## Relationship to the Operator Autoscaler

The **Operator Autoscaler** (`operatorBox.autoscale`) scales the operator's own machinery: workers, queue depth limit, resync period. It answers: *how fast should the operator reconcile?*

The **Workload Autoscaler** (`deployments[].autoscale`) scales the resources the operator manages: Deployment replicas. It answers: *how many instances of the managed workload should run?*

They share the same condition engine and run in the same reconcile loop. They are independent — you can use one, both, or neither.

---

## Documentation structure

### 1. [Scaling Signals](01-scaling-signals.md)
Time conditions, external HTTP endpoints, cross-operator metrics, notes, and composite conditions — every source the Resolver reads is a scaling signal.

### 2. [Schema Reference](02-schema.md)
Scaling modes (`target` vs `increment`/`decrement`), the full field reference, and validation rules.

---

## Try it

```bash
ork init --pack use-cases/workload-autoscaler
# Follow the steps in the README
```
