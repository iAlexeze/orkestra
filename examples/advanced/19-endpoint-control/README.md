# 19 — Endpoint Control

Every CRD in a Katalog gets a live HTTP API by default. Two independent controls let you restrict visibility:

| Control | What it affects |
|---|---|
| `endpoints:` | Which HTTP endpoints the runtime registers for this CRD |
| `crossAccess:` | Whether sibling operators can read this CRD via the `cross:` block |

They do not imply each other. Use them independently or together.

---

## Examples

### [01-selective-health](./01-selective-health/README.md)

Disables the `/health` endpoint only. Metrics and info remain up.

```yaml
endpoints:
  health: false
```

Use when health details (consecutive fails, last error) are sensitive but you still want the CRD visible in dashboards.

---

### [02-full-disable](./02-full-disable/README.md)

Disables all per-CRD HTTP endpoints. `/katalog/credential` is not registered — health, info, and CR list all return 404. Token is generated once via `randomAlphanumeric` and rotated on schedule. The CRD still appears in the top-level `/katalog` count.

```yaml
endpoints:
  enabled: false
```

---

### [03-fully-dark](./03-fully-dark/README.md)

Combines both controls. `crossAccess: false` closes the CRD to all `cross:` reads from siblings. `endpoints: enabled: false` removes all HTTP access. The `auditlog` CRD alongside it attempts a cross read — and receives empty, recording `keyRef: access-denied`.

```yaml
crossAccess: false
endpoints:
  enabled: false
```

---

## Run the full stack

```bash
# Validate all four operators
ork validate -f komposer.yaml

# Simulate without a cluster
ork simulate

# Start the runtime — applies CRDs and CRs automatically via crdFile/crFiles
ork run -f komposer.yaml

# Run the full E2E suite
ork e2e
```

`simulate.yaml` and `e2e.yaml` at the root use `imports:` to chain each sub-example's own simulate and e2e in sequence. Run any sub-example individually by stepping into its directory.

---

## What "disabled" means

- `endpoints: enabled: false` — the runtime never registers `/katalog/{crd}`, `/katalog/{crd}/health`, or `/katalog/{crd}/cr`. Requests return 404.
- The top-level `/katalog` entry always remains — the CRD appears in the summary count regardless.
- `crossAccess: false` — `cross:` reads targeting this CRD resolve to empty. No error; the reading operator continues normally.
- Neither control affects reconciliation.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Related

- [Endpoint access control](https://orkestra.sh/docs/concepts/live-api/endpoints/)
- [Live API concepts](https://orkestra.sh/docs/concepts/live-api)
