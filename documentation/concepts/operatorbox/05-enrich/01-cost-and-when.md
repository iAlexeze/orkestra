# Cost and When to Use

## The API call budget

Every enrichment target issues at least one API call per reconcile cycle, per CR. This is not a framework abstraction — it is a real `list` or `get` request to the Kubernetes API server. The API server rate-limits, throttles, and queues these like any other request.

**Concrete example:** a `Microservice` CRD with `enrich: [pods, events]`, 30 CRs, resync every 15s.

| Target | Calls per reconcile | Calls per minute |
|---|---|---|
| `pods` | 1 | 2 per CR × 30 CRs = **60/min** |
| `events` | 1 | 2 per CR × 30 CRs = **60/min** |
| **Total** | 2 | **120/min** from enrichment alone |

Add `replicasets` and you are at 180/min. Scale to 200 CRs and it becomes 1,200 calls/minute — a meaningful load on a shared API server.

The right question before adding an enrichment target: **how often do I actually need this data?**

---

## When to use always-on enrichment

Use always-on enrichment only for data you surface on every reconcile:

```yaml
enrich:
  - pods   # always: pod count and readiness in status on every cycle
```

Good candidates:
- `pods` — pod count, ready count, and crash detection are useful in steady state
- `endpoints` — service reachability is useful continuously
- `pvc`, `storageclass` — cheap gets, useful when bound state matters

Bad candidates for always-on:
- `events` — warning events are noise in steady state, expensive at scale
- `replicasets` — only meaningful during a rollout
- `owner` — only useful when navigating ReplicaSet ownership

---

## The gating pattern

Gate expensive targets on the state that needs them. In steady state the condition fails, the API call is skipped, and the enriched key is absent from the child map. The note returns zero or empty — which is fine, because you gate the status field on the same condition.

```yaml
enrich:
  - pods                     # always — cheap, always useful
  - events:                  # only when something is wrong
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
  - replicasets:             # only during rollouts or debug
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"

status:
  fields:
    # Always written
    - path: podCount
      value: "{{ podCount .children.deployment }}"

    # Only written when degraded — matches the enrichment gate
    - path: firstWarning
      value: "{{ firstWarning .children.deployment }}"
      when:
        - field: "{{ hasWarnings .children.deployment }}"
          equals: "true"
```

**In steady state:** zero event-list calls. Zero replicaset-list calls. Just pod-list.
**When degraded:** events and replicasets are fetched. Warning message appears in status.

**Try it:**
```bash
ork init --pack use-cases
cd enrich/02-warning-events
ork run
kubectl apply -f cr-broken.yaml   # bad image → degraded → events fetched
kubectl apply -f cr-healthy.yaml  # good image → ready → events skipped
```

---

## `enrichAll` — development only

```yaml
enrichAll: true
```

`enrichAll` enables every target for the CRD. It is useful for exploring what enrichment data is available without having to declare targets one by one.

**Do not use `enrichAll` in production.** It issues the maximum possible number of API calls per reconcile, regardless of whether the data is needed. For a CRD with many replicas, `enrichAll` at a short resync interval can become the dominant source of API server load.

Use `enrichAll` to discover what's available, then declare explicit `enrich:` targets with `when:` gates for production.

---

**Next →** [Targets](02-targets.md)
