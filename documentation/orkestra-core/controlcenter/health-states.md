# Health States

The Control Center tracks two independent health dimensions: the Orkestra runtime itself, and the CRDs it manages.

---

## Runtime Health

**Runtime Health** reflects whether the Orkestra process is reachable and operating normally.

| State | Meaning |
|-------|---------|
| Operational | Runtime is running and serving requests |
| Degraded | Runtime is unreachable or has internal errors |

---

## Katalog Health

**Katalog Health** reflects whether the CRDs within a Katalog are reconciling correctly.

| State | Meaning |
|-------|---------|
| Healthy | All CRDs are reconciling without errors |
| Degraded | One or more CRDs have persistent failures |

---

## CRD States

Each CRD card shows one of four states:

| State | Color | Meaning |
|-------|-------|---------|
| Healthy | Green | Fully operational, no errors |
| Started | Blue | Running but may have initial reconcile errors |
| Pending | Yellow | Waiting — dependencies not ready or CRD not yet applied |
| Degraded | Red | Persistent failures, needs attention |

---

## Why Runtime and Katalog Health Can Differ

Runtime Operational + Katalog Degraded is the normal failure case. The runtime is fine — the problem is in your CRD templates or the resources they create.

Runtime Degraded means the Orkestra process itself has a problem: check the deployment, network, or leader election.

---

## Queue Pressure

Queue depth is shown as a progress bar on each CRD card:

| Fill | Meaning |
|------|---------|
| < 50% | Normal |
| 50–80% | Moderate — watch it |
| > 80% | High — consider increasing `workers` for this CRD |
