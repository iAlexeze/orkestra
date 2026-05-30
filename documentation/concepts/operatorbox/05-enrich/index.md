# Enrich

`enrich` tells Orkestra which secondary Kubernetes objects to fetch at reconcile time and embed into the child resource's template context. Without it, notes like `podCount`, `hasWarnings`, and `deploymentReplicaSetCount` return zero or empty — they have no data to work with.

**Why it exists.** Orkestra's informer watches your CR. When something changes — a pod crashes, a new ReplicaSet is created, an event is emitted — the CR is requeued. But the reconciler only reads the CR itself from the cache. The child objects (pods, events, ReplicaSets) exist in Kubernetes, not in the operator's in-memory cache. `enrich` is the explicit instruction to go fetch them and attach them to the child map that notes and status templates consume.

```yaml
enrich:
  - pods     # always fetch pod list for .children.deployment
  - events   # always fetch warning events
```

Or conditionally:

```yaml
enrich:
  - pods
  - events:
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

---

## Where enrichment sits in the pipeline

```text
informer cache → DeepCopy → normalize → mutation → validation
    → template reconcile
        → read children
        → enrich ← you are here
        → status fields (now have _pods, _warnings, etc.)
```

Enrichment runs after children are read and before status fields are written. The `_pods`, `_warnings`, `_replicaSets` keys are available in every status field template that reconcile cycle.

---

## The cost

Each enrichment target is one or more Kubernetes API calls per reconcile. This is the most important thing to understand about enrichment.

| Target | API calls | When it matters |
|---|---|---|
| `pods` | 1 list-pods per child | Every reconcile — keep this cheap |
| `events` | 1 list-events per child | Events are verbose — gate this |
| `replicasets` | 1 list-replicasets per child | Only useful during rollouts |
| `pvc`, `storageclass` | 1 get per object | Usually cheap |
| `endpoints` | 1 list-endpointslices per service | Scales with service count |

A Deployment with `enrich: [pods]` reconciling every 15 seconds = **4 pod-list calls per minute per CR**. With 50 CRs that is 200 calls/minute purely for pod enrichment. Gate what you can.

See [Cost and When to Use](01-cost-and-when.md) for the full cost model and gating patterns.

---

## Try it

```bash
ork init --pack use-cases
cd enrich/01-pod-health      # always-on pod enrichment — count, readiness, crash detection
cd enrich/02-warning-events  # conditional event enrichment — only fetched when degraded
cd enrich/03-rollout-observer # conditional replicaset enrichment — only during rollouts
ork run
```

---

## Where to go next

- [Cost and When to Use](01-cost-and-when.md) — API call budget, enrichAll warning, the gating pattern
- [Targets](02-targets.md) — all enrichment targets, what they embed, which notes they unlock
- [Conditional Enrichment](03-conditional.md) — gate patterns, anyOf, combining conditions
