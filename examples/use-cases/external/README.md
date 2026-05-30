# External Examples

Four focused examples showing what the `external:` block does and how real teams use it — from gating deployments on upstream health to injecting live config, signing images, and chaining sequential API calls.

| Example | What it teaches |
|---|---|
| [01 — Health Gate](01-health-gate/README.md) | Required upstream check — Deployment only created when the health endpoint returns 200 |
| [02 — Config Inject](02-config-inject/README.md) | Optional config fetch — response body embedded into a ConfigMap on every reconcile |
| [03 — Image Signing](03-image-signing/README.md) | Idiomatic "once per image" pattern — call only fires when `spec.image` changes |
| [04 — Chained Calls](04-chained/README.md) | Sequential calls — second call uses the first call's response body as its auth token |

All four share one CRD (`crd.yaml` at the root of this directory).

---

**Further reading:** [External concept doc](../../documentation/concepts/operatorbox/07-external/index.md)
