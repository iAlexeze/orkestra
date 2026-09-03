# Conditional Enrichment

Enrichment targets support an optional `when:` gate. When the condition is false, Orkestra skips the API call for that target — the enriched key is absent from the child map that reconcile cycle. This reduces API server load to near-zero in steady state.

---

## Basic gate

```yaml
enrich:
  - pods                    # always fetched
  - events:                 # only when deployment is not fully ready
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

When all replicas are ready, `events` is skipped entirely. `_warnings` is absent from `.children.deployment`. Notes like `hasWarnings` return their zero value — which is correct because there are no warnings to surface.

---

## Matching status field gates

Gate status fields on the same condition as the enrichment. If the enriched key is absent, the note returns zero — so the condition prevents writing a misleading value:

```yaml
enrich:
  - events:
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"

status:
  fields:
    # Only written when enrichment ran — gate matches
    - path: firstWarning
      value: "{{ firstWarning .children.deployment }}"
      when:
        - field: "{{ hasWarnings .children.deployment }}"
          equals: "true"
```

In steady state: no event fetch, no `firstWarning` field written. Under degradation: events fetched, warning message written. One condition, two effects.

---

## `or:` — OR logic

`or:` fetches the target when any one condition is true:

```yaml
enrich:
  - events:
      or:
        - field: "{{ hasCrashingPod .children.deployment }}"
          equals: "true"
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

Events are fetched when there is a crashing pod OR when replicas are not fully ready.

---

## Debug mode gate

Gate expensive diagnostic enrichment on a spec flag:

```yaml
enrich:
  - pods
  - replicasets:
      when:
        - field: spec.debug
          equals: "true"

status:
  fields:
    - path: replicaSetCount
      value: "{{ deploymentReplicaSetCount .children.deployment }}"
      when:
        - field: spec.debug
          equals: "true"
```

Set `spec.debug: "true"` on a CR to enable detailed rollout visibility for that instance only. Other CRs pay zero cost.

---

## Combining always-on and conditional

Always-on for cheap, always-useful targets. Conditional for expensive or situational ones:

```yaml
enrich:
  - pods                     # always — 1 API call, always useful
  - events:                  # conditional — skip in steady state
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
  - replicasets:             # conditional — only during rollouts
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

In steady state: 1 API call (pods). Under degradation: 3 API calls (pods + events + replicasets). The additional calls happen exactly when you need the data and stop as soon as the deployment recovers.

**Try it:**
```bash
ork init --pack use-cases
cd enrich/02-warning-events
ork run
# Apply a broken CR — watch events appear
kubectl apply -f cr-broken.yaml
kubectl get microservice broken-app -o yaml | grep -A5 "status:"

# Fix it — watch events disappear
kubectl patch microservice broken-app --type=merge -p '{"spec":{"image":"nginx:1.25"}}'
kubectl get microservice broken-app -o yaml | grep "firstWarning"
# <empty> — no events fetched, no field written
```

---

## Evaluation order

Enrichment conditions evaluate after children are read — so `.children.*` fields are available. This is what makes `{{ replicasReady .children.deployment }}` work as a gate: the Deployment object is already in context when the condition is evaluated.

**Back →** [Enrich](index.md)
