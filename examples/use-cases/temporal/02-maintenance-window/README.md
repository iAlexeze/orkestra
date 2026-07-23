# Temporal 02 — Maintenance Window

One CR. Three resources that always exist, one that changes shape on a schedule. The primary StatefulSet runs 24/7. The replicas StatefulSet scales to zero every Sunday 02:00–04:00 UTC and restores automatically when the window closes. A ConfigMap tells downstream load balancers what's happening. No human involvement.

**What you learn:** user-defined notes composing built-ins into domain vocabulary (`inMaintenance`, `nextMaintenance`, `activeReplicas`); using one note result in multiple places — replica counts, status fields, and ConfigMap data.

---

## What gets created

At all times:

```
Database/my-db               Ready    readReplicas=3   nextMaintenance=2026-07-19T02:00:00Z
StatefulSet/my-db-primary    1/1 Running
StatefulSet/my-db-replicas   3/3 Running
ConfigMap/my-db-routing      maintenance=false  readReplicas=3
```

During the maintenance window (Sun 02:00–04:00 UTC):

```
Database/my-db               Maintenance   readReplicas=0   nextMaintenance=2026-07-26T02:00:00Z
StatefulSet/my-db-primary    1/1 Running
StatefulSet/my-db-replicas   0/0
ConfigMap/my-db-routing      maintenance=true   readReplicas=0
```

The StatefulSet and ConfigMap are patched automatically. No deletion, no recreation — the reconciler converges the existing resources to the correct state.

---

## How it works

Three user-defined notes drive the behavior:

```yaml
notes:
  functions:
    - name: inMaintenance
      expression: '{{ and weekend (timeInWindow "02:00" "04:00") }}'

    - name: nextMaintenance
      expression: '{{ nextCron "0 2 * * 0" }}'

    - name: activeReplicas
      expression: "{{ if inMaintenance }}0{{ else }}{{ .spec.readReplicas | default 3 }}{{ end }}"
```

`activeReplicas` flows into the StatefulSet replica count and the ConfigMap in one place each — no duplication:

```yaml
statefulSets:
  - name: "{{ .metadata.name }}-replicas"
    replicas: "{{ activeReplicas }}"
    reconcile: true

configMaps:
  - name: "{{ .metadata.name }}-routing"
    data:
      maintenance: "{{ inMaintenance }}"
      readReplicas: "{{ activeReplicas }}"
    reconcile: true
```

---

## Step 1 — Validate and inspect notes

```bash
ork validate --notes
```

## Step 2 — Simulate

```bash
ork simulate
```

## Step 3 — Start the runtime

```bash
ork run
```

## Step 4 — Observe

> **Tip:** to see replicas drop to zero without waiting for Sunday, edit the `timeInWindow` expression in `inMaintenance` to bracket your current local time, save, and watch the StatefulSet scale down on the next resync.

In a separate terminal, watch the StatefulSet replica count and ConfigMap together:

```bash
kubectl get statefulset,configmap
```

Outside the window: `my-db-replicas` has 3 replicas, ConfigMap shows `maintenance=false`.
During the window: `my-db-replicas` drops to 0, ConfigMap shows `maintenance=true`.

---

## Step 5 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the StatefulSet replica count and the ConfigMap — both driven by the `activeReplicas` note, updating on every resync.

---

## E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[03 — Regional Peak](../03-regional-peak/README.md) — three deployments scaled independently by timezone, each with its own peak-hour window and per-note replica count.
