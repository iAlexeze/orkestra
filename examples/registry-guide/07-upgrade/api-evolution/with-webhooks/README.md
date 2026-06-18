# 07 — with-webhooks: CRD API evolution with bidirectional v1↔v2 conversion

> **Before you start:** Replace `myorg` with your actual registry path in [`katalog.yaml`](katalog.yaml) (the motif import `oci://ghcr.io/myorg/motifs/web-service:v1.1.0`).

The WebApp CRD grows up. `spec.port` and `spec.host` (flat, v1) become `spec.expose.port` and `spec.expose.host` (structured, v2). A new field — `spec.expose.protocol` — has no v1 equivalent.

Team A's CI pipelines and Terraform still write v1 CRs. Team B is ready for v2. The API server must serve both. Orkestra Gateway handles conversion — in-process, no extra pod.

## The version change

```yaml
# v1 — flat, terse
spec:
  image: ghcr.io/orkspace/orkestra-dev-server:0.7.5
  port: 9999
  host: api.example.com

# v2 — structured, self-documenting, extensible
spec:
  image: ghcr.io/orkspace/orkestra-dev-server:0.7.5
  expose:
    port: 9999
    host: api.example.com
    protocol: HTTPS   # new in v2 — no v1 equivalent
```

Orkestra expresses the round-trip as four template lines:

```yaml
conversion:
  paths:
    - from: v1
      to: v2
      spec:
        expose:
          port:     "{{ .spec.port }}"
          host:     '{{ default "" .spec.host }}'
          protocol: HTTP

    - from: v2
      to: v1
      spec:
        port: "{{ .spec.expose.port }}"
        host: '{{ default "" .spec.expose.host }}'
```

`spec.expose.protocol` is dropped on v2→v1 — no v1 equivalent. The object is stored once in v2. Orkestra converts on read when v1 is requested.

## Validate first

Before generating or applying anything, check what you are authorizing:

```bash
ork validate --full
```

Two things to notice in the output: `webapp-v2` lists `customresourcedefinitions patch` — that is the gateway registering the conversion webhook endpoint on the CRD object. The `gateway` section lists `secrets` — that is Orkestra provisioning and rotating the TLS certificate. Nothing else. What you see here is exactly what gets applied.

---

## Steps

### 1. Apply the CRD

```bash
kubectl apply -f crd.yaml
```

### 2. Generate and apply the operator bundle

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

### 3. Install Orkestra with Gateway

`gateway.enabled=true` starts the `/convert` endpoint and manages TLS automatically:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

### 4. Apply both CRs

```bash
# v2 CR — stored directly
kubectl apply -f cr-v2.yaml

# v1 CR — Orkestra converts to v2 before storage
kubectl apply -f cr-v1.yaml
```

### 5. Verify the round-trip

```bash
# v1 CR read back as v1 — flat fields are reconstructed
kubectl get webapps.v1.rkguide.demo my-webapp-v1 -n default -o yaml | grep port:
# port: 9999

# Same object read as v2 — expose block is present
kubectl get webapps.v2.rkguide.demo my-webapp-v1 -n default -o yaml | grep -A4 expose:
# expose:
#   host: ""
#   port: "9999"
#   protocol: HTTP
```

### 6. Verify reconciliation

```bash
kubectl get webapps -n default
# NAME           IMAGE                                     PORT   PROTOCOL   PHASE     AGE
# my-webapp-v1   ghcr.io/orkspace/orkestra-dev-server...  9999   HTTP       Running   8s
# my-webapp-v2   ghcr.io/orkspace/orkestra-dev-server...  9999   HTTP       Running   6s
```

Both CRs — one written in v1 format, one in v2 format — produce identical Deployments and Services.

### 7. Observe conversions

Open the Control Center for live stats:

```bash
kubectl port-forward svc/orkestra-cc 8081:8081 -n orkestra-system
# username:password → orkestra
```

---

## Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

Simulate and e2e run automatically before the artifact is published. Gate results are written as OCI annotations — visible to any consumer via `ork inspect webapp-operator:v2.0.0`.

> **Note:** The simulate gate here uses a native v2 CR and checks steady state only — no ops assertions. CRD conversion via the Gateway cannot be exercised offline. See [what simulate cannot cover](https://orkestra.sh/docs/concepts/simulate/limitations/#what-simulate-cannot-cover).

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Files

| File | Purpose |
|---|---|
| `crd.yaml` | v1 and v2 schemas with conversion webhook config |
| `katalog.yaml` | Operator — motif import, conversion paths, status |
| `cr-v1.yaml` | v1 CR with flat spec.port |
| `cr-v2.yaml` | v2 CR with structured spec.expose |
| `simulate.yaml` | Smoke-test with native v2 CR — no ops assertions (v1 conversion requires the live Gateway) |
| `e2e.yaml` | Full lifecycle test |
| `cleanup.sh` | Teardown script |

---

→ Compare: [without-webhooks](../without-webhooks/README.md) — same result, no Gateway required
