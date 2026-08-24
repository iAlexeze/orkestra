# Cross‑Operator Autoscaling Scenarios  
*Real‑world patterns for scaling based on other operators.*

Cross‑Operator Autoscaling enables an operatorBox: to scale using **runtime metrics from another operator**.  
This unlocks pipeline‑wide coordination, upstream/downstream flow control, and distributed load‑aware behavior.

Below are real‑life scenarios showing how teams use cross‑operator metrics to stabilize pipelines, prevent overload, and optimize throughput.

---

# 1. Upstream Slows Down When Downstream Is Overloaded  
*Protect downstream systems from overload.*

```yaml
autoscale:
  interval: 15s
  cooldown: 1m

  conditions:
    when:
      - field: cross.db.metrics.queueDepth
        greaterThan: "800"
      - field: cross.db.metrics.workersBusyPercent
        greaterThan: "85"

  do:
    workers: 2
    queueDepth: 50
```

### Behavior  
- If the **database operator** is overwhelmed, the upstream operator slows down  
- Prevents cascading failures  
- Reduces retries, backpressure, and queue explosions  

### Real‑life use case  
API ingestion → DB writer.  
If the DB writer is drowning, ingestion must throttle.

---

# 2. Upstream Speeds Up When Downstream Has Capacity  
*Exploit idle downstream capacity.*

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: cross.storage.metrics.workersIdlePercent
        greaterThan: "40"

  do:
    workers: 12
    queueDepth: 1500
```

### Behavior  
- If storage is idle, upstream accelerates  
- Maximizes throughput  
- Reduces end‑to‑end latency  

### Real‑life use case  
Transform → Storage.  
If storage is free, transform can push harder.

---

# 3. Multi‑Stage Pipeline Coordination  
*Scale only when upstream is busy AND downstream is ready.*

```yaml
autoscale:
  interval: 30s
  cooldown: 2m

  conditions:
    when:
      - field: cross.ingest.metrics.queueDepth
        greaterThan: "1000"
      - field: cross.indexer.metrics.workersIdlePercent
        greaterThan: "30"

  do:
    workers: 20
```

### Behavior  
- Ingest must be overloaded  
- Indexer must have capacity  
- Only then does the middle operator scale  

### Real‑life use case  
Ingest → Transform → Index.  
Transform should scale only when both sides justify it.

---

# 4. Cross‑Operator Error‑Driven Scaling  
*Scale up when a downstream operator is failing.*

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: cross.db.metrics.errorRatePercent
        greaterThan: "5"

  do:
    workers: 10
    resync: 15s
```

### Behavior  
- If the DB operator is failing too often, upstream increases concurrency  
- Helps retry storms clear faster  
- Reduces backlog buildup  

### Real‑life use case  
Payment processor → Ledger writer.  
If ledger is failing, processor must help clear the backlog.

---

# 5. Coordinated Batch Windows  
*Scale multiple operators together during a shared batch window.*

```yaml
autoscale:
  interval: 30s
  cooldown: 5m

  conditions:
    or:
      - cron: "0 23 * * *"
        duration: 3h

    when:
      - field: cross.etl.metrics.queueDepth
        greaterThan: "2000"

  do:
    workers: 18
    queueDepth: 4000
```

### Behavior  
- Window opens at 23:00  
- Operator scales only if ETL is also under load  
- Prevents unnecessary scaling during quiet nights  

### Real‑life use case  
ETL → Aggregator → Archiver.  
Archiver scales only when ETL is active and heavy.

---

# 6. Fan‑Out Coordination  
*Scale based on the combined pressure of multiple downstream operators.*

```yaml
autoscale:
  interval: 20s
  cooldown: 2m

  conditions:
    when:
      - field: cross.a.metrics.queueDepth
        greaterThan: "300"
      - field: cross.b.metrics.queueDepth
        greaterThan: "300"

  do:
    workers: 16
```

### Behavior  
- Both downstream operators must be under load  
- Prevents one slow consumer from dominating decisions  

### Real‑life use case  
Event router → multiple consumers.  
Router scales only when both consumers are busy.

---

# 7. Cross‑Cluster Coordination  
*Scale based on a remote operator in another cluster.*

```yaml
autoscale:
  interval: 30s
  cooldown: 2m

  conditions:
    when:
      - field: cross.remoteIndexer.metrics.queueDepth
        greaterThan: "1000"

  do:
    workers: 14
```

### Behavior  
- Uses HTTP fallback path  
- Works across clusters, regions, or binaries  
- Enables global autoscaling strategies  

### Real‑life use case  
Multi‑region ingestion → global indexer.

---

# 8. Cross‑Operator + Local Hybrid  
*Scale only when both local and downstream signals agree.*

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    or:
      - field: cross.db.metrics.queueDepth
        greaterThan: "500"

    when:
      - field: metrics.queueDepth
        greaterThan: "300"

  do:
    workers: 12
```

### Behavior  
- Either DB is overloaded OR local queue is high  
- But local queue must always exceed 300  
- Prevents false positives  

### Real‑life use case  
API → DB writer.  
API scales only when both sides justify it.

---

# 9. Cross‑Operator Safety Valve  
*Scale down when downstream is unhealthy.*

```yaml
autoscale:
  interval: 10s
  cooldown: 30s

  conditions:
    when:
      - field: cross.db.metrics.errorRatePercent
        greaterThan: "20"

  do:
    workers: 1
    queueDepth: 20
```

### Behavior  
- If DB is failing catastrophically, upstream throttles hard  
- Prevents meltdown  
- Protects data integrity  

### Real‑life use case  
High‑risk pipelines (payments, compliance, ledgering).

---

# 10. Full Pipeline Self‑Optimization  
*Every operator scales based on its neighbors.*

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: cross.prev.metrics.queueDepth
        greaterThan: "500"
      - field: cross.next.metrics.workersIdlePercent
        greaterThan: "20"

  do:
    workers: 10
```

### Behavior  
- Scale up only when upstream is busy AND downstream is ready  
- Creates a self‑balancing pipeline  
- No central controller required  

### Real‑life use case  
Any multi‑stage workflow:  
ingest → transform → validate → store → index → notify.