---
title: "Universal Observer Whitepaper"
weight: 75
---

# Orkestra: The Universal Observer That Belongs in Kubernetes Core

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes extensibility is built on a structural absence. The API server
stores and serves custom resources. It knows their schema. It emits watch
events when they change. What it does not do — and has never done — is
watch what happens after those events are emitted. The operator pattern
emerged to fill that gap: a controller that watches a resource and
reconciles the cluster toward the state declared in it. But every operator
is separate. Every operator is isolated. There is no shared layer that
sees across the managed surface — no permanent observer of the extension
ecosystem as a whole.

This absence has compelled the invention of adjacent systems that compensate
for it. Admission webhooks exist because the API server cannot ask its own
operators whether a CR is valid — it must delegate externally. Policy engines
exist because operators have no shared policy model — policy must live in a
separate process. Observability tooling exists because every operator invents
its own metrics and health conventions — there is no standard interface.

We argue that all of these systems are compensations for the same architectural
gap. We describe Orkestra — a runtime that acts as the permanent observer of
its managed surface — and show that when such an observer exists, the
external compensations become unnecessary. We then make the case that this
observer belongs inside Kubernetes itself: that Katalog and Komposer should
be native Kubernetes resource kinds, that the Orkestra runtime should ship
as a core controller, and that the API server's admission model should be
extended to call this trusted in-process observer directly. We examine the
technical path to this integration, the community and governance steps
required, and the implications for how the Kubernetes ecosystem manages
extensibility.

---

## 1. The Structural Absence

### 1.1 What Kubernetes provides

Kubernetes makes a specific set of guarantees about custom resources. When a
CRD is installed, Kubernetes will store objects of that type, validate them
against the declared schema, emit watch events when they change, and serve
them over the REST API in any version the webhook infrastructure supports.

These guarantees are complete and correct within their scope. The API server
is a general-purpose storage and serving system. It is excellent at what it
does.

What it does not do is observe the downstream consequences of those changes.
The API server does not know that a `Website` CR at `spec.replicas: 5` should
produce a Deployment with five replicas. It does not know that setting
`spec.image` to an image from an untrusted registry should be rejected. It
does not know that when a `Database` CR is deleted, a cleanup Job must run to
completion before the CR itself is removed. All of that knowledge lives
elsewhere — in the operator — and the API server has no visibility into it.

### 1.2 The operator fills the gap — partially

The operator pattern was introduced in 2016 to fill this gap [1]. A controller
watches a custom resource and continuously reconciles the cluster toward the
desired state declared in it. This is the right abstraction. It encodes
domain knowledge as a continuous control loop, which is precisely what
Kubernetes's level-triggered architecture was designed to support.

The operator fills the gap for the resources it watches. It does not fill
it for anything else. And critically, it fills it in a way that is
completely invisible to the rest of the system. The API server does not know
which operators are running, which CRDs they watch, or what state they have
produced. There is no registry of active operators. There is no shared
visibility layer. Operators are point solutions to a platform problem.

### 1.3 The compensations

The absence of a shared observer has produced a generation of compensating
tools. Each compensates correctly for the specific gap it addresses. Together,
they create a picture of what the observer should have been.

**Admission webhooks** [2] compensate for the API server's inability to ask
operators whether a CR is valid. The API server cannot consult the operator
synchronously during admission — the operator is a separate process with no
defined interface. So the webhook pattern provides a defined interface: an
HTTP endpoint the API server can call. The endpoint receives the object before
storage and returns accept or reject.

The infrastructure required to make this work — TLS certificates, separate
deployments, webhook registration, availability guarantees — all exists to
make the external call safe. The infrastructure is not the solution. It is
the overhead of the coordination pattern.

**Policy engines** — OPA, Kyverno, Gatekeeper — compensate for the fact that
operators have no shared policy model [3]. Each operator is a point solution.
Its validation logic is its own code. When an organisation needs uniform
policy across many CRDs — "all images must come from the internal registry,"
"all pods must have owner references" — it cannot enforce this through
operators because the operators are separate and have no coordination
mechanism. So a separate policy engine sits at the admission layer and
enforces policy uniformly.

**`ValidatingAdmissionPolicy`** [4], introduced in Kubernetes 1.26, compensates
for the same gap more efficiently: CEL expressions evaluated in-process by
the API server, with no external HTTP call. This is a meaningful step toward
what we are describing. But it is still a separate mechanism — declarative
policy is not connected to the operator that manages the resource, and the
policy engine has no runtime visibility into what the operator has produced.

**Observability tooling** compensates for the fact that every operator invents
its own metrics, health conventions, and operational interface. The Prometheus
Operator exposes metrics on one endpoint with one set of label conventions.
The Cert Manager Operator exposes metrics on a different endpoint with
different conventions. The External Secrets Operator exposes metrics on yet
another. Platform engineers aggregate these into dashboards that require
per-operator knowledge of the label schema, the endpoint path, and the metric
names. Observability tooling — Kube State Metrics, custom dashboards, custom
alerts — compensates for the absence of a standard interface.

### 1.4 The pattern in the compensations

Each compensation is correct within its scope. Admission webhooks work.
Policy engines work. `ValidatingAdmissionPolicy` works. Observability tooling
works. What none of them are is architecturally unified. They are four
separate systems, each compensating for a different consequence of the same
structural absence.

The absence is: **no permanent observer watching the managed surface.**

If such an observer existed, the coordination problem that webhooks solve
would not require external delegation — the observer could answer "is this
valid?" in-process. The policy problem that policy engines solve would not
require a separate system — the observer's policy model would apply uniformly
to every CRD it manages. The observability problem would not require
per-operator tooling — the observer would provide a standard interface for
every CRD it manages by construction.

Orkestra is that observer.

---

## 2. The Observer Model

### 2.1 What an observer provides

A permanent observer watching the managed surface provides things that no
collection of isolated point solutions can provide.

**Cross-surface visibility.** An observer that watches ten CRDs sees all ten
simultaneously. It can enforce policy uniformly. It can answer questions about
the aggregate state — how many CRDs are healthy, which have queued work,
which have elevated error rates. It can order their lifecycle — starting CRDs
in dependency order, shutting them down in reverse. None of these are possible
when each CRD has its own isolated process.

**In-process authority.** An observer that is trusted by the system can be
called synchronously during admission. There is no network hop. There is no
TLS. There is no availability concern separate from the observer's own
availability. The API server can ask "is this object valid?" and receive an
answer from the observer directly, with the same latency as any other
in-process function call.

**Standard interface by construction.** An observer that manages every CRD
through the same runtime provides the same operational interface for every
CRD. The health endpoint has the same format. The metrics have the same label
conventions. The CLI commands work identically. This is not a convention that
must be negotiated across operator teams. It is a consequence of the shared
runtime.

**Declarative policy composition.** An observer that interprets declarations
rather than executing code can apply policy at the declaration level. Validation
rules, mutation rules, namespace restrictions — these are properties of the
Katalog declaration, not properties of the operator code. They compose through
the same Komposer model as CRD definitions. They are overridable, versionable,
and shareable.

### 2.2 What Orkestra currently provides

Orkestra is a production implementation of the observer model. As of March 2026,
it runs in production with the following demonstrated properties.

Each CRD managed by Orkestra receives a dedicated operator stack: its own
informer watching a specific GVK, its own workqueue with independent depth
and backoff, its own worker pool whose goroutines cannot be consumed by other
CRDs, its own reconciler interpreting only that CRD's templates, and its own
health endpoint and metrics.

The runtime provides these stacks for any number of CRDs simultaneously. The
resource consumption of the shared infrastructure — the API server connections,
the informer factory, the health server, the leader election lease — is paid
once. The per-CRD isolation that the one-operator-per-CRD principle mandates
is preserved and strengthened. The operational overhead of running many CRDs
is collapsed to the cost of running one runtime.

Multi-version CRD conversion — the most complex standard operator capability
— is demonstrated in production with 62 conversions, zero failures, and
sub-millisecond average latency, with zero conversion code written [5].
This is possible because each CRD version is a first-class operator entry
with its own operator stack. Conversion rules are declared alongside reconcile
templates and evaluated by the same resolver.

The `sources.registry` model distributes operator definitions as OCI artifacts.
Consumers import patterns with a URL reference. The five-file validation
enforces structure at pull time. The ecosystem is nascent but the distribution
infrastructure is proven: OCI is how the Kubernetes community distributes
everything else.

### 2.3 The gap between Orkestra today and Orkestra in core

What Orkestra provides today is a user-space implementation of the observer
model — a process that users deploy and operate. This is correct and valuable
as a starting point. But it is not the same as the observer being trusted
infrastructure.

The key difference is the admission model. Orkestra's validation runs in the
reconcile loop — after the CR is stored. A CR that violates a validation rule
is stored, then rejected on the next reconcile, then the user is notified
through a Kubernetes Warning event. The feedback loop is sub-second for a
healthy cluster. But it is not synchronous rejection at admission time.

Synchronous rejection requires trust — the API server must be willing to
call the observer during the admission request and block until the answer
arrives. The API server will not do this for an arbitrary user-space process.
It will do it for a trusted in-process component. The path from user-space
Orkestra to in-core Orkestra is the path from "an observer" to "the trusted
observer."

---

## 3. The Case for Core Integration

### 3.1 Precedent: kube-controller-manager

Kubernetes has always shipped with a runtime that watches many resource types
through a shared process. `kube-controller-manager` runs the Deployment
controller, the ReplicaSet controller, the StatefulSet controller, the Job
controller, and dozens of others [6].

Each of these controllers is isolated — it watches one set of resources,
reconciles according to its own logic, and has no coupling to the others.
What they share is the infrastructure: the API server connection, the informer
factory, the leader election lease, the process boundary.

This is exactly the Orkestra model applied to core Kubernetes resources.
The analogy is exact: `kube-controller-manager` is to core resources what
Orkestra is to custom resources. Both run multiple isolated controllers in a
shared runtime. The only difference is that `kube-controller-manager` is
trusted infrastructure and Orkestra is not yet.

If it is correct for `kube-controller-manager` to run multiple controllers
in one process, it is correct for a custom resource meta-controller to do
the same. The principle does not change at the boundary between core and
custom resources.

### 3.2 The ValidatingAdmissionPolicy signal

The introduction of `ValidatingAdmissionPolicy` in Kubernetes 1.26 [4] is
significant not for what it provides but for what it reveals about the
direction. The Kubernetes project decided to move policy evaluation in-process
— to eliminate the external HTTP call for the common case. This required
adding a new resource type (`ValidatingAdmissionPolicy`), a new controller
to evaluate it, and a new execution engine (CEL) to run the policy expressions.

This is the API server taking the first step toward trusting an observer.
The CEL evaluator is, in a narrow sense, an in-process observer. It watches
admission events, evaluates policy, and returns accept or reject — all without
external delegation.

The step is narrow because the CEL evaluator is static policy, not a
reconciling observer. It can check that `spec.image` matches a prefix. It
cannot check that the Deployment created by the operator has the correct
image in its container spec. It cannot observe the downstream effects of
admission decisions. It cannot enforce policy across the operator's managed
surface because it has no visibility into that surface.

Orkestra's observer model is the extension of this step. Not CEL expressions
evaluated at admission time, but a full reconciling observer called at admission
time — one that knows what the operator has produced, what the dependencies
are, and what the full state of the managed surface looks like.

### 3.3 The built-in enrichment precedent

The built-in kind enrichment already demonstrates the principle at a small
scale. When a Katalog declares `apiTypes.kind: Deployment` with no other
fields, Orkestra queries the cluster's discovery API and enriches the entry
with the correct group, version, plural, and scope. The cluster knows this
information. Orkestra asks for it.

When Orkestra is a core component, this enrichment becomes instantaneous and
authoritative. The API server already holds the complete schema for every
resource it serves. A native meta-controller can access this directly —
not through the discovery API, but through the same internal schema registry
that the API server itself uses. The meta-controller becomes the first
resource manager that can inspect the full schema of every CRD it manages
without a round trip.

### 3.4 The unified admission interface

The full vision of in-core Orkestra is a unified admission interface:

```
kubectl apply → API server → Orkestra validation engine (in-process) → stored
```

The admission path for a CR managed by Orkestra calls the validation engine
before storage. The validation engine evaluates the Katalog's declared rules —
not CEL, but the full Orkestra condition model with its nine operators and
template expression support. The answer is authoritative because the validation
engine is the observer — it has the full reconcile context, not just the
object being admitted.

For built-in resources — Deployments, Pods, Namespaces — the Kind alone is
enough. The meta-controller already knows the schema. For custom resources,
the CRD definition provides the schema at install time. Either way, the
Katalog is the policy declaration and the meta-controller is the enforcer.

This collapses the admission model. `ValidatingWebhookConfiguration` is
replaced by `reconciler.validation` in the Katalog. `MutatingWebhookConfiguration`
is replaced by `reconciler.mutation`. Policy engines are replaced by
Komposer-level validation rules that compose across all managed CRDs.
Certificate management for webhooks is replaced by nothing — there is no
external endpoint.

### 3.5 Katalog and Komposer as native kinds

The final integration makes Katalog and Komposer native Kubernetes resource
kinds, registered by the cluster itself:

```yaml
apiVersion: core.kubernetes.io/v1
kind: Katalog
metadata:
  name: website-operator
spec:
  crds:
    website:
      apiTypes:
        kind: Website
```

`kubectl get katalogs` works without installing Orkestra. `kubectl describe
katalog website-operator` shows the full CRD health, reconcile statistics,
and active warnings. The Katalog is a Kubernetes object — versioned,
RBAC-controlled, auditable through the standard audit log.

Every cluster that ships Kubernetes ships with a meta-controller that
understands Katalogs and Komposers. Platform teams write Katalogs. Kubernetes
manages them. Installation of operator infrastructure is eliminated. The
question "which operator framework should we use?" is answered by the cluster.

---

## 4. The Path

The path from user-space Orkestra to in-core Orkestra is multi-year and
community-governed. It is not a single PR. It is a staged progression with
clear milestones.

### 4.1 Production adoption (Year 1)

The prerequisite for any community proposal is evidence that the model works
at scale. Three to five organisations running Orkestra in production —
managing real CRDs, handling real load, relying on the runtime for their
platform — is the baseline.

Each production deployment generates evidence: reconcile throughput, failure
rate, conversion latency, memory consumption, startup time under load. This
evidence is the foundation of any credible KEP. A proposal without production
data is a thought experiment. A proposal with production data from multiple
organisations is an engineering proposal.

### 4.2 CNCF Sandbox (Year 2)

CNCF Sandbox establishes vendor-neutrality and community governance. The
submission requires production usage at multiple organisations, a governance
model, a code of conduct, and a maintainer group from multiple employers.

CNCF Sandbox is the signal to the Kubernetes community that Orkestra is a
project, not a product. Projects that make it through CNCF have been reviewed
for viability and vendor neutrality. They are taken seriously in KEP discussions.

### 4.3 Kubernetes Enhancement Proposal (Year 3)

A KEP for a native operator runtime. The KEP proposes the model — Katalog,
Komposer, declarative reconciliation, in-process validation — not the
codebase verbatim. The Orkestra codebase is the reference implementation
that proves the model works.

The KEP enters the SIG Apps and SIG API Machinery review process. This is
where the Kubernetes community stress-tests the design. The super-operator
model, the per-CRD isolation, the conversion webhook integration, the admission
model — all will be challenged. The strong designs survive. Some things will
change. The core insight — a permanent observer belongs in the runtime —
will not.

### 4.4 Alpha behind a feature gate (Year 4)

`OrkestraRuntime=true` ships as an alpha feature. Katalog and Komposer kinds
are registered. The runtime runs as part of `kube-controller-manager`. Existing
Orkestra users can migrate their Katalogs with zero changes — the API is
identical.

The graduation criteria are standard: alpha → beta → GA over multiple releases,
with each stage requiring feature completeness, test coverage, documentation,
and evidence of production usage at scale.

### 4.5 General availability (Year 5)

`OrkestraRuntime=true` becomes default. Every Kubernetes cluster ships with
a meta-controller. Platform teams write Katalogs. Kubernetes manages them.

The admission model is extended: for CRDs managed by a Katalog, validation
and mutation run in-process. The webhook infrastructure for Orkestra-managed
CRDs is deprecated.

---

## 5. Implications

### 5.1 Fewer resource types, more capability

One of the recurring challenges in the Kubernetes extensibility model is
CRD proliferation. Each tool that extends Kubernetes installs its own CRDs.
OperatorHub operators install hundreds of CRDs across a typical cluster.
Many of these CRDs serve the same purpose: managing the lifecycle of another
system. A native meta-controller eliminates the need for each of these to
install its own management CRD. The Katalog is the one management CRD.

This is the "less tooling, fewer CRD types per resolution" outcome described
in the Orkestra roadmap: a single native interface for both admission and CRD
management — unified, composable, observable by default.

### 5.2 The operator ecosystem restructures

The ecosystem currently organized around operator frameworks —
Kubebuilder, Operator SDK — would restructure around Katalog authoring and
OrkestraRegistry. The dominant question moves from "how do I write the
reconciler?" to "what does this CRD need to do declaratively, and what
requires hooks?"

This is not a loss for the ecosystem. It is a maturation. The frameworks
still serve use cases that require Go hooks — complex business logic, external
API calls, stateful workflows. The frameworks become the way to write typed
hooks, not the way to write entire operators.

### 5.3 The webhook ecosystem is deprecated — gradually

Admission webhooks are not deprecated overnight. There are thousands of
deployed webhooks. Many manage resources not controlled by Orkestra. The
deprecation path is targeted: for CRDs managed by a Katalog, the Katalog's
validation and mutation rules replace the webhook. For everything else,
webhooks continue to function.

The CRD conversion webhook is deprecated more completely. Declarative
conversion in Orkestra replaces it for the common case — field mapping,
defaulting, dropping — which covers the overwhelming majority of version
conversions. Complex transformations that require external state remain as
hooks. The webhook infrastructure for conversion is no longer the default.

### 5.4 RBAC for operator behavior

A notable implication of Katalog being a native kind: operator behavior is
now RBAC-controlled. A team can be granted write access to `Katalog` resources
in their namespace but not in `kube-system`. A platform team can create a
`Katalog` that manages all namespaces; application teams can create `Katalog`
resources that manage only their own namespace. Operator behavior becomes
part of the standard Kubernetes access model.

This is a significant governance improvement over the current state, where
installing an operator binary effectively grants it cluster-wide access by
default.

---

## 6. What This Requires

The integration described above requires work across three dimensions.

**Runtime work.** The per-CRD operator stack model is proven. The conversion
webhook is proven. The health API and metrics are proven. The work required
for core integration is adaptation: removing external dependencies, fitting
into the `kube-controller-manager` lifecycle, adopting Kubernetes-standard
client patterns, passing the Kubernetes conformance test suite.

**API work.** The Katalog and Komposer kinds need formal API review. Field
names, version strategies, conversion between Katalog versions (Orkestra
would manage its own meta-CRDs), and the admission integration for the
in-process validation model all require API machinery review and approval.

**Community work.** The most important dimension. The Kubernetes community
does not accept proposals from projects. It accepts proposals from people —
contributors who have earned trust through engagement with SIGs, through
production evidence, through code review, through KEPs that survive the
review process. This is the slowest and most important part of the path.

No amount of technical correctness substitutes for community trust. Orkestra's
publications, production data, and reference implementation establish technical
credibility. The community trust is built through participation, through
openness, through the willingness to have the design challenged and improved.

---

## 7. Conclusion

Kubernetes extensibility has, since its inception, relied on an external
delegation model for the capabilities that require awareness of the managed
surface. Admission webhooks delegate to external processes. Policy engines
run separately. Observability tooling fragments across operator implementations.
All of this infrastructure exists to compensate for the absence of a permanent,
trusted observer.

Orkestra demonstrates that this observer can exist, that it can be built on
the existing Kubernetes primitives, and that when it exists, the external
compensations become unnecessary. The admission model, the policy model, and
the observability model all simplify when the observer is trusted.

The argument for core integration is the same argument that justified
`kube-controller-manager` running multiple controllers in one process: the
isolation that matters is at the reconciler level, not at the process level.
When the runtime provides that isolation structurally — through dedicated
informers, queues, worker pools, and failure domains — the operational
overhead of running many CRDs collapses without sacrificing the per-CRD
independence that the operator pattern requires.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The observer that makes this possible belongs in the runtime.
The runtime belongs in the cluster.

---

## References

[1] Coreos. (2016). *Introducing Operators: Putting Operational Knowledge
into Software*. CoreOS Blog. https://coreos.com/blog/introducing-operators.html

[2] Kubernetes Project. (2019). *Admission Webhooks*. Kubernetes Documentation.
https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/

[3] Open Policy Agent Project. (2018). *Open Policy Agent: Policy-based
Control for Cloud Native Environments*. CNCF.
https://www.openpolicyagent.org/docs/latest/kubernetes-introduction/

[4] Kubernetes Enhancement Proposal 3488. (2022). *CEL for Admission Control*.
Kubernetes SIG API Machinery.
https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3488-cel-admission-control

[5] Orkestra Project. (2026). *Declarative Version Conversion for Kubernetes
CRDs: Production Results*. Internal metrics, Orkestra deployment.
`orkestra_conversion_requests_total{result="success"} 62 / failures 0`.

[6] Kubernetes Project. (2022). *Kubernetes Controller Manager*.
Kubernetes Documentation.
https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/

[7] Operator SDK Project. (2021). *Operator Best Practices*.
GitHub, operator-framework/operator-sdk.
https://github.com/operator-framework/operator-sdk/blob/main/website/content/en/docs/best-practices/best-practices.md

[8] CNCF. (2023). *CNCF Annual Survey 2023*.
https://www.cncf.io/reports/cncf-annual-survey-2023/

[9] Kubernetes Enhancement Proposal 2258. (2021). *Node Swap Support*.
(Referenced as an example of the KEP process timeline from proposal to GA.)

[10] Schlotter, C. and Pandini, F. (2025). *Kubernetes CRD Design for the
Long Haul*. KubeCon EU 2025.
https://kccnceu2025.sched.com/event/1txHi

[11] Kambalimath, G. (2026). *One Operator To Rule Them All? CRD Management
Strategies for Cloud Native Apps*. KubeCon + CloudNativeCon.
https://www.classcentral.com/course/youtube-one-operator-to-rule-them-all-crd-management-strategies-for-cloud-native-apps

[12] Google, Microsoft, AWS. (2024). *kro: Kubernetes Resource Orchestrator*.
https://kro.run

[13] Kubernetes Project. (2020). *Custom Resource Definitions*. Kubernetes
Documentation.
https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026 — https://github.com/orkspace/orkestra*