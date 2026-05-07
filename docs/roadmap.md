# Roadmap

*Last updated: March 2026*

---

## Where we are

Orkestra v1.0 is the first complete declarative operator runtime for Kubernetes.
The core promise is delivered: write a Katalog, run `ork run`, get a
production-grade operator — with its own informer, workqueue, worker pool, health
endpoint, metrics, leader election, and drift correction — without writing a single
line of Go.

What exists in production today:

**Runtime**

- Dynamic mode — zero-code operators, no generated types, no compilation step
- Typed mode — Go types, Go hooks, custom constructors when you need them
- GenericReconciler with three-path dispatch: templates, hooks, constructor
- Per-CRD isolation — dedicated informer, workqueue, and worker pool per CRD
- Dependency graph — topological startup order, cycle detection, missing CRD retry
- safeReconcile — panic recovery per CRD, other CRDs unaffected
- Konductor election — leader election with warm-cache follower failover

**Declarations**

- Katalog — CRDs, reconcile templates, workers, resync, dependencies, conversion rules
- Komposer — compose Katalogs from files, URLs, Helm charts, Git and OCI registries
- Conditions (`when:`) — conditional resource creation based on CR field values
- Declarative version conversion — conversion rules in YAML, no Go code
- Declarative validation — deny/warn rules at reconcile and admission time
- Declarative mutation — defaults and overrides at reconcile and admission time

**Distribution**

- OrkestraRegistry — the internal resource library: Deployment, Service, Secret, ConfigMap, ServiceAccount, Job, CronJob, Pod
- Registry source — pull operator patterns from OCI or Git with `sources.registry`
- Five-file pattern structure — enforced at pull time, not at author time
- Authenticated sources — bearer, GitHub, and basic auth from environment variables

**Webhooks**

- Built-in conversion webhook — Orkestra serves `/convert` over HTTPS
- Built-in admission webhooks — `/validate` and `/mutate` on the same HTTPS server
- Auto-registration — ValidatingWebhookConfiguration and MutatingWebhookConfiguration created at startup
- Admission metrics — per-CRD, per-field, per-source (admission vs reconcile)

**Observability**

- Health API — `/health`, `/ready`, `/katalog`, `/katalog/{crd}`, `/katalog/{crd}/health`
- Prometheus metrics — reconcile, conversion, and admission metrics with consistent GVK labels
- Rolling stats — in-process ConversionStats and AdmissionStats with p95 latency

**CLI**

- `ork init` — scaffold a new operator project
- `ork validate` — validate any Katalog or Komposer without a cluster
- `ork run` — start the operator runtime
- `ork status` — live view of all managed CRDs (queries `/katalog`)
- `ork get` / `ork describe` / `ork events` / `ork top` — per-CRD inspection
- `ork template` — preview the merged, validated configuration
- `ork generate registry` — code generation for typed-mode CRDs
- `ork version` — version information

**Distribution**

- Homebrew tap — `brew install orkspace/tap/ork`
- curl installer — `curl -sSL .../install.sh | bash` with GPG signing
- Docker image — GHCR, distroless, two-stage build
- Helm chart — production-ready deployment chart

---

## Phase 1 — Stability (Complete)

**Goal:** Ensure the core is production-ready for early adopters.

The architecture is settled. The features are complete. Phase 1 hardened the
runtime, added test coverage, and removed the last friction before real users.

All Phase 1 milestones are shipped.

---

## Phase 2 — Adoption (Q3–Q4 2026)

**Goal:** Remove every obstacle between "interested" and "running in production."

### `ork dashboard`

A terminal UI showing the live state of a running operator — the `/katalog` endpoint
rendered for the terminal in real time.

```
CRD                  Workers  Queue  Health   Reconciles  Errors  Conversions
website              2/2      0      healthy  1,247       0       47
postgres             4/4      3      healthy  8,891       0       0
platform-namespace   2/2      0      healthy  412         1       0
database             2/2      0      degraded 201         4       12
```

Real-time queue depth, worker utilisation, error rates, conversion latency, and
active validation warnings. The data is already in `/katalog` — the dashboard is
a renderer, not a new data source.

### `ork diff`

Show what would change in a running operator if the Katalog were updated. Like
`kubectl diff` for operator configuration.

```bash
ork diff --file ./katalog-v2.yaml
# ~ website: workers 2 → 4
# ~ website: resync 15s → 30s
# + logging (new CRD)
# - legacy-resource (removed)
```

### `ork lint`

Deeper Katalog analysis beyond `ork validate`. Catches patterns that are valid
but likely wrong:

```bash
ork lint --file katalog.yaml
# WARN website: reconcile: true on all resources — consider onReconcile for drift
# WARN database: workers: 1 — single worker will serialise all reconciles
# WARN platform: resync: 1s — very short interval across 500+ CRDs
```

### `ork registry` CLI

The planned command suite for working with OCI-distributed operator patterns:

```bash
ork registry login ghcr.io
ork registry push ghcr.io/myorg/postgres:v14 ./postgres/v14
ork registry pull ghcr.io/myorg/postgres:v14 ./local
ork registry list ghcr.io/orkspace/orkestra-registry
ork registry search postgres
ork registry info ghcr.io/orkspace/orkestra-registry/postgres:v14
```

OCI patterns are consumable today via direct `oci:` references in Komposers.
The CLI is the first-class interface being built on top.

### Additional source types

```yaml
sources:
  configMap:
    - name: platform-crds
      namespace: orkestra-system
      key: katalog.yaml

  s3:
    - bucket: my-org-katalogs
      key: platform/crds.yaml
      region: us-east-1
      auth:
        type: aws
        fromEnv: AWS_PROFILE
```

ConfigMap and S3 are the two most-requested sources for platform teams managing
Katalog distribution internally.

### Performance benchmarks

Published numbers for reconcile throughput, queue latency, and informer memory
usage at 50+ and 100+ CRDs. Stress test results with quality gates. Production
evidence at scale beyond the current deployment.

---

## Phase 3 — Ecosystem (2027)

**Goal:** Make Orkestra the standard way to distribute and consume operator
definitions. Position for CNCF Sandbox.

### Public Katalog registry

The OrkestraRegistry grows from a resource implementation library to a full
pattern registry — versioned, searchable, consumable with a version reference.

```yaml
sources:
  registry:
    - url: myorg-io/postgres@v1.0.0
      oci: true
    - url: myorg-io/monitoring@v0.3.0
      oci: true
```

Discoverable on Artifact Hub. Community-contributed patterns with maintainer
review. The npm of operator behavior.

### Katalog versioning and dependencies

```yaml
name: postgres
version: 14.2.0
orkestra: ">=1.0.0"
dependencies:
  - storage-class@v1
  - monitoring@>=2.0
```

Semantic versioning, dependency resolution, and compatibility checks. The package
management layer the registry needs to be trustworthy at scale.

### Database-backed state (optional)

For teams who need more than in-process rolling windows — historical reconcile
data, long-term trend analysis, cross-cluster aggregation — an optional
persistence backend:

```yaml
state:
  backend: postgres   # or sqlite, mysql
  dsn: "$DATABASE_URL"
  retentionDays: 90
```

This is deliberately optional. The default is in-process and sufficient for
most operators. The database backend is for platform teams managing large operator
fleets who want historical data beyond what Prometheus provides — per-CR reconcile
history, error timelines, conversion audit trails.

The interface is the same regardless of backend. `/katalog` returns the same
response. `ork status` works the same way. The backend choice is an operational
decision, not an architectural one.

### Web dashboard

The terminal dashboard from Phase 2 extended to a web UI. Useful for
organisations managing large numbers of operators across clusters where a browser
is more appropriate than a terminal.

### CNCF Sandbox submission

Target Q1 2027. Requires production usage at multiple organisations — which Phase
2 adoption efforts should produce. CNCF Sandbox gives Orkestra vendor neutrality,
community governance, and the credibility that enterprise platform teams require
before adopting an open-source runtime.

---

## The longer horizon

The long-term vision is Katalog and Komposer as native Kubernetes kinds — registered
by the cluster itself, understood by `kube-controller-manager`, RBAC-controlled,
auditable through the standard Kubernetes audit log.

```bash
kubectl get katalogs          # not yet, but this is where we are going
kubectl describe katalog website-operator
```

The path is: production adoption → CNCF Sandbox → Kubernetes Enhancement Proposal →
alpha behind a feature gate → beta → general availability. A realistic timeline is
five years. The work is not primarily technical — the design is largely correct. The
work is community trust.

Every production deployment is evidence. Every publication is an argument made early.
Every pattern in the registry is a demonstration that the ecosystem works.

See [Orkestra: The Universal Observer That Belongs in Kubernetes Core](./publications/universal-observer-whitepaper.md)
for the full argument.

---

## What we are not building

These are deliberate non-goals, reconsidered carefully:

**Multi-cluster federation.** Orkestra manages CRDs within one cluster. Cross-cluster
operations belong to a different architectural layer. Not planned.

**Replacing controller-runtime.** Orkestra is a higher-level abstraction. For use
cases that genuinely need the full flexibility of controller-runtime, custom
constructors provide the bridge. Orkestra and controller-runtime are complementary,
not competitive.

**A general-purpose policy engine.** Orkestra's validation and mutation are scoped
to the CRDs it manages. Global cluster-wide policy across resources Orkestra does
not manage belongs in OPA, Kyverno, or VAP. Orkestra's admission model complements
these — it does not replace them.

---

## Contributing

The highest-value contributions right now:

| Area | What helps most |
|---|---|
| **Production deployments** | Run Orkestra on real workloads, report what breaks |
| **Registry patterns** | Five-file patterns for common CRDs — postgres, redis, cert-manager |
| **Testing at scale** | 50+ CRD deployments, stress test results |
| **Documentation** | Edge cases, gotchas, things that weren't obvious |
| **Hooks** | Real-world hook implementations for complex operators |

Open a [GitHub issue](https://github.com/orkspace/orkestra/issues) or
[Discussion](https://github.com/orkspace/orkestra/discussions) for anything not
covered above. See [Contributing to Orkestra](./technical-docs/CONTRIBUTING.md) for the development setup
and code standards.

---

## Release schedule

| Version | Target | Focus |
|---|---|---|
| v1.0.0 | Q2 2026 | Production-ready core — current |
| v1.1.0 | Q3 2026 | `ork dashboard`, `ork diff`, `ork lint`, `ork registry` CLI |
| v1.2.0 | Q4 2026 | Additional sources (ConfigMap, S3), performance benchmarks |
| v2.0.0 | Q1 2027 | Public registry, Katalog versioning, CNCF Sandbox submission |
| v2.x | 2027+ | Optional database backend, web dashboard, KEP preparation |

---

[Start here →](./getting-started/index.md)