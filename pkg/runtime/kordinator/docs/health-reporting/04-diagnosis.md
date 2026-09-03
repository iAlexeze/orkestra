# CRD Health Reporting — Diagnosis

## Identifying the leader pod

Port-forward directly to each runtime pod and check `isKonductor` on both `/katalog` and `/katalog/{crd}/health`:

```bash
# Forward to each pod individually
kubectl port-forward -n orkestra-system-01 pod/<pod-a> 8010:8080 &
kubectl port-forward -n orkestra-system-01 pod/<pod-b> 8011:8080 &

# Check which pod is the leader
curl -sSL localhost:8010/katalog | jq .isKonductor   # true  = leader
curl -sSL localhost:8011/katalog | jq .isKonductor   # false = follower
```

The leader will have non-zero `totalReconciles` and a real `lastReconcile` timestamp:

```bash
curl -sSL localhost:8010/katalog/website/health | jq '{isKonductor,state,totalReconciles,lastReconcile}'
# Leader:
{
  "isKonductor": true,
  "state": "healthy",
  "totalReconciles": 101,
  "lastReconcile": "2026-05-29T15:05:14Z"
}

curl -sSL localhost:8011/katalog/website/health | jq '{isKonductor,state,totalReconciles,lastReconcile}'
# Follower:
{
  "isKonductor": false,
  "state": "pending",
  "totalReconciles": 0,
  "lastReconcile": "not started"
}
```

---

## Confirming the CC is getting authoritative data

Check the CC logs for each runtime fetch. Every tick logs the katalog name and CRD count:

```
INFO: fetched katalog "data-platform" from http://orkestra-runtime.orkestra-system-05.svc.cluster.local:8080 (10 CRDs, gateway="")
```

If you see all runtimes fetched consistently every tick but some still show pending in the UI, the CC is hitting the follower and the `isKonductor` guard is discarding the update. The state will resolve within 1–2 ticks once the guard accepts a leader response.

---

## Common failure patterns

### All runtimes show pending indefinitely

**Cause:** The runtime binary is old — it does not emit `isKonductor` in its response. The field defaults to `false` in JSON. The CC guard discards every update after the first follower response, freezing state as pending.

**Fix:** Rebuild and redeploy the runtime. Both runtime and CC must be on builds that include `isKonductor`.

**Check:**
```bash
curl -sSL <runtime-svc>/katalog | jq 'has("isKonductor")'
# true  → new build
# false → old build (isKonductor will be absent, Go unmarshals as false)
```

---

### Some runtimes healthy, others stuck on pending

**Cause (pre-fix):** Connection pooling in the CC HTTP client. The first TCP connection to each runtime Service lands on a random pod. If it lands on the follower, all subsequent requests reuse that connection and always return follower data.

**Fix:** `DisableKeepAlives: true` on the CC HTTP client (see [03-control-center.md](03-control-center.md)).

**Confirm:**
```bash
# From inside the CC pod, verify the runtime responds with isKonductor
kubectl exec -n orkestra-system-01 deploy/orkestra-cc -- \
  wget -qO- http://orkestra-runtime.orkestra-system-05.svc.cluster.local:8080/katalog \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('isKonductor:', d['isKonductor'])"
```

Run this a few times — with `DisableKeepAlives`, each run opens a new connection. You should see `isKonductor: true` approximately half the time (leader) and `false` the other half (follower).

---

### CRD detail page flaps but katalog index is stable

**Cause:** The background fetch (index) is cached and guarded. The CRD detail page makes live on-demand calls that can land on the follower.

**Fix:** Retry logic on `FetchCRDDetail` (see [03-control-center.md](03-control-center.md)).

**Confirm:** Open the CRD detail page and watch the `state` badge — if it alternates between `healthy` and `pending` on each page load, the retry is not working or the runtime binary is old.

---

### `isKonductor: false` on both pods

**Cause:** Leader election is not running, OR the runtime binary predates the `SetIsKonductor` call in `Kordinate()`.

Check the Lease object:
```bash
kubectl get lease orkestra-konductor -n orkestra-system-01 -o yaml
```

The `holderIdentity` field shows the current leader's pod hostname. If the Lease doesn't exist, leader election hasn't started.

Check that `LEADER_ELECTION=true` is set on the runtime pods:
```bash
kubectl exec -n orkestra-system-01 deploy/orkestra-runtime -- env | grep LEADER_ELECTION
```

---

## Quick reference — expected responses by pod type

| Field | Leader | Follower |
|-------|--------|----------|
| `isKonductor` | `true` | `false` |
| `OrkReady` | `true` | `true` |
| `healthy` | `true` (after warmup) | `false` |
| `state` | `healthy` | `pending` |
| `totalReconciles` | > 0 | `0` |
| `lastReconcile` | RFC3339 timestamp | `"not started"` |
| `resourceCount` | > 0 (informer cache) | > 0 (informer cache) |

`resourceCount` is the same on both pods — it comes from the informer cache, which is synced from the Kubernetes API independently of leadership.
