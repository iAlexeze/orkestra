# Temporal 03 — Regional Peak

One CR. Three Deployments — one per region — all always running. Their replica count changes based on whether that region's business hours are currently active. A routing ConfigMap reflects the live state of all three. No CronJobs. No per-region operators.

**What you learn:** composing `timeInWindow` into per-region peak notes; driving replica counts from notes; how `reconcile: true` makes the operator patch resources continuously, not just on create.

---

## What gets created

At any point in time, all three Deployments and the ConfigMap exist. Only the replica counts differ:

```
CDNEdge/my-cdn-edge   Active   US-EAST=false   EU-CENTRAL=true   APAC=false

Deployment/my-cdn-edge-us-east      2/2 Running   (off-peak)
Deployment/my-cdn-edge-eu-central   8/8 Running   (peak)
Deployment/my-cdn-edge-apac         2/2 Running   (off-peak)

ConfigMap/my-cdn-edge-routing
  usEastPeak=false     usEastReplicas=2
  euCentralPeak=true   euCentralReplicas=8
  apacPeak=false       apacReplicas=2
```

When a region's window opens, its Deployment scales from 2 → 8. When it closes, 8 → 2. The ConfigMap updates in the same reconcile.

---

## UTC windows

| Region | Local window | UTC window |
|--------|-------------|------------|
| US-East | 09:00–17:00 ET (UTC-5) | 14:00–22:00 |
| EU-Central | 09:00–17:00 CET (UTC+1) | 08:00–16:00 |
| APAC | 09:00–17:00 JST (UTC+9) | 00:00–08:00 |

---

## How it works

Six user-defined notes — three peak flags, three replica counts that reference them:

```yaml
notes:
  functions:
    - name: usEastPeak
      expression: '{{ timeInWindow "14:00" "22:00" }}'
    - name: euCentralPeak
      expression: '{{ timeInWindow "08:00" "16:00" }}'
    - name: apacPeak
      expression: '{{ timeInWindow "00:00" "08:00" }}'

    - name: usEastReplicas
      expression: "{{ if usEastPeak }}{{ .spec.peakReplicas | default 8 }}{{ else }}{{ .spec.baseReplicas | default 2 }}{{ end }}"
    - name: euCentralReplicas
      expression: "{{ if euCentralPeak }}{{ .spec.peakReplicas | default 8 }}{{ else }}{{ .spec.baseReplicas | default 2 }}{{ end }}"
    - name: apacReplicas
      expression: "{{ if apacPeak }}{{ .spec.peakReplicas | default 8 }}{{ else }}{{ .spec.baseReplicas | default 2 }}{{ end }}"
```

Each Deployment uses its region's replica note. `reconcile: true` means the operator patches the replica count on every resync — not just on create:

```yaml
deployments:
  - name: "{{ .metadata.name }}-us-east"
    replicas: "{{ usEastReplicas }}"
    reconcile: true
  - name: "{{ .metadata.name }}-eu-central"
    replicas: "{{ euCentralReplicas }}"
    reconcile: true
  - name: "{{ .metadata.name }}-apac"
    replicas: "{{ apacReplicas }}"
    reconcile: true
```

---

## Step 1 — Validate and inspect notes

```bash
ork validate --notes
```

Six user-defined notes appear: three peak flags and three replica counts.

## Step 2 — Simulate

```bash
ork simulate
```

## Step 3 — Start the runtime

```bash
ork run
```

## Step 4 — Observe

> **Tip:** to watch a Deployment scale live, edit one of the `timeInWindow` expressions in [`katalog.yaml`](katalog.yaml) to bracket your current local time, save, and watch the replica count change on the next resync.

In a separate terminal, watch all three Deployments and the CDNEdge status together:

```bash
kubectl get cdnedge,deployment
```

The `US-EAST`, `EU-CENTRAL`, and `APAC` printer columns on the CDNEdge show the live peak flags. The Deployment `READY` column shows the current replica count for each region.

---

## Step 5 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the three Deployments and their current replica counts — each driven by its regional note, updating on every resync.

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
