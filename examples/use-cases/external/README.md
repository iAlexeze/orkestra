# External Examples

Five focused examples showing what the `external:` block does and how real teams use it — from gating deployments on upstream health to live flag-driven rollouts.

| Example | What it teaches |
|---|---|
| [01 — Health Gate](01-health-gate/README.md) | Required upstream check — Deployment only created when the health endpoint returns 200 |
| [02 — Config Inject](02-config-inject/README.md) | Optional config fetch — response body embedded into a ConfigMap on every reconcile |
| [03 — Image Signing](03-image-signing/README.md) | "Once per image" pattern — call only fires when `spec.image` changes; 4xx locks out retries |
| [04 — Chained Calls](04-chained/README.md) | Sequential calls — second call uses the first call's response body as its auth token |
| [05 — Feature Flags](05-feature-flags/README.md) | External call drives a resource attribute (replicas) — flip a flag, cluster converges |

All five share one CRD (`crd.yaml` at the root of this directory).

Run any example in isolation from its subfolder, or run them all at once with the komposer at the root.

---

**Further reading:** [External concept doc](https://orkestra.sh/docs/concepts/operatorbox/external/)
