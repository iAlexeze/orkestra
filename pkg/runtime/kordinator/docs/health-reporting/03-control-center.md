# CRD Health Reporting — Control Center

The CC fetches health data from the runtime over HTTP. Two paths exist: the background fetch loop (periodic, cached) and the CRD detail page (on-demand, per request). Both need to handle multi-replica runtimes without showing stale follower data.

---

## The connection pooling problem

Go's `http.Client` reuses TCP connections by default (HTTP keep-alive). With a Kubernetes Service in front of a multi-replica runtime:

```
CC http.Client
  └── first request to orkestra-runtime.svc:8080
        → kube-proxy picks a backend pod (e.g. the follower)
        → TCP connection established
        → all subsequent requests reuse this connection
        → CC is pinned to the follower forever
```

The Service's load balancing only applies to new connections. With a pooled client, the CC never naturally migrates to the leader.

**Fix:** `DisableKeepAlives: true` forces a new TCP connection on every fetch:

```go
func NewClient(baseURL string, _ time.Duration, _ string) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                DisableKeepAlives: true,
            },
        },
    }
}
```

With a new connection per tick, kube-proxy distributes across pods, and every fetch has a fair chance of hitting the leader.

---

## Background fetch — `fetchAllKatalogs`

Runs every `RefreshInterval` seconds. Calls `GET /katalog` on each runtime URL and updates `inst.Katalog`.

### Cache-update rule

`inst.Katalog` is only replaced when `KatalogResponse.IsKonductor == true`:

```
Response from leader (isKonductor: true)
  → replace inst.Katalog with authoritative data
  → inst.Status = "online", inst.Healthy = true

Response from follower (isKonductor: false), inst.Katalog == nil
  → accept it — better than nothing; shows CRDs as pending
  → inst.Status = "starting", inst.Healthy = false

Response from follower (isKonductor: false), inst.Katalog already set
  → discard — keep last known-good snapshot
  → display does not change
```

**Why the nil case matters:** With multiple runtime Services, the first fetch for each one opens a new connection. If it lands on the follower, `inst.Katalog` would stay nil forever without the nil case — the katalog would never appear in the UI until a lucky tick hits the leader. Accepting the first response (even stale) makes every katalog visible immediately; the leader's data overwrites it on the next hit.

### Convergence

After startup, with 2 pods and true random load balancing per connection:

```
Tick 1:  hits follower → inst.Katalog = pending (nil case)
Tick 2:  hits leader   → inst.Katalog = healthy ✓
Tick 3:  hits follower → discard, keep healthy ✓
Tick 4:  hits leader   → update healthy ✓
```

All runtimes converge to showing healthy within 1–2 ticks (~10–20 seconds).

---

## CRD detail page — `FetchCRDDetail`

Called on-demand when a user opens a CRD page. Makes two live requests:
- `GET /katalog/{crd}/health` — state, errors, reconcile counters
- `GET /katalog/{crd}` — GVK, workers config, RBAC, providers

With `DisableKeepAlives`, each call opens a fresh connection and can land on any pod. The health fields flap if the health request hits the follower — configuration fields (GVK, workers) are static and don't flap.

### Retry until leader

`FetchCRDDetail` retries the health call up to 3 times until it gets `isKonductor: true`:

```go
var health *CRDHealth
for i := 0; i < 3; i++ {
    h, err := getJSON[CRDHealth](c, "/katalog/"+name+"/health")
    if err != nil {
        return nil, fmt.Errorf("fetching health: %w", err)
    }
    if h.IsKonductor {
        health = h
        break
    }
}
if health == nil {
    // All 3 attempts hit a follower — use last response rather than error
    health, _ = getJSON[CRDHealth](c, "/katalog/"+name+"/health")
}
```

The info call (`GET /katalog/{crd}`) is retried with the same logic. Worker fields (`workersActive`, `workersIdle`, `workerDetails`) are live runtime state — zero on follower pods. Showing zero workers alongside 101 reconciles would be misleading. Static fields (GVK, resync, namespace) are identical on both pods.

```go
var info *CRDInfo
for i := 0; i < 3; i++ {
    inf, err := getJSON[CRDInfo](c, "/katalog/"+name)
    if err != nil {
        return nil, fmt.Errorf("fetching info: %w", err)
    }
    if inf.IsKonductor {
        info = inf
        break
    }
}
if info == nil {
    info, _ = getJSON[CRDInfo](c, "/katalog/"+name)
}
```

With 2 pods and 50/50 routing, the probability of hitting the leader at least once in 3 tries is 87.5%. The fallback renders with stale data rather than an error.

---

→ Next: [04-diagnosis.md](04-diagnosis.md)  
→ Previous: [02-runtime.md](02-runtime.md)
