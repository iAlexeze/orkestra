# Async Reconciliation

Orkestra's reconciler is synchronous — it receives a CR key, runs, and returns.
Real systems are not synchronous — a Deployment takes time to become ready, a
database takes time to accept connections. The `when:` gate on `onReconcile`
bridges this: if the condition is not met, the block is skipped and the CR is
requeued automatically. No sleeping. No polling. No custom state machine.

---

## The pattern

```yaml
operatorBox:
  default: true

  # Phase 1 — runs once on CR creation
  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true

  # Phase 2 — runs only when Phase 1 is complete
  onReconcile:
    services:
      - name: "{{ .metadata.name }}-svc"
        port: 80
        targetPort: 8080
        reconcile: true
        when:
          - field: children.deployment.status.readyReplicas
            equals: "{{ .spec.replicas }}"
```

What happens:

1. `onCreate` runs — Deployment is created
2. `onReconcile` evaluates `when:` — Deployment not ready yet — block skipped, CR requeued
3. Deployment pods become Ready — watch event fires — CR requeued
4. `onReconcile` evaluates `when:` — condition passes — Service is created
5. Every subsequent reconcile: Deployment still ready — Service is re-applied (drift correction)

---

## How requeue works without explicit requeue calls

**Watch events.** When a child resource's status changes — pods becoming Ready,
a Job completing, a LoadBalancer receiving an IP — the informer fires an event
that requeues the owning CR. This happens automatically because Orkestra sets
owner references and `orkestra-owner` labels on every child resource.

**Resync.** Each operatorbox has a `resync:` interval (default 30s–120s). At
each tick, all known CRs are re-enqueued regardless of changes. This is the
backstop — even if a watch event is delayed or missed, the CR reconciles within
one resync cycle.

You never need `ctrl.Result{RequeueAfter: N}`. The runtime handles it.

---

## The `.children.*` namespace

The `when:` gate references child resource status via the `.children.*` resolver
namespace. Orkestra populates this before every reconcile from the API server's
watch cache — no etcd round-trip, no added latency.

Common fields:

| Note expression | What it returns |
|---|---|
| `{{ readyReplicas .children.deployment }}` | Number of ready pods |
| `{{ desiredReplicas .children.deployment }}` | Total desired pods |
| `{{ jobSucceeded .children.job }}` | `true` when job completed |
| `{{ serviceLoadBalancerIP .children.service }}` | LB ingress IP |
| `{{ serviceLoadBalancerHost .children.service }}` | LB ingress hostname |
| `{{ get .children.cronjob "status" "lastScheduleTime" }}` | Last scheduled run |

When multiple children of the same kind exist, `.children.deployment` returns
the first. Access by name via `.children.deployment[0]` for arrays (when
multiple deployments are declared).

The same `.children.*` data is available in status field expressions, giving
the CR's status a live view of its children:

```yaml
status:
  fields:
    - path: readyReplicas
      value: "{{ get .children.deployment "status" "readyReplicas" }}"
    - path: endpoint
      value: "{{ get .children.service "status" "loadBalancer" "ingress" }}"
```

---

## Multiple ordered phases

Phases are ordered by structure: `onCreate` runs before `onReconcile`. Within
`onReconcile`, multiple `when:` groups execute in declaration order:

```yaml
onReconcile:
  # Phase A — always runs
  configMaps:
    - name: "{{ .metadata.name }}-config"
      reconcile: true

  # Phase B — runs only when Deployment is ready
  services:
    - name: "{{ .metadata.name }}-svc"
      reconcile: true
      when:
        - field: children.deployment.status.readyReplicas
          equals: "{{ .spec.replicas }}"

  # Phase C — runs only when Service has a LoadBalancer IP
  ingresses:
    - name: "{{ .metadata.name }}-ingress"
      host: "{{ .spec.host }}"
      reconcile: true
      when:
        - field: children.service.status.loadBalancer
          operator: exists
```

Each `when:` gate applies to the resource groups that follow it in the block,
up to the next `when:` gate. The first phase (ConfigMap) has no gate and runs
unconditionally. The second (Service) waits for the Deployment. The third
(Ingress) waits for the Service's LoadBalancer.

---

## Idempotency

Every resource group uses create-or-update semantics. A `when:` gate that
evaluates to true on ten consecutive reconciles executes the same block ten
times — each execution produces the same result. Idempotency is a property of
the resource application, not something the operator author needs to prove for
each phase.

---

## What this replaces

In a traditional Go operator, the equivalent of the three-phase example above
requires:

- A phase annotation on the CR status
- A switch statement routing to per-phase handlers
- Each handler returning `ctrl.Result{RequeueAfter: 5s}` while waiting
- Explicit existence checks before creating each resource
- Idempotency proofs for each creation call
- Tests for each phase transition

In Orkestra, it is two resource groups and one `when:` condition. The runtime
handles scheduling, requeue, and idempotency.