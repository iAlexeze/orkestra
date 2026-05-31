# External 10 — Motif Composition

Policy logic as shared infrastructure. Four motifs — admission rules, external supply chain checks, Vault secret gating, and OPA policy enforcement — are each written once and imported by the katalogs that need them. Two katalogs (`WebApp`, `Worker`) compose from those motifs via `with:`. A komposer runs both operators together.

**What you learn:** how admission rules, external calls, and status fields are all parameterizable in a motif; how `with:` decouples the motif's input names from the CR's field names; and why shared motifs mean a change in one file propagates to every operator that imports it.

---

## Structure

```
motifs/
  admission.yaml      # shared validation rules — spec.image and spec.serviceUrl required
  supply-chain.yaml   # inputs: serviceUrl, image → SBOM + cosign calls + status
  vault-gate.yaml     # inputs: serviceUrl, vaultToken → Vault KV v2 check + status
  opa-policy.yaml     # inputs: serviceUrl, policy → OPA decision check + status

katalog-webapp.yaml   # WebApp — imports all four motifs
katalog-worker.yaml   # Worker — imports admission + vault-gate + opa-policy
komposer.yaml         # runs both operators in one runtime
```

**WebApp** imports all four motifs: admission validation, supply chain verification, vault secret readiness, and OPA policy check. The Deployment is only created when all three runtime checks pass.

**Worker** skips supply chain verification — internal images are already trusted — but still imports the admission motif (same validation rules) and the vault and OPA motifs.

`admission`, `vault-gate`, and `opa-policy` are written once and imported by both katalogs. `supply-chain` is imported by WebApp only. No validation rule or external call is repeated anywhere.

---

## How the parameterization works

Inside `motifs/vault-gate.yaml`:

```yaml
inputs:
  - name: serviceUrl
    required: true

resources:
  external:
    - name: vaultSecret
      url: "{{ .inputs.serviceUrl }}/vault/v1/secret/data/apps/{{ .metadata.name }}"
      token: "{{ .inputs.vaultToken }}"
```

Inside `katalog-webapp.yaml`:

```yaml
imports:
  - motif: ./motifs/vault-gate.yaml
    with:
      serviceUrl: "{{ .spec.serviceUrl }}"
```

Inside `katalog-worker.yaml`:

```yaml
imports:
  - motif: ./motifs/vault-gate.yaml
    with:
      serviceUrl: "{{ .spec.serviceUrl }}"
```

Same motif file. Same external call logic. Different CRDs, different spec field names — `with:` is the adapter.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Start the operator

```bash
ork run --dev-server 
```

Both the `webapp-composed` and `worker-composed` operators start. The dev server handles all motif endpoints on `:9999`.

---

## Step 3 — Open the Control Center

```bash
ork control   # http://localhost:8081 → orkestra/orkestra
```

You will see two CRD types in the sidebar: **WebApp** and **Worker**.

---

## Step 4 — Apply both CRs

```bash
kubectl apply -f cr-webapp.yaml
kubectl apply -f cr-worker.yaml
```

Wait one reconcile (~30s). The WebApp runs all three motifs in sequence — SBOM, cosign, vault, policy. The Worker skips supply chain and runs vault + policy only.

```bash
kubectl get webapp my-app -o yaml | grep -A12 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  supplyChainPhase: Verified
  verifiedImage: nginx:1.25
  vaultPhase: SecretReady
  secretStatus: "200"
  policyPhase: Allowed
```

```bash
kubectl get worker my-worker -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  vaultPhase: SecretReady
  secretStatus: "200"
  policyPhase: Allowed
```

---

## Step 5 — Observe what the Worker does not run

Check the metrics:

```bash
curl -s localhost:8080/metrics | grep external_call
```

The Worker shows calls to `vaultSecret` and `policy` only. No `sbom` or `cosign` calls — the supply-chain motif was not imported, so those endpoints are never hit. Policy changes in one motif file propagate to every operator that imports it.

---

## Step 6 — Test a rejection

Apply a webapp with an unsigned image:

```bash
kubectl patch webapp my-app --type=merge -p '{"spec":{"image":"nginx:unsigned"}}'
```

The supply-chain motif calls cosign, gets 403, writes `rejectedImage`. The Deployment gate closes. The Worker is unaffected — it does not import supply-chain.

Expected logs:

```json
{
  "level":"warn","request_id":"28f9b66b-63af-4247-8be2-a9b6b40efaa8",
  "crd":"demo.orkestra.io/v1, Kind=WebApp","resource":"default/my-app",
  "call":"cosign","url":"http://localhost:9999/cosign/verify",
  "error":"HTTP 403","time":1780254517,"message":"external call failed"
}
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
