# CRD Conversion

Two approaches to the same problem: a CronJob operator that accepts multiple schedule formats. The Kubebuilder CronJob tutorial rewritten in YAML — with and without a conversion webhook.

| Example | What it teaches |
|---------|-----------------|
| [with-webhooks](with-webhooks/README.md) | Multi-version CRD with Orkestra Gateway handling v1 ↔ v2 conversion. No Go webhook server to deploy — Orkestra registers and serves the webhook itself. |
| [without-webhooks](without-webhooks/README.md) | Single-version CRD. `normalize` collapses cron-string and structured-object formats into one canonical schedule before reconcile runs. No conversion webhook needed at all. |
| [with-serve-translation](with-serve-translation/README.md) | Single-version CRD. Callers submit a flat cron string via the Gateway API; `serve.fields.values` fans it out to the five structured schedule fields before the CR reaches the API server. No conversion, no normalize — the transformation lives entirely in the serve layer. |

All three produce identical runtime behavior. The difference is where the cron string is translated: at the API server (webhook), at reconcile time (normalize), or at the serve layer (field translation).

---

## E2E

Every example ships with a runnable `e2e.yaml`. Run a single example end-to-end:

```bash
cd with-webhooks && ork e2e
cd without-webhooks && ork e2e
```

Or run both together:

```bash
ork e2e -f e2e.yaml
```
