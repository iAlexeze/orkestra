# External Examples

Nine focused examples showing what the `external:` block does and how real teams use it — from gating deployments on upstream health to live flag-driven rollouts and supply chain enforcement.

| Example | What it teaches |
|---|---|
| [01 — Health Gate](01-health-gate/README.md) | Required upstream check — Deployment only created when the health endpoint returns 200 |
| [02 — Config Inject](02-config-inject/README.md) | Optional config fetch — response body embedded into a ConfigMap on every reconcile |
| [03 — Image Signing](03-image-signing/README.md) | "Once per image" pattern — call only fires when `spec.image` changes; 4xx locks out retries |
| [04 — Chained Calls](04-chained/README.md) | Sequential calls — second call uses the first call's response body as its auth token |
| [05 — Feature Flags](05-feature-flags/README.md) | External call drives a resource attribute (replicas) — flip a flag, cluster converges |
| [06 — SBOM and Cosign](06-sbom-cosign/README.md) | Two chained supply chain checks — SBOM gates cosign, both must pass before Deployment is created |
| [07 — Vault Secret Gate](07-vault-secret-gate/README.md) | Secret readiness gate — distinguishes missing vs expired, rotation recovery is automatic |
| [08 — OPA Policy](08-opa-policy/README.md) | Declarative policy enforcement — OPA decision gates the Deployment, denial reason in status |
| [09 — Cert Readiness](09-cert-readiness/README.md) | Certificate issuance gate — Deployment held until cert is issued, toggleable for local demo |
| [10 — Motif Composition](10-motif-composition/README.md) | Policy as shared motifs — vault-gate and opa-policy imported by two katalogs via `with:`, run together in a komposer |

Examples 01–09 share one CRD (`crd.yaml` at the root of this directory). Example 10 defines its own CRDs and motifs — it is self-contained.

Run any example in isolation from its subfolder, or run them all at once with the komposer at the root.

---

## Simulate

Three examples (01, 02, 05) ship with a `simulate.yaml`. Run simulate for a single example:

```bash
cd 01-health-gate && ork simulate --dev-server
```

Or run all three at once with the suite aggregator:

```bash
ork simulate -f simulate.yaml --dev-server
```

---

## E2E

Run the full external suite — deploys the mock dev server into the cluster, runs each example in sequence, then tears everything down:

```bash
ork e2e -f e2e.yaml --dev-server
```

---

**Further reading:** [External concept doc](https://orkestra.sh/docs/concepts/operatorbox/external/)
