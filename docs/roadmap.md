# Orkestra Roadmap

*Last updated: March 2026*

---

## Where We Are

Orkestra v1.0 is the first complete declarative operator runtime. The core is done. The promise — write a Katalog, run `ork run`, get a production-grade operator — is delivered.

What exists today:

- **Katalog** — declare CRDs, reconcile templates, dependencies, workers, resync
- **Komposer** — compose Katalogs from files, URLs, Helm charts, environment variables
- **Dynamic mode** — zero-code operators, no compiled types, no generation step
- **Typed mode** — compiled Go types, Go hooks, custom constructors
- **OrkestraRegistry** — Deployments, Services, Secrets, ConfigMaps, ServiceAccounts, Jobs, CronJobs, Pods
- **Declarative conversion** — version conversion rules in YAML, no Go code, no separate webhook
- **Built-in conversion webhook** — Orkestra serves `/convert` over HTTPS, metrics exposed
- **Template resolver** — full Go `text/template` against live CR objects
- **Dependency graph** — topological startup, cycle detection, missing CRD retry
- **GenericReconciler** — three-path dispatch: templates, Go hooks, custom constructor
- **DependencyKontroller** — per-CRD worker pools, safe reconcile, panic recovery
- **KonductorElection** — leader election, warm cache failover
- **Health API** — `/health`, `/ready`, `/metrics`, `/katalog/*` per CRD
- **Prometheus metrics** — reconciliation and conversion metrics, per‑CRD labels
- **CLI** — `ork init`, `ork validate`, `ork template`, `ork generate runtime`, `ork run`, `ork version`, `ork status`
- **Install** — `curl | bash` installer for macOS and Linux
- **Documentation** — architecture, CLI, registry, conversion, templating, extending, use cases, whitepaper

---

## Completed Milestones (Already Shipped)

| Item | Status |
|------|--------|
| Runtime template interpretation | ✅ Dynamic CRDs require no `generate` step |
| Declarative version conversion | ✅ Conversion rules in YAML, no Go code |
| Built‑in conversion webhook | ✅ Orkestra serves `/convert` over HTTPS |
| Conversion metrics | ✅ `orkestra_conversion_requests_total`, `orkestra_conversion_duration_seconds` |
| Multi‑source authentication | ✅ Bearer, GitHub, Basic auth with `fromEnv` |
| `ork status` command | ✅ Live view of CRD health, workers, queue, resources |
| End‑to‑end tests | ✅ Website, platform‑namespace, komposer examples |
| `ork validate` full error coverage | ✅ Clear, actionable error messages for all invalid Katalog patterns |
| Module path migration | ✅ `github.com/orkestra-sh/orkestra` |
| Helm chart for Orkestra | ✅ Production-ready deployment chart |

---

## Phase 1 — Stability (Q2 2026)

**Goal:** Ensure v1.0 is production‑ready for early adopters.

The architecture is right. The features are complete. Phase 1 focuses on hardening, coverage, and removing the last friction before Orkestra is in front of real users.

| Item | Status |
|------|--------|
| `ork init` end‑to‑end test | ✅ Complete |
| Integration tests for all examples | ✅ Complete |
| `ork validate` full error coverage | ✅ Complete |
| Module path migration | ✅ Complete |
| Helm chart for Orkestra | ✅ Complete |

**Phase 1 complete.** Orkestra is ready for production evaluation.

---

## Phase 2 — Adoption (Q3–Q4 2026)

**Goal:** Remove every obstacle between "interested" and "running in production."

### `ork dashboard`

A terminal UI showing the live state of a running operator.

```
CRD                    Workers   Queue   Health   Reconciles   Errors   Conversions
website                2/2       0       ✅        1,247        0        47
platformnamespace      2/2       3       ✅        412          1        0
application            4/4       0       ✅        8,891        0        0
database               2/2       0       ⚠️         201          4        12
```

Real-time queue depth, worker utilization, error rates, conversion statistics, and the dependency graph.

### Additional source types

```yaml
sources:
  s3:
    - bucket: my-org-katalogs
      key: platform/crds.yaml
      region: us-east-1

  configMap:
    - name: platform-crds
      namespace: orkestra-system
      key: katalog.yaml
```

S3 and ConfigMap are the two most-requested sources.

### `ork diff`

Show what would change in a running operator if the Katalog were updated. Like `kubectl diff` for operator configuration.

```bash
ork diff --katalog ./katalog-v2.yaml
# ~ website: workers 2 → 4
# + logging (new CRD)
# - legacy-resource (removed)
```

### Performance benchmarks and stress testing

Numbers for reconcile throughput, queue latency, informer memory usage at 100+ CRDs. Quality gates to be published.

### `ork lint`

Deeper Katalog analysis beyond `ork validate`. Detects patterns that are valid but likely wrong:

```bash
ork lint --katalog ./katalog.yaml
# WARN website: reconcile: true on all resources — consider onReconcile
# WARN database: workers: 1 — single worker may cause latency spikes
# WARN application: resync: 1s — very short resync interval for 500+ CRDs
```

---

## Phase 3 — Ecosystem (2027)

**Goal:** Make Orkestra the standard way to distribute and consume operator definitions.

### Public Katalog registry

The OrkestraRegistry already exists as the community home for resource implementations. The next layer is a registry for complete Katalogs — reusable operator definitions that anyone can consume.

```bash
ork registry search postgres
ork registry pull konduktor-io/postgres@v14
ork run --from konduktor-io/postgres@v14

ork registry publish ./my-katalog.yaml --name myorg/postgres
```

The same model as Helm Hub, but for operator behavior rather than deployment manifests.

### Katalog versioning and dependencies

```yaml
name: postgres
version: 14.5.0
orkestra: ">=1.0.0"
dependencies:
  - storage-class@v1
  - monitoring@>=2.0
```

Semantic versioning, dependency resolution, compatibility checks. The package management layer that the registry needs.

### `ork dashboard` — web version

The terminal dashboard from Phase 2 extended to a web UI. Useful for organisations managing large numbers of operators across clusters.

### CNCF Sandbox submission

Target Q1 2027. Requires production usage at multiple organisations — which Phase 2 adoption efforts should produce. The CNCF sandbox gives Orkestra vendor neutrality and community governance.

---

## What We Are Not Building

These are deliberate non-goals:

**Admission webhooks.** Orkestra is a reconciler runtime. Webhooks are a synchronous validation mechanism in the API server request path. Different tool, different model. No plans to add.

**Multi-cluster federation.** Orkestra manages CRDs within one cluster. Cross-cluster operations belong to a different architectural layer.

**Replacing controller-runtime.** Orkestra is not a replacement for controller-runtime. It is a higher-level abstraction that makes the common case trivial and defers to Go code when needed. The goal is to make even the most complex use cases declarative.

---

## Contributing

The highest-value contributions right now:

| Area | What Helps Most |
|------|-----------------|
| **Testing** | Run Orkestra in real environments and report what breaks |
| **Registry implementations** | Add new resource types to OrkestraRegistry |
| **Examples** | More Katalog examples showing real operator patterns |
| **Documentation** | Edge cases, gotchas, things that weren't obvious |

Open a [GitHub issue](https://github.com/orkestra-sh/orkestra/issues) or start a [Discussion](https://github.com/orkestra-sh/orkestra/discussions) for anything not covered above.

---

## Release Schedule

| Version | Target | Focus |
|---------|--------|-------|
| v1.0.0 | Q2 2026 | Production‑ready core (current) |
| v1.1.0 | Q3 2026 | `ork dashboard`, `ork diff`, `ork lint` |
| v1.2.0 | Q4 2026 | Additional sources (S3, ConfigMap) |
| v2.0.0 | 2027 | Public registry, versioning, CNCF Sandbox |

---

**Want to try it out?** [Start here](./getting-started/index.md).