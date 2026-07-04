# Endpoint Access Control

Orkestra exposes a live HTTP API for every CRD in a Katalog. Two independent controls let you restrict who can see what:

| Control | Scope | What it restricts |
|---|---|---|
| `endpoints:` | CRD | Which HTTP paths the runtime registers |
| `crossAccess:` | CRD or Katalog | Whether sibling operators can read this CRD via `cross:` |

They are independent. `crossAccess: false` does not disable HTTP endpoints. `endpoints: enabled: false` does not prevent cross reads. Use both together for a CRD that is completely invisible to all consumers.

---

## `endpoints:` — controlling HTTP visibility

### Disable one endpoint

```yaml
spec:
  crds:
    payment:
      endpoints:
        health: false
```

`/katalog/payment/health` returns 404. All other endpoints remain registered. Use when health details — consecutive fails, last error, uptime — are sensitive but you still want metrics and CR state accessible from dashboards.

`info: false` disables `/katalog/payment` (the config and metrics endpoint) while leaving health up.

### Disable all endpoints

```yaml
spec:
  crds:
    credential:
      endpoints:
        enabled: false
```

The runtime does not register any per-CRD paths for `credential`. `/katalog/credential`, `/katalog/credential/health`, and `/katalog/credential/cr` all return 404. The CRD still appears in the top-level `/katalog` summary count — it is counted but not drillable.

The reconciler runs normally. Nothing about `enabled: false` affects reconciliation, workers, or queue behaviour.

### What always remains

Regardless of `endpoints:` settings, the top-level `/katalog` endpoint always includes the CRD in its summary count. There is no way to make a CRD invisible to the top-level listing except by disabling the CRD — only its per-CRD paths can be suppressed.

---

## `crossAccess:` — controlling cross-operator reads

`crossAccess` controls whether other Katalogs can read this CRD's CR state via the `cross:` block — in-binary or cross-binary via ONCOP.

### At the CRD level

```yaml
spec:
  crds:
    keyrotation:
      crossAccess: false
```

Any `cross:` reference that targets `keyrotation` resolves to empty. No error is raised in the reading operator — the read silently returns nothing.

### At the Katalog level

```yaml
crossAccess: false

spec:
  crds:
    payment:
      ...
    ledger:
      crossAccess: true   # overrides the Katalog default for this CRD only
      ...
```

`crossAccess: false` at the Katalog level closes all CRDs in that Katalog by default. Individual CRDs can override back to `true`.

---

## Combining both — fully dark

```yaml
spec:
  crds:
    keyrotation:
      crossAccess: false
      endpoints:
        enabled: false
```

`keyrotation` is now invisible to every consumer:

- Sibling operators using `cross:` receive empty results
- HTTP clients hitting `/katalog/keyrotation` receive 404
- The Control Center cannot drill into it — no health badge, no CR list
- It still appears in the top-level `/katalog` count

The operator reconciles silently. Other CRDs in the same Katalog are unaffected.

---

## When to use each

| Situation | Recommendation |
|---|---|
| Health details are business-sensitive but dashboards need metrics | `endpoints: health: false` |
| Operator must be completely opaque to HTTP consumers | `endpoints: enabled: false` |
| Operator manages secrets or keys — no sibling should read its CR state | `crossAccess: false` |
| Maximum isolation — credentials, key management, audit trails | `crossAccess: false` + `endpoints: enabled: false` |

---

---

## Try it

```bash
ork init --pack advanced/19-endpoint-control
# Follow the README examples
```

---

## Further Reading

- **[Endpoint reference](endpoints.md)** — full list of per-CRD HTTP paths
- **[ONCOP](../oncop/)** — cross-binary observation protocol that `crossAccess` gates
