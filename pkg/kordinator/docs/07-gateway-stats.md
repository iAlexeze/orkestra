# 07 — Gateway stats and the advertise pattern

## The problem

The runtime and gateway are separate processes. The runtime serves `/katalog`
with reconciler stats (workers, queue depth, error rate). The gateway serves
admission, conversion, and deletion-protection webhooks and is the only process
that counts those events.

A control center reading only the runtime's `/katalog` would see zeros for all
webhook-related stats — accurate, but incomplete.

## Design

Each process serves its own `/katalog` endpoint. Neither pushes to the other.

| Process | `/katalog` content |
|---------|-------------------|
| Runtime | Reconciler health — workers, queue depth, error rate, resource count |
| Gateway | Webhook stats — admission, conversion, deletion protection, namespace protection |

The runtime advertises the gateway URL via the `"gatewayEndpoint"` field in its
`/katalog` response. The control center reads that field, fetches the gateway's
`/katalog`, and merges the two responses per CRD using the **GVR string as the
merge key**.

```
control center
  │
  ├── GET runtime:/katalog
  │     → crds[].gvr, reconciler stats, "gatewayEndpoint": "http://..."
  │
  └── GET gatewayEndpoint:/katalog
        → crds[].gvr, webhook stats
```

## GVR as the merge key

Both processes key their per-CRD stats by the GVR string
(`"group/version/resource"`, e.g. `"demo.io/v1/websites"`).

- `pkg/gateway/webhook` uses `crdGVRKey(group, version, resource)` when recording stats.
- `pkg/gateway/handlers` uses the same `gvrKey(group, version, resource)` helper when
  building the response.
- The control center matches entries by the `"gvr"` field in each CRD object.

## Configuring the gateway endpoint

Set `ORK_GATEWAY_ENDPOINT` on the runtime deployment to the HTTP base URL of the
companion gateway, e.g.:

```yaml
env:
  - name: ORK_GATEWAY_ENDPOINT
    value: "http://orkestra-gateway.orkestra-system.svc:8080"
```

The field is optional — when empty, `"gatewayEndpoint"` is omitted from the
response and the control center treats the runtime as standalone (no gateway
stats available).

## Stats accuracy: per-CRD, not process-global

Each CRD gets its own stats instance in the gateway process. Stats are keyed by
GVR and pre-initialized from the Katalog at startup. This means:

- `/katalog/website` returns only `Website` webhook traffic, not total across all CRDs.
- Infra-level events (webhook self-protection, Orkestra Deployment/Service
  protection) are counted in `"infraProtection"` at the top level, separate
  from per-CRD counters.

## Gateway standalone mode

When the runtime is not deployed (zero-CRD Katalog + deletion protection), the
gateway is the only process running. The control center can point directly at the
gateway's `/katalog` endpoint. The `"source": "gateway"` field in the response
distinguishes it from a runtime response.

→ Back: [06-handlers.md](06-handlers.md)
