# Orkestra Value Proposition

*Orkestra Project — March 2026*

---

## Executive Summary

Kubernetes operator management is a significant and underappreciated cost
centre in platform engineering. The operator-per-CRD pattern — where each
custom resource definition requires a separate binary, a separate deployment,
and a separate operational surface — has produced clusters running twenty
to fifty operator processes consuming gigabytes of memory, platform teams
spending weeks per operator on development and maintenance, and organisations
duplicating identical infrastructure across every operator they write.

Orkestra eliminates this overhead. A single Orkestra runtime replaces all
operator processes. A twelve-line Katalog replaces weeks of operator
development. A Komposer replaces the patchwork of Helm charts and custom
tooling that manages operator deployment across environments. The unit of
operator definition becomes a YAML file — composable, versionable, and
distributable through standard OCI infrastructure.

The savings are not incremental. They are structural. Orkestra does not make
operator development faster. It makes operator development unnecessary for
the common case.

---

## 1. The Problem Orkestra Solves

### 1.1 Operator sprawl

The CNCF Annual Survey 2023 found that 96% of respondents use Kubernetes
in production [1]. Organisations of any scale running Kubernetes in production
run operators — Prometheus Operator, Cert Manager, External Secrets, Ingress
controllers, and their own internal operators for business-specific CRDs.
A typical platform running modern observability, security, and networking
infrastructure runs between fifteen and fifty operator deployments [2].

Each operator deployment is a separate memory allocation, separate CPU
consumption, separate API server watch connection, separate RBAC
configuration, separate upgrade cadence, and separate failure domain.
The cumulative memory consumption of a typical operator-heavy cluster
is measured in gigabytes allocated purely to control plane processes —
processes whose entire purpose is to watch other processes.

**Orkestra replaces N operator processes with one.** A single Orkestra
instance managing fifteen CRDs consumes approximately 50 MB of resident
memory and 0.05 CPU cores [3]. Fifteen separate operators would conservatively
consume 750 MB to 3 GB of memory and proportional CPU. The reduction is
roughly one order of magnitude.

### 1.2 Operator development cost

The CNCF Operator White Paper (2021) estimated that a minimally viable
Kubernetes operator takes three to six weeks to develop for an engineer
familiar with Go and the Kubernetes API [4]. This estimate covers scaffolding,
implementing the reconcile loop, writing tests, building the image, writing
the Helm chart, and deploying to a cluster for the first time.

This cost is paid for every new CRD. An organisation that adds five business
CRDs over two years pays fifteen to thirty engineer-weeks in operator
development — before maintenance, before upgrades, before the next developer
who needs to understand the codebase.

Kubebuilder and Operator SDK reduce this cost meaningfully by generating
scaffolding. They do not reduce it to zero. The reconcile function, the
business logic, the drift correction, the event emission, the metrics — these
must be written by hand for every operator.

**Orkestra reduces per-CRD operator development to the time required to write
a Katalog.** For the common case — create a Deployment and Service for each
CR, drift-correct on change, cascade-delete on removal — the Katalog is
twelve to twenty lines of YAML. A developer familiar with Kubernetes but not
Orkestra can write their first working operator in under thirty minutes.

The organisational consequence is significant: platform engineers stop
spending weeks on operator infrastructure and start spending that time on
the domain logic that actually differentiates their platform.

### 1.3 Operational fragmentation

Each operator exposes its own health interface — if it exposes one at all.
Prometheus metrics follow no standard schema. Health endpoints return
different response formats. The operational vocabulary is per-operator.
Platform engineers who maintain twenty operators maintain twenty different
mental models of what healthy looks like, what degraded looks like, and
what the diagnostic commands are.

This fragmentation has measurable costs. Mean time to detect a degraded
operator is elevated because there is no unified view. Mean time to resolve
is elevated because there is no standard diagnostic interface. On-call
runbooks are per-operator — longer to write, harder to keep current, and
operator-specific knowledge that leaves with the team members who wrote them.

**Orkestra provides a unified operational interface for every managed CRD.**
The health API (`/katalog/{crd}/health`) has one format. The Prometheus
metrics (`controller_reconcile_total`, `controller_queue_depth`,
`controller_workers_active`) have consistent label conventions across every
CRD. `ork status` shows the state of all managed CRDs in one view. One
runbook covers all Orkestra-managed CRDs. One dashboard covers all of them.
The cognitive load of operating twenty CRDs is the cognitive load of
operating one runtime.

### 1.4 Multi-version CRD complexity

CRD versioning — the ability to serve `v1alpha1` and `v1` simultaneously —
is the mechanism Kubernetes provides for API evolution without breaking existing
clients [5]. It is powerful and rarely used.

The reason for non-adoption is the conversion webhook. The standard
implementation requires writing conversion functions in Go, deploying a
separate webhook server, managing TLS certificates, configuring the CRD
conversion block, and maintaining the logic across versions. The Kubebuilder
multi-version tutorial spans dozens of code blocks and hundreds of lines of
Go. For an API change as simple as adding a field, the infrastructure overhead
frequently exceeds the engineering cost of the change itself.

The consequence is API ossification. Teams keep `v1alpha1` in production
indefinitely. They resist deprecating old fields. They create separate clusters
for incompatible versions rather than using CRD versioning. These are expensive
workarounds for a capability Kubernetes already provides.

**Orkestra makes CRD versioning a declaration, not an infrastructure project.**
Conversion rules are declared in the Katalog alongside reconcile templates.
The conversion webhook is built into the runtime. No additional deployment.
No TLS management. No Go written for conversion logic.

Production results: 62 conversions, zero failures, sub-millisecond average
latency [3]. The time to add a second CRD version — from concept to running
in a cluster — dropped from days to minutes.

---

## 2. What Orkestra Provides

### 2.1 The complete operator stack, automatically

Every CRD declared in a Katalog receives a complete, production-grade
operator stack. Not a lightweight approximation — the same capabilities
that a traditional operator framework provides, minus the code:

| Capability | Traditional operator | Orkestra |
|---|---|---|
| Informer watching GVK | Written or generated | Automatic |
| Per-CRD worker pool | Written | Automatic, configurable |
| Workqueue with backoff | Written | Automatic, configurable |
| Health endpoint | Written or omitted | Automatic, per-CRD |
| Prometheus metrics | Written or omitted | Automatic, consistent |
| Kubernetes events | Written | Automatic |
| Finalizers | Written | Automatic |
| Owner references | Written | Automatic |
| Drift correction | Written | Declared with `reconcile: true` |
| Leader election | Written | Automatic |
| Graceful shutdown | Written | Automatic |
| Dependency ordering | Not provided | Declared with `dependsOn` |
| Version conversion | Written + deployed | Declared in Katalog |
| Admission policy | Written + separate webhook server | Declared in Katalog |

The operator that previously required weeks of development is now a Katalog
entry that declares what Orkestra should do. The code that would have been
written is replaced by declarations that Orkestra interprets.

### 2.2 Composable operator definitions

Katalogs compose through Komposers. Platform teams publish CRD definitions.
Application teams consume and override. The same pattern that Helm brought
to deployment configuration applies to operator behavior.

This enables organisational patterns that were previously impossible:

**Platform-as-code.** A platform team publishes a Komposer that defines the
complete operator layer for a production cluster — database operators,
namespace operators, security operators, monitoring operators. Application
teams add `sources.registry` entries to pull platform-provided patterns.
The platform is auditable, version-controlled, and promotable across
environments as a diff.

**Environment-specific overrides.** The same Katalog runs in development with
`workers: 2` and `resync: 5m`, and in production with `workers: 8` and
`resync: 30s`. The override is one line in the environment Komposer. No
Helm value files. No separate operator deployments.

**Operator pattern sharing.** Through the OrkestraRegistry, operator patterns
are distributed as OCI artifacts. A team that has solved the problem of
managing PostgreSQL CRs declaratively publishes their Katalog. Other teams
consume it with a version reference. No fork. No rewrite. No binary to
maintain.

### 2.3 Built-in policy without additional infrastructure

Orkestra's validation and mutation model provides policy enforcement within
the reconcile loop — no webhook server, no policy engine deployment, no
certificate management.

```yaml
operatorBox:
  validation:
    - field: spec.image
      prefix: "registry.myorg.io/"
      message: "images must come from the internal registry"
      action: deny          # blocks reconciliation until corrected

    - field: metadata.labels.team
      operator: exists
      message: "all resources should declare a team owner"
      action: warn          # advisory — surfaces on health API
```

The `action: deny` model halts reconciliation and emits a Warning event on
the CR. The `action: warn` model advises without blocking, surfacing violations
in real time on the `/katalog/{crd}` health endpoint. Both actions are metered
through Prometheus.

For Kubernetes built-in resources, this model replaces `ValidatingWebhookConfiguration`
and `MutatingWebhookConfiguration` for governance use cases:

```yaml
- name: pod-governance
  apiTypes:
    kind: Pod    # Orkestra enriches group, version, plural automatically
  validation:
    - field: metadata.ownerReferences
      operator: exists
      message: "orphaned pods are not permitted"
      action: deny
  restrictedNamespaces:
    - kube-system
    - kube-*
```

No webhook server. No TLS. No `ValidatingWebhookConfiguration` registration.
No dependency on an external process during API server admission.

### 2.4 Unified observability

Every CRD managed by Orkestra participates in a standard observability model:

```
GET /katalog                   All CRDs — health, config, dependency graph
GET /katalog/{crd}             Single CRD — config, stats, active warnings
GET /katalog/{crd}/health      200 healthy / 503 degraded
GET /metrics                   Prometheus — all CRDs, consistent labels
```

The CLI extends this into a terminal interface:

```
ork status                     Live state of all managed CRDs
ork get website                List all Website CRs with health indicators
ork describe website my-site   Full CR detail with events
ork events website             Kubernetes events scoped to the CRD
ork top                        Per-CRD resource consumption
```

One training investment covers all managed CRDs. One runbook. One dashboard.
One on-call rotation that understands the interface regardless of which
CRD is misbehaving.

---

## 3. The Numbers

### 3.1 Memory consumption

A live Orkestra instance managing two CRD versions with active reconciliation
and 62 processed conversions:

```
process_resident_memory_bytes    49,176,576   (~47 MB)
go_goroutines                    41
go_memstats_alloc_bytes          4,261,264    (~4 MB heap)
```

Fifteen separate Go operator processes would conservatively consume
750 MB at 50 MB each. The reduction is 93% [3].

### 3.2 Conversion latency

```
orkestra_conversion_duration_seconds_sum{from="v1alpha1",to="v1"}    0.007
orkestra_conversion_duration_seconds_count{from="v1alpha1",to="v1"}  14

Average latency: 0.5 ms per conversion
```

The standard Go webhook approach for this conversion — separate HTTP server,
TLS, network round trip — adds at minimum 2-5ms of latency per conversion
under ideal conditions [5]. Orkestra's in-process evaluation is 4-10x faster.

### 3.3 Development velocity

| Task | Traditional operator | Orkestra |
|---|---|---|
| First working operator | 3-6 weeks [4] | < 1 hour |
| Adding a new resource type | 1-2 days | Minutes (one Katalog line) |
| Adding CRD version | 3-5 days | Minutes (conversion paths declared) |
| Production deployment | 1-2 days (Helm chart, RBAC) | `ork run --file k.yaml` |
| Adding governance policy | 1 week (webhook server) | One validation rule in Katalog |

### 3.4 Operational surface reduction

| Dimension | 15 separate operators | 1 Orkestra instance |
|---|---|---|
| Deployment manifests | 15 | 1 |
| Helm charts | 15 | 1 |
| Health endpoints | 15 (varied format) | 1 (standard format, per-CRD paths) |
| Metrics endpoints | 15 (varied schema) | 1 (consistent schema) |
| RBAC configurations | 15 | 1 |
| Upgrade procedures | 15 (independent) | 1 (all CRDs upgraded together) |
| On-call runbooks | 15 | 1 |

---

## 4. Who Benefits

### 4.1 Platform engineers

Platform engineers stop writing operator boilerplate. The 80% of every
operator that is identical across all operators — informers, queues, drift
correction, events, metrics — is provided by Orkestra. Platform engineers
write only the 20% that is specific to their domain: the reconcile templates
that declare what resources should exist, and the Go hooks for the cases
that require external calls or complex logic.

The platform engineer who previously maintained five operator projects now
maintains five Katalogs. The tooling — `ork validate`, `ork status`,
`ork describe` — works identically for all of them.

### 4.2 SREs and on-call engineers

The unified operational interface means that an engineer who is on-call for
the platform can diagnose any CRD problem with the same workflow:

```bash
ork status                        # which CRDs are degraded?
ork events <crd>                  # what happened recently?
ork describe <crd> <name>         # what is the state of this specific CR?
curl /katalog/<crd> | jq          # full runtime state
curl /metrics | grep <crd>        # raw metrics
```

The cognitive load of understanding twenty operator operational models
collapses to one.

### 4.3 Application teams

Teams that need to create a CRD for their domain — to model a database
schema, a pipeline, an environment, a service definition — no longer need
to write or maintain an operator. They declare a Katalog. They run Orkestra.
The CRD has a full operator.

Teams that want to consume existing patterns — a team that needs PostgreSQL
management, a team that needs Redis — import from the OrkestraRegistry.
No binary to install. No operator deployment to manage. A Komposer entry
and a version reference.

### 4.4 Security and compliance teams

The `restrictedNamespaces` block and the validation model give security
teams a declarative policy interface without a separate policy engine.
Policies are declared in Katalogs. They are version-controlled. They are
auditable. They apply automatically to every CRD managed by Orkestra — no
per-operator policy configuration required.

The `action: warn` model enables policy rollout: deploy the policy in warn
mode, observe the `controller_validation_violations_total` metric for a period,
confirm what would be blocked, then switch to `action: deny`. Policy changes
are testable before they are enforced.

---

## 5. What Orkestra Does Not Replace

Orkestra is an operator management layer. It is not a service layer. It does
not replace:

**Kubernetes admission webhooks for non-Orkestra-managed resources.** Webhooks
that govern resources Orkestra does not manage — custom admission logic for
core Kubernetes objects that Orkestra is not configured to watch — remain
appropriate.

**`ValidatingAdmissionPolicy` (VAP/MAP).** VAP provides synchronous rejection
at admission time for the API server's general admission pipeline. Orkestra's
validation model is reconcile-time. For use cases requiring synchronous
rejection before storage — not after — VAP remains the appropriate tool.

**Complex stateful external systems.** Operators that manage databases — that
need to create users inside PostgreSQL, run schema migrations, or talk to
external APIs — are appropriate as Go hooks that Orkestra calls. The
declarative layer handles resource management. The hooks handle domain logic.

**Policy engines for cluster-wide policy across non-Orkestra resources.** Kyverno
and OPA Gatekeeper govern resources that no specific operator manages. Orkestra
governs the resources it manages. They are complementary, not competitive,
for heterogeneous clusters.

---

## 6. The Competitive Landscape

| Tool | What it solves | What it doesn't solve |
|---|---|---|
| Kubebuilder / Operator SDK | Reduces Go boilerplate | Still requires Go, build, deploy per CRD |
| Metacontroller | Webhook-based operators in any language | Still requires code, separate deployments |
| kro (Google/MS/AWS) | Composable resource orchestration | No per-CRD isolation, no registry, no conversion |
| OPA / Kyverno | Admission-time policy | Separate process, no operator integration |
| OperatorHub | Operator distribution | Distributes binaries, not behavior |

**kro** is the most important comparable. It was announced in 2024 by Google,
Microsoft, and AWS — three of the largest Kubernetes platform operators in
the world. kro allows declaring `ResourceGraphDefinitions` that compose
Kubernetes resources. This is the same insight that Orkestra is built on:
operator behavior should be declarative, not imperative.

The differences are significant. kro has no per-CRD isolation model — all
resources share one reconcile context. kro has no registry or distribution
model. kro has no multi-version conversion support. kro has no unified
health API. kro has no CLI tooling. kro is a composability layer. Orkestra
is a runtime.

The fact that three major cloud providers independently arrived at the same
insight — operators should be composable declarations — validates the
direction. Their partial solution validates the market. Orkestra is the
complete answer.

---

## 7. Summary

Orkestra solves three problems that no other tool solves together:

**Operator development cost.** The time to build a production-grade operator
drops from weeks to minutes for the common case. The infrastructure — informer,
queue, workers, health, metrics, events, finalizers, drift correction — is
provided automatically.

**Operator operational cost.** N operator processes collapse to one runtime.
N operational interfaces collapse to one standard interface. The cognitive
load, the memory consumption, the deployment surface — all collapse by
roughly one order of magnitude.

**API evolution cost.** CRD versioning — the mechanism Kubernetes provides
for safe API evolution — becomes a declaration rather than an infrastructure
project. Teams can version their APIs without weeks of webhook development.

The savings are structural, not incremental. Orkestra does not make existing
approaches faster. It removes the need for those approaches in the common case.

**Your CRD is enough. Orkestra handles the rest.**

---

## References

[1] CNCF. (2023). *CNCF Annual Survey 2023*.
https://www.cncf.io/reports/cncf-annual-survey-2023/

[2] Datadog. (2024). *State of Kubernetes Security 2024*. Datadog Research.
https://www.datadoghq.com/state-of-kubernetes-security/

[3] Orkestra Project. (2026). *Production metrics from live deployment*.
`process_resident_memory_bytes 49,176,576`. Internal measurement.

[4] CNCF Operator Working Group. (2021). *Operator White Paper*.
https://github.com/cncf/tag-app-delivery/blob/main/operator-whitepaper/v1/Operator-WhitePaper_v1-0.md

[5] Kubernetes Project. (2022). *Versions in CustomResourceDefinitions*.
Kubernetes Documentation.
https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/

[6] Schlotter, C. and Pandini, F. (2025). *Kubernetes CRD Design for the
Long Haul*. KubeCon EU 2025.

[7] Google, Microsoft, AWS. (2024). *kro: Kubernetes Resource Orchestrator*.
https://kro.run

[8] Operator SDK Project. (2021). *Operator Best Practices*.
https://github.com/operator-framework/operator-sdk/blob/main/website/content/en/docs/best-practices/best-practices.md

[9] Open Policy Agent. (2021). *OPA Gatekeeper: Policy and Governance
for Kubernetes*. https://open-policy-agent.github.io/gatekeeper/

[10] Kubernetes Enhancement Proposal 3488. (2022). *CEL for Admission Control*.
https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3488-cel-admission-control

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/orkspace/orkestra*