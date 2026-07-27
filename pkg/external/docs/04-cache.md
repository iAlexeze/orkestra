# 04 — The Result Cache

`cacheFor:` decouples the reconcile rate from the query rate. A Deployment reconciling every 5 seconds can read from Prometheus every 30 seconds. The cache holds the last result and serves it for the TTL — no protocol call is made on a hit.

## Opt-in per call

The cache is per-call and opt-in. Calls without `cacheFor:` always hit the network:

```yaml
external:
  - name: queueDepth
    protocol: prometheus
    url: "http://prometheus.monitoring.svc:9090"
    query: "sum(rabbitmq_queue_messages)"
    cacheFor: 15s        # served from cache for 15s after the last fetch

  - name: featureFlag
    url: "http://flags.internal/api/flags"
    # no cacheFor: — hits the network on every reconcile
```

## Cache key

The key is derived from four components, null-byte separated:

```
<gvk> + "\x00" + <call-name> + "\x00" + <resolved-url> + "\x00" + <resolved-query>
```

`gvk` is the CRD's Group/Version/Kind string — the same CRD in different namespaces produces different keys only if `url:` or `query:` resolves differently per instance. If both a `website` CR named `foo` and one named `bar` resolve to the same URL and query, they share a cache entry.

This is intentional for cluster-wide signals (Prometheus metrics, global queue depth) where the value does not vary per instance. It is a consideration for per-instance signals — use `{{ .metadata.name }}` in `url:` or `query:` to scope the key to the instance.

## Eviction

The cache is an in-memory `sync.Map`. Entries are not pre-evicted on a background ticker — they are evicted lazily on the next `cacheGet` call when `time.Now().After(entry.expiresAt)`. A cache entry that is never accessed again after expiry stays in memory until the process restarts.

This is acceptable for the reconcile use case where every active CR re-accesses its cache entry on every reconcile cycle.

## No cache invalidation

There is no explicit invalidation API. If a dependency changes and you need fresh data immediately, set `cacheFor:` to a short duration or remove it. The cache is not aware of external state changes — it holds what was fetched at `cacheSet` time until the TTL expires.

## Relationship to KEDA

KEDA's `pollingInterval` is a global per-ScaledObject setting — it controls how often KEDA evaluates the scaler, and there is no per-metric caching inside that interval. Orkestra's `cacheFor:` is per-call and independent of the reconcile rate. A CRD reconciling every 5 seconds with `cacheFor: 30s` makes one Prometheus call every 30 seconds — 6x fewer than KEDA at equivalent responsiveness.

---

**← Back** [03 — Auth](03-auth.md) · **Next →** [05 — Adding a Protocol](05-adding-a-protocol.md)
