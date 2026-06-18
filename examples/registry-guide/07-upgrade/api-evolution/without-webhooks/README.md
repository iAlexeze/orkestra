# 07 — without-webhooks: CRD API evolution via normalize

> **Before you start:** Replace `myorg` with your actual registry path in [`katalog.yaml`](katalog.yaml) (the motif import `oci://ghcr.io/myorg/motifs/web-service:v1.1.0`).

Same WebApp domain. Same two formats. No conversion webhook. No multi-version CRD. No TLS. No cert-manager.

The key insight: conversion webhooks solve one problem — the API server stores objects in one version while clients request another. `normalize` eliminates the problem at the source. Orkestra collapses both input formats into a canonical shape before any reconcile logic runs. The API server sees one version. There is nothing to convert.

## Both formats are valid

```yaml
# Old format — spec.port at top level (still accepted)
spec:
  image: ghcr.io/orkspace/orkestra-dev-server:0.7.5
  port: 9999

# New format — spec.expose (preferred)
spec:
  image: ghcr.io/orkspace/orkestra-dev-server:0.7.5
  expose:
    port: "9999"
    host: ""
```

The operator handles both identically. Users migrate CRs on their own schedule.

## How normalize works

```yaml
normalize:
  spec:
    # v2: spec.expose present → use expose.port directly
    # v1: spec.expose absent  → lift spec.port
    expose.port: '{{ if .spec.expose }}{{ .spec.expose.port }}{{ else }}{{ .spec.port }}{{ end }}'
    # v2: spec.expose present → use expose.host directly
    # v1: spec.expose absent  → lift spec.host
    expose.host: '{{ if .spec.expose }}{{ .spec.expose.host | default "" }}{{ else }}{{ .spec.host | default "" }}{{ end }}'
```

The keys use dot notation (`expose.port`) instead of nested YAML. normalize maps each key to a template string — if you wrote `expose:` as a nested block, YAML would parse the value as a map object instead of a string and template rendering would fail.

The guard is `if .spec.expose`, not `if .spec.expose.port`. Go templates evaluate arguments before calling any function — checking the nested field directly when the parent is absent causes a nil pointer error at runtime. Guarding on the parent block first keeps the inner access safe.

Before any mutation, validation, or template rendering runs, `.spec.expose.port` is always set — whether the CR used `spec.port` or `spec.expose.port`.

After normalize:
- All downstream logic uses `spec.expose.port` — no branching needed anywhere.
- The stored object in etcd is **unchanged** — normalize is not a mutation webhook.
- Old CRs continue to reconcile correctly.

## Migrating to pure structured format

Once all CRs are updated to use `spec.expose`:

1. Remove the `normalize` block.
2. Add a validation rule: `field: spec.port, operator: notExists, action: deny`.
3. Re-run `ork simulate` + `ork e2e` — gates pass.
4. Publish v1.3.0.

No webhook. No cert. No Go. No stored object migration.

---

## Validate first

Before generating or applying anything, check what you are authorizing:

```bash
ork validate --full
```

Single-version CRD, no conversion webhook — the output has no `gateway` section. What you see here is exactly what gets applied.

---

## Steps

### 1. Apply the CRD

```bash
kubectl apply -f crd.yaml
```

### 2. Generate and apply the operator bundle

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

### 3. Install Orkestra

No `--set gateway.enabled=true` needed — single-version CRD, no conversion webhook:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

### 4. Apply both CRs

```bash
kubectl apply -f cr-port-string.yaml      # old format: spec.port: 9999
kubectl apply -f cr-port-structured.yaml  # new format: spec.expose.port: "9999"
```

### 5. Verify both CRs reconcile identically

```bash
kubectl get webapps -n default
# NAME               IMAGE                                     PORT   FORMAT                PHASE     AGE
# webapp-flat        ghcr.io/orkspace/orkestra-dev-server...   9999   flat (deprecated)     Running   6s
# webapp-structured  ghcr.io/orkspace/orkestra-dev-server...   9999   structured            Running   4s
```

Both run identically. normalize collapsed the format difference before reconcile.

---

## Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

Simulate and e2e run automatically before the artifact is published. Gate results are written as OCI annotations — visible to any consumer via `ork inspect webapp-operator:v1.2.0`.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Files

| File | Purpose |
|---|---|
| `crd.yaml` | Single-version WebApp CRD (no conversion block) |
| `katalog.yaml` | Operator with normalize block — motif import, status |
| `cr-port-string.yaml` | Old format: flat `spec.port` |
| `cr-port-structured.yaml` | New format: structured `spec.expose` |
| `simulate.yaml` | Simulate gate — verifies normalize collapses both formats |
| `e2e.yaml` | Full lifecycle test |
| `cleanup.sh` | Teardown script |

---

## with-webhooks vs without-webhooks

| | [with-webhooks](../with-webhooks/README.md) | without-webhooks |
|---|---|---|
| CRD versions | v1 + v2 (multi-version) | v1 only (single version) |
| Conversion webhook | Orkestra Gateway `/convert` | Not needed |
| External clients locked to v1 | Supported | Not supported |
| Gateway required | Yes (`--set gateway.enabled=true`) | No |
| Best for | Public APIs, external consumers | Internal operators, platform teams |

---

→ Next: [08-bad-actor](../../../08-bad-actor/README.md) — audit trail: trace who pushed what and when

