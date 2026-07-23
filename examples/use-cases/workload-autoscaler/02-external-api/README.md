# 02 — Workload Autoscaler: External API

Scale a Deployment based on live metrics from an external HTTP endpoint. When the queue is deep, the pool scales up. When it drains, it scales down.

---

## What You Learn

- How to use `external:` to fetch live metrics before autoscale evaluation
- Step scaling with `increment:` and `decrement:` for metric-driven workloads
- How `continueOnError: true` keeps the operator healthy when the metrics endpoint is temporarily unavailable
- How `external.*` data flows into `autoscale:` conditions via the same resolver that powers `when:` blocks everywhere

---

## Run the Example

### 1. Validate

```bash
ork validate
```

### 2. Simulate

```bash
ork simulate
```

### 3. Start Orkestra with the dev server

```bash
ork run --dev-server
```

The dev server starts on port `9999` and serves worker pool metrics at `/workload-metrics`.

### 4. Start the Control Center

```bash
ork control
# username:password → orkestra
# → http://localhost:8081
```

### 5. Check current metrics

```bash
curl -s http://localhost:9999/workload-metrics | jq .
```

```json
{
  "queue": {
    "pendingJobs": 8,
    "processingRate": 120,
    "errorRate": 0,
    "workerUtilPct": 22
  }
}
```

Queue is light — 8 pending jobs. Pool stays at 2 replicas (baseline).

### 6. Flip to high load

```bash
curl -X POST http://localhost:9999/workload-metrics/flip
```

```
high-load
```

`/workload-metrics` now returns 152 pending jobs — above the `greaterThan: "100"` threshold. On the next resync, the autoscaler adds 2 replicas. Each subsequent resync adds 2 more until `max: 10` is reached or the queue drains.

Watch the Deployment scale in Control Center or:

```bash
kubectl get deployment my-pool -w
```

### 7. Flip back

```bash
curl -X POST http://localhost:9999/workload-metrics/flip
```

```
baseline
```

`/workload-metrics` now returns 8 pending jobs — below `lessThan: "20"`. After the **3-minute cooldown**, the pool scales down 1 replica per resync until it reaches `min: 2`.

---

## How It Works

```yaml
onReconcile:
  external:
    - name: queue
      url: "{{ .spec.metricsEndpoint }}"
      method: GET
      timeout: 5s
      continueOnError: true

  deployments:
    - name: "{{ .metadata.name }}"
      replicas: 2
      autoscale:
        min: 2
        max: 10
        cooldown: 3m
        scaleUp:
          conditions:
            when:
              - field: external.queue.queue.pendingJobs
                greaterThan: "100"
          increment: 2
        scaleDown:
          conditions:
            when:
              - field: external.queue.queue.pendingJobs
                lessThan: "20"
          decrement: 1
```

The external call resolves before deployment evaluation. `external.queue.queue.pendingJobs` reads directly from the response using the same field path syntax as `when:` conditions elsewhere in the Katalog.

`continueOnError: true` means a flaky metrics endpoint does not block the reconcile or break the operator. When the endpoint is unreachable, `external.queue.queue.pendingJobs` is empty, the threshold conditions are not met, and the Deployment holds its current replica count until the endpoint recovers.

---

## E2E

```bash
ork e2e --dev-server
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
