Here's a comprehensive `DependencyKordination.md` that documents the design, the problem, the solution, and the behavior of the dependency-aware controller.

```markdown
# Dependency Kordinator

The `DependencyKordinator` is the core orchestration engine of Orkestra. It manages the lifecycle of Custom Resource Definitions (CRDs) in a dependency‑aware manner, ensuring CRDs start in the correct topological order, shut down in reverse order, and self‑heal when CRDs are added or removed from the cluster at runtime.

## Overview

Orkestra operators are defined via a `Katalog` that specifies:

- Which CRDs to reconcile
- Dependencies between CRDs (`dependsOn` with `condition: started` or `condition: healthy`)
- Worker pool sizes, resync intervals, and other runtime parameters

The `DependencyKordinator`:

- Computes a deterministic startup order using Kahn's algorithm
- Starts CRDs only after their dependencies are satisfied
- Shuts down CRDs in reverse order to allow graceful draining
- Monitors the cluster continuously and reactivates CRDs that appear after startup
- Deactivates CRDs that are deleted from the cluster, stopping workers without closing dependency channels

## Problem: Blocking on `healthy` Dependencies

### Original (Flawed) Behavior

In earlier versions, the main startup loop would **block** when a CRD required a dependency to be `healthy`:

```go
case string(types.DependencyConditionHealthy):
    select {
    case <-k.healthyCh[depGVK]:
    case <-ctx.Done():
        return
    }
```

Because the main goroutine processed CRDs sequentially, a CRD waiting for a dependency to become healthy would **stall the entire startup sequence**. Any CRD appearing later in the topological order would never be reached until that dependency became healthy—which could take minutes or hours.

### Symptom

The symptom was that a CRD like `service-sentinel` would remain `pending` forever, even though its dependencies (`started`) were already satisfied. The problem appeared only when a CRD with a `healthy` dependency appeared **before** it in alphabetical order within the same topological tier. Renaming the CRD to appear earlier in the sort order "fixed" the issue—but only by chance.

### Root Cause

The topological sort grouped CRDs by dependency depth, but within the same depth, alphabetical order determined the processing sequence. A CRD that blocked on `healthy` would starve all alphabetically later CRDs.

## Solution: Non‑Blocking Startup + Deferred Activation

### Core Principle

**The main startup loop must never block on a condition that may take an arbitrary amount of time.**

Instead, the loop checks each dependency **non‑blocking** using a `select` with a `default` case:

```go
func (k *DependencyKordinator) dependenciesReady(crd orktypes.CRDEntry, nameToGVK map[string]string) bool {
    for depName, depCond := range crd.DependsOn {
        depGVK := nameToGVK[depName]
        switch strings.ToLower(depCond.Condition) {
        case string(orktypes.DependencyConditionHealthy):
            select {
            case <-k.healthyCh[depGVK]:
                // ready
            default:
                return false
            }
        default: // started
            select {
            case <-k.startedCh[depGVK]:
                // ready
            default:
                return false
            }
        }
    }
    return true
}
```

If any dependency is not yet satisfied, the CRD is **skipped** during the initial startup pass.

### Deferred Activation via Retry Loop

A background goroutine (`retryMissingCRDs`) runs forever on a ticker. In addition to its original responsibilities (handling missing CRDs and detecting runtime deletions), it now includes **Phase 3**:

```go
// Phase 3: Activate deferred CRDs (skipped at startup because dependencies weren't ready)
for _, name := range k.depGraph.StartupOrder() {
    gvk := nameToGVK[name]
    if k.started[gvk] || k.informerFactory.IsMissing(gvk) {
        continue
    }
    crd := k.depGraph.GetNode(name).CRD
    if k.dependenciesReady(crd, nameToGVK) {
        entry := k.informerFactory.Registered()[gvk]
        k.activateCRD(ctx, entry)
    }
}
```

This ensures that once a dependency becomes `healthy` (its `healthyCh` is closed by the health checker), any waiting CRDs are activated on the next tick.

## Key Components

| Component       | Purpose                                                                                                 |
|-----------------|---------------------------------------------------------------------------------------------------------|
| `startedCh`     | Closed when a CRD's workers have started. Signals dependents that require `condition: started`.          |
| `healthyCh`     | Closed when a CRD has successfully processed its first reconciliation. Signals `condition: healthy`.     |
| `retryMissingCRDs` | Background loop that: <br>1. Activates CRDs that appear after startup <br>2. Deactivates CRDs that are deleted <br>3. Activates deferred CRDs when dependencies become ready |
| `dependenciesReady()` | Non‑blocking check of all dependency channels.                                                           |
| `activateCRD()` | Centralised routine to start informer, launch workers, close `startedCh`, and update health.             |
| `deactivateCRD()` | Stops workers and marks CRD as degraded **without** closing `startedCh`.                                 |

## Startup and Runtime Scenarios

### 1. All CRDs Present, All Dependencies `started`

- Main loop processes CRDs in topological order.
- Each CRD's dependencies are immediately ready.
- Workers start, `startedCh` closed, loop proceeds without delay.

### 2. CRD Missing at Startup

- Main loop skips CRD (`IsMissing == true`).
- Retry loop Phase 1 periodically checks for CRD appearance.
- When CRD is applied, `activateCRD` starts workers and closes `startedCh`.
- Dependents unblock and start normally.

### 3. CRD Deleted After Startup

- Retry loop Phase 2 detects CRD is gone.
- `deactivateCRD` stops workers, marks CRD as degraded.
- `startedCh` **is not closed**; dependents continue running (degraded).
- When CRD is recreated, `activateCRD` restores full operation.

### 4. Dependency Chain with `healthy` Condition (Fixed)

Given: `A` (no deps), `B` depends on `A:healthy`, `C` depends on `A:started`.

- Main loop: `A` starts, closes `startedCh`.
- `B` → `dependenciesReady()` returns false (healthyCh not closed) → **skipped**.
- `C` → dependencies ready → starts immediately.
- Retry loop Phase 3 periodically checks `B`.
- When `A` becomes healthy, health checker closes `healthyCh`.
- Next tick, Phase 3 activates `B`.

**Result:** `C` is not blocked by `B` waiting for `A` to become healthy.

### 5. Multiple Dependents with Mixed Conditions

Given: `A`, `B` depends on `A:started`, `C` depends on `A:healthy`, `D` depends on `A:started`.

Topological order (alphabetical tie‑breaking) may be `B`, `C`, `D`.

- Main loop processes `B` → starts.
- `C` → `dependenciesReady()` false → skipped.
- `D` → dependencies ready → starts.
- Retry loop later activates `C`.

All CRDs except `C` start immediately; `C` starts when healthy. No starvation.

## Design Decisions

| Decision                                                                 | Rationale                                                                                                   |
|--------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------|
| `startedCh` is **never closed during deactivation**.                      | Dependents should continue running (degraded) rather than block.                                              |
| `retryMissingCRDs` runs **forever**, not just at startup.                 | CRDs can be added/deleted at any time; continuous monitoring is required.                                     |
| `activateCRD` uses `select/default` when closing channels.                | Channels may already be closed from a previous activation; closing again would panic.                         |
| Main loop is **non‑blocking** for all dependency conditions.              | Prevents a single `healthy` dependency from stalling the entire startup sequence.                             |
| Deferred activation is handled by the existing retry loop (Phase 3).      | Reuses the proven self‑healing infrastructure without introducing new goroutines or timers.                   |
| Dependency graph uses **alphabetical tie‑breaking** for determinism.      | Order within the same depth is predictable, though the non‑blocking fix makes the order irrelevant.           |

## Summary

The `DependencyKordinator` now correctly handles:

- Topological startup and shutdown
- Self‑healing of missing or deleted CRDs
- Non‑blocking activation of CRDs waiting for `healthy` dependencies

The fix eliminates the alphabetical tie‑breaking bug and ensures that the startup sequence proceeds as quickly as possible, deferring only those CRDs whose dependencies are genuinely not ready. The system remains fully self‑healing and resilient to runtime changes in the cluster's CRD inventory.
```

This document can be placed in the project's `docs/` directory or alongside the code. It explains the design, the problem that was fixed, and how the new behavior works, making it a valuable reference for future contributors.