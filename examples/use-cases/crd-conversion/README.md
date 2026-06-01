# CRD Conversion

Two approaches to the same problem: a CronJob operator that accepts multiple schedule formats. The Kubebuilder CronJob tutorial rewritten in YAML — with and without a conversion webhook.

| Example | What it teaches |
|---------|-----------------|
| [with-webhooks](with-webhooks/README.md) | Multi-version CRD with Orkestra Gateway handling v1 ↔ v2 conversion. No Go webhook server to deploy — Orkestra registers and serves the webhook itself. |
| [without-webhooks](without-webhooks/README.md) | Single-version CRD. `normalize` collapses cron-string and structured-object formats into one canonical schedule before reconcile runs. No conversion webhook needed at all. |

Both approaches produce identical runtime behavior. The difference is in how you model the problem: multi-version API surface vs. single-version with input normalization.

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
