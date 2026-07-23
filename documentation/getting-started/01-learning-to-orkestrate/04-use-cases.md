# Use-cases Pack

Real operational patterns. Each use-case is a self-contained example set with a shared CRD and a root README.

```bash
ork init --pack use-cases
```

---

## Normalize

Cleaning and defaulting user input before it reaches the reconciler — no webhooks, no custom validators.

```bash
cd normalize/01-string-cleanup
ork run
```

| Example | What it teaches |
|---|---|
| `01-string-cleanup` | Lowercase and trim user-submitted strings at the normalize phase. Downstream resources always see clean values regardless of what the user submitted. |
| `02-image-normalization` | Rewrite shorthand image references to canonical form. One place to enforce image registry policy. |
| `03-defaults-without-webhook` | Apply field defaults at normalize time — no defaulting webhook needed. |
| `04-webservice` | Full-service normalization: slug the name, default the port, inject the environment prefix, canonicalize the image. Normalize as the pre-processing layer for everything that follows. |

---

## Enrich

Live cluster state embedded into status — powers Control Center visibility and cross-operator decisions.

```bash
cd enrich/01-pod-health
ork run
```

| Example | What it teaches |
|---|---|
| `01-pod-health` | Always-on pod enrichment. `_podHealth` embedded in status on every reconcile: ready count, restart count, crash detection. |
| `02-warning-events` | Conditional event enrichment. Kubernetes warning events only fetched when the operator detects degraded state — `when:` keeps the cost near-zero in steady state. |
| `03-rollout-observer` | Conditional ReplicaSet enrichment. Rollout history only fetched during active rollouts — `anyOf:` limits calls to when they actually matter. |

---

## Profiles

Named presets that expand at load time — teams declare intent, the profile fills in the details.

```bash
cd profiles/01-resource
ork run
```

| Example | What it teaches |
|---|---|
| `01-resource` | CPU and memory profiles (`small`, `standard`, `compute-heavy`). Team writes `profile: standard`; the operator fills in requests and limits. |
| `02-security` | Security context profiles. `restricted` applies non-root, read-only filesystem, dropped capabilities. |
| `03-probes` | Liveness, readiness, and startup probe profiles. Protocol-aware defaults — HTTP, EXEC, gRPC. |
| `04-rolling-update` | Deployment rollout strategy profiles. `safe` uses zero max-unavailable; `fast` trades availability for speed. |
| `05-pdb` | PodDisruptionBudget profiles. `high-availability` requires majority available; `best-effort` allows full disruption. PDB created and managed automatically. |

---

## Full-stack app

Six patterns that each solve one real problem. Example 06 combines all five into a production-shaped operator.

```bash
cd full-stack-app
ork run   # runs the komposer — all six operators together
```

| Example | What it teaches |
|---|---|
| `01-multi-region` | `forEach` over a list. One CR, one `spec.regions` list, one Deployment per region. No per-region code. |
| `02-external-gate` | `external:` with two calls: a blocking health check and a non-blocking feature flag fetch. `continueOnError: false` vs `true` side by side. |
| `03-cross-crd` | `cross:` between two CRDs. `DatabaseBackedApp` waits for `ManagedDatabase` to reach `Ready`. The database endpoint flows into a ConfigMap automatically. |
| `04-once-secret` | `once: true` on a Secret. Generated exactly once. Stable credentials across the lifetime of the CR. |
| `05-anyof` | `anyOf:` for OR conditions. Combines with `when:` (AND) for `(all of these) AND (any of those)` logic without Go code. |
| `06-full-stack` | All five patterns together: forEach, external, cross-CRD, once-secret, anyOf. |

---

## Multi-region map

`forEach` over a map — each entry carries its own per-region configuration.

```bash
cd multi-region-map
ork run
```

| Example | What it teaches |
|---|---|
| `app` | Map forEach where each region has its own replica count and port. Contrast with `full-stack-app/01-multi-region` (list forEach, uniform config) — the map enables per-entry differences. |

---

## External

HTTP calls before any resource group runs. Results available in template expressions, `when:` conditions, and status fields. All ten examples use `--dev-server` — no real upstream services needed.

```bash
cd external/01-health-gate
ork run --dev-server
```

| Example | What it teaches |
|---|---|
| `01-health-gate` | Required upstream check. `continueOnError: false` vs `true`. Phase state machine surfaces failure in status. |
| `02-config-inject` | Config fetch on every reconcile. Response body embedded into a ConfigMap. Service outage leaves the last-written config in place. |
| `03-image-signing` | "Once per image" pattern. Call only fires when `spec.image` changes. 4xx writes `rejectedImage` and suppresses retries. 5xx leaves the gate open. |
| `04-chained` | Sequential calls. Second call uses the first call's response body as its auth token. |
| `05-feature-flags` | External call drives a resource attribute — not a gate. `onCreate` with `reconcile: true`. Flip a flag; cluster converges on the next reconcile. |
| `06-sbom-cosign` | Two chained supply chain checks. SBOM gates cosign — a vulnerable image never reaches the signature service. Both rejection paths write the same `rejectedImage` gate. |
| `07-vault-secret-gate` | Vault KV v2 secret readiness. Runs every reconcile for expiry detection. Distinguishes `SecretExpired` (403) from `SecretMissing` (404) in status. |
| `08-opa-policy` | OPA policy enforcement. `continueOnError: false` ensures the Deployment never exists without a passing check. Full OPA response in status for observability. |
| `09-cert-readiness` | TLS certificate issuance gate. Deployment removed when cert goes pending; restored when issued. Toggle endpoint for local demo. |
| `10-motif-composition` | External calls, admission rules, and status as reusable motifs. Four motifs parameterized via `inputs:`, bound by two katalogs via `with:`. `admission`, `vault-gate`, and `opa-policy` shared by both; `supply-chain` by the WebApp only. A Komposer runs both operators. |

---

## Temporal

Time-dependent workloads without CronJobs. The reconciler re-evaluates built-in time notes on every resync and converges the cluster to the correct state for *right now*.

```bash
cd temporal/01-business-hours
ork run
```

| Example | What it teaches |
|---|---|
| `01-business-hours` | Provision a dev environment Mon–Fri 09:00–18:00 UTC and deprovision it automatically when the window closes. `inBusinessHours` and `nextBusinessHour` user-defined notes power the status message. |
| `02-maintenance-window` | Weekly database maintenance window (Sun 02:00–04:00 UTC). Read replicas scale to zero, a ConfigMap signals downstream load balancers, replicas restore when the window closes. |
| `03-regional-peak` | CDN edge operator with per-region peak-hour replica scaling. US-East, EU-Central, and APAC each scale between `peakReplicas` and `baseReplicas` based on their local UTC window. One operator, no CronJobs. |
| `04-autoscale` | Business-hours autoscaling. `autoscale:` driven by a user-defined note — `inBusinessHours` composes `weekday` and `timeInWindow` into one named boolean. `target` jump scaling; `cooldown` prevents oscillation at window boundaries. |

---

## Workload Autoscaler

Orkestra collapses KEDA, VPA, and custom scaling controllers into one field on a Deployment. The same condition engine that controls resource creation controls scaling — `external:`, `cross:`, and time conditions all work as scaling signals.

```bash
cd workload-autoscaler/01-time-based
ork run
```

| Example | What it teaches |
|---|---|
| `01-time-based` | `autoscale:` with `target` jump scaling; `when: time:` and `when: dayOfWeek:` as scaling signals; `negate: true` to invert a condition. |
| `02-external-api` | `external:` feeds live queue depth into `autoscale:` conditions; `increment`/`decrement` step scaling; `continueOnError: true` keeps the operator alive when the metrics endpoint is down. Run with `ork run --dev-server`. |
| `03-cross-operator` | `cross:` reads a sibling CRD's queue depth from the informer cache; workers scale with the producer's load. Any field a sibling publishes to status is a valid scaling signal. |
