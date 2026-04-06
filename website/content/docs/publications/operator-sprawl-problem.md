---
title: "The Operator Sprawl Problem"
weight: 50
description: "*Orkestra Project — March 2026*"
---

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operator adoption has grown faster than the tooling to manage it.
The CNCF Annual Survey 2023 found that 96% of organisations use Kubernetes in
production [1]. Of those, the majority run between ten and fifty operator
deployments — each a separate binary, a separate process, a separate operational
surface. The cumulative cost of this infrastructure has become significant,
underappreciated, and, until recently, unavoidable.

This paper quantifies the operator sprawl problem through a concrete case study
and examines the structural reasons it has been allowed to grow. It then shows
why Orkestra's shared runtime model resolves the problem at the architectural
level — not by reducing the cost per operator, but by eliminating the per-operator
cost structure entirely.

---

## 1. The growth of the operator ecosystem

The operator pattern was introduced in 2016 [2]. By 2021, the CNCF Operator
White Paper estimated over two hundred production-quality operators publicly
available [3]. By 2024, OperatorHub.io listed over three hundred. Most
organisations running Kubernetes in production run a combination of community
operators and internally developed operators.

The community operators cover infrastructure concerns: Prometheus Operator,
Cert Manager, External Secrets Operator, Ingress NGINX, Strimzi, CloudNativePG.
The internal operators cover business concerns: application lifecycle, namespace
provisioning, compliance enforcement, internal service registries.

A typical platform team at an organisation with 50–200 engineers running three
clusters (development, staging, production) runs between fifteen and forty
operators. The number grows over time — new infrastructure dependencies, new
compliance requirements, new platform capabilities.

---

## 2. The per-operator cost structure

Each operator imposes a repeating cost in four dimensions.

### 2.1 Memory and CPU

Each operator is a Go binary running as a Kubernetes Deployment. A minimal
operator — informer, workqueue, reconcile loop, health endpoint — consumes
approximately 50–80 MB of resident memory at steady state. An operator with a
larger reconcile scope, more CRD versions, or higher throughput requirements
consumes 100–200 MB.

For a cluster running twenty operators, this is 1–4 GB of memory allocated
purely to control plane processes. These processes produce nothing of business
value directly — they exist to watch other processes. At cloud provider rates
for managed Kubernetes, this represents $50–200/month in compute costs per
cluster, multiplied across every cluster in the organisation.

### 2.2 API server load

Each operator maintains one or more informer watch connections to the API server.
Each informer receives a copy of every watch event for its CRD. Twenty operators
with informers watching twenty different resource types means twenty parallel
streams of watch events flowing from the API server.

The API server handles this load, but it is redundant. If a single process
watched all twenty resource types, it would receive the same events and process
them with the same logic — but through one connection instead of twenty.

### 2.3 Development time

The CNCF Operator White Paper estimated three to six weeks for a minimally
viable operator using Kubebuilder or Operator SDK [3]. This covers scaffolding,
implementing the reconcile loop, writing tests, building the container image,
and writing the Helm chart for deployment.

For an organisation that adds ten operators over three years — a conservative
estimate for a growing platform — this represents thirty to sixty engineer-weeks
of investment. That is roughly one engineer-year, before any maintenance cost
is counted.

### 2.4 Operational burden

The operational burden of twenty operators is not twenty times the burden of
one. It is worse, because each operator has a different operational interface.

Different health endpoint paths and response formats. Different Prometheus
metric names and label conventions. Different log formats and verbosity levels.
Different upgrade procedures. Different failure modes. Different on-call
runbooks — twenty documents, each requiring specific knowledge of one operator's
internals.

Platform engineers responsible for operator health cannot build a single mental
model. They build twenty partial models and work from memory and documentation.
Mean time to detect an operator failure is elevated because there is no unified
view. Mean time to resolve is elevated because each operator requires specific
knowledge.

---

## 3. A concrete case study

Consider a platform team supporting a product organisation with 80 engineers.
The cluster runs the following operators:

**Community operators (12):** Prometheus Operator, Grafana Operator,
Cert Manager, External Secrets Operator, Ingress NGINX, Strimzi (Kafka),
CloudNativePG, Argo CD, External DNS, Sealed Secrets, Keda, Reloader.

**Internal operators (8):** Namespace provisioner, Application lifecycle,
Database schema manager, Service account provisioner, Cost attribution tagger,
Compliance enforcer, Internal registry sync, Team workspace provisioner.

**Memory consumption (estimated):**
- Community operators: 12 × 100 MB average = 1.2 GB
- Internal operators: 8 × 75 MB average = 600 MB
- Total: ~1.8 GB per cluster × 3 clusters = ~5.4 GB

**Development investment (internal operators only):**
- 8 operators × 4 weeks average = 32 engineer-weeks = 8 engineer-months

**Ongoing maintenance (estimated):**
- Version upgrades: 20 operators × 2 hours/year = 40 hours/year
- Incident response: 20 operators × 1 incident/year × 2 hours = 40 hours/year
- Documentation: 20 operators × 2 hours/year = 40 hours/year
- Total: ~120 engineer-hours/year = 3 engineer-weeks/year

The 8-month investment to build the internal operators would, at the time, have
been considered reasonable for the value delivered. The ongoing maintenance cost
of 3 engineer-weeks/year is paid indefinitely. After three years, the total
investment exceeds a year of engineering time — for operators that produce no
direct business value, only platform capability.

---

## 4. Why the cost structure exists

The operator cost structure exists because the operator pattern was defined as
one-binary-per-CRD. This was a reasonable initial design — it gave operators
clear failure boundaries and independent deployment lifecycles. The cost was
acceptable when operators were rare. It has become structural overhead as
operators proliferated.

The underlying cause is that the operator frameworks accepted an inherited
premise from the Kubernetes controller design: controllers are programs. A
controller for a Deployment is a Go function registered in `kube-controller-manager`.
A controller for a custom resource is a Go program running as a separate process.

The distinction between these two models is not architectural — it is historical.
`kube-controller-manager` runs the Deployment controller, the ReplicaSet
controller, the Job controller, and dozens of others in a single process. The
per-CRD isolation is at the reconcile function level, not at the process level.
Operator frameworks did not adopt this model. They adopted the separate-process
model, and the ecosystem built on top of it.

The Kubernetes Operator White Paper (2021) noted this as an area for future
improvement but provided no mechanism for change [3]. The kro project (2024),
jointly developed by Google, Microsoft, and Amazon Web Services, represents
the first major signal from cloud providers that the operator model needs to
evolve [4].

---

## 5. The Orkestra resolution

Orkestra's shared runtime model addresses the operator sprawl problem at its
root by replacing the per-binary cost structure with a per-declaration cost
structure.

**Memory:** A live Orkestra instance managing two CRDs and processing active
reconciles consumes approximately 47 MB of resident memory. The same number
of CRDs managed by separate operators would consume 100–400 MB. For twenty
CRDs, the ratio is approximately 20:1. The savings compound with cluster count.

**API server load:** One Orkestra instance opens one informer factory. Watches
for twenty CRDs flow through one connection multiplexer. The API server handles
one client rather than twenty.

**Development time:** The time to add a CRD to an Orkestra Katalog is under
one hour for the common case — declaring reconcile templates, dependencies,
and validation rules. The 3–6 week estimate for a traditional operator reflects
the time to write and test Go code. Orkestra eliminates that work for operators
that do not require external API calls.

**Operational burden:** One Orkestra instance provides one health endpoint
schema, one Prometheus metric schema, one CLI interface, one Helm chart to
maintain, one upgrade to perform per release. The cognitive load of operating
twenty CRDs is the cognitive load of operating one runtime.

**Applying the case study numbers:**

| | 20 separate operators | Orkestra |
|---|---|---|
| Memory (per cluster) | ~1.8 GB | ~50 MB |
| Memory (3 clusters) | ~5.4 GB | ~150 MB |
| Internal operator dev | 8 months | 2 weeks |
| Annual maintenance | 3 weeks | 0.5 weeks |
| Operational surfaces | 20 | 1 |

The memory reduction alone — from 5.4 GB to 150 MB across three clusters —
represents a meaningful reduction in cloud compute costs. At current cloud
provider rates for general-purpose memory, this translates to approximately
$150–400/month in savings, depending on region and provider.

The development time reduction is the more significant number. Eight months
versus two weeks for the internal operator suite represents six months of
engineering time — time that can be invested in the platform capabilities
that operators provide rather than in the operators themselves.

---

## 6. The adoption calculus

An organisation evaluating Orkestra faces a specific question: at what scale
does the shared runtime model justify the migration cost?

The answer depends on the number of internal operators and the growth rate.
For an organisation currently running fewer than five internal operators with
no plans to add more, the migration cost may exceed the benefit. For an
organisation running eight or more internal operators, or planning to add
more, Orkestra's cost structure is strictly superior.

The community operator question is different. Orkestra can manage community
CRDs — any CRD that Kubernetes accepts, Orkestra can reconcile — but the
migration path from existing community operators to Orkestra-managed patterns
requires the OrkestraRegistry to have production-tested patterns for those
CRDs. That ecosystem is growing.

The internal operator question is straightforward. Internal operators are
built from scratch and maintained indefinitely. Writing them as Katalogs
rather than Go programs is the correct default for any operator whose logic
can be expressed declaratively.

---

## 7. Conclusion

Operator sprawl is a predictable consequence of the per-binary operator model
applied at scale. The cost is real, measurable, and compounding. The CNCF's
own data shows that operator adoption continues to grow — which means the
sprawl cost continues to grow with it.

Orkestra's shared runtime model resolves the sprawl problem by changing the
unit of operator cost from per-binary to per-declaration. The cost of adding
a CRD to an Orkestra deployment is the cost of writing a Katalog entry — an
hour of work rather than weeks. The ongoing cost of operating twenty CRDs is
the ongoing cost of operating one runtime.

The economics are not incremental. They are structural.

---

## References

[1] CNCF. (2023). *CNCF Annual Survey 2023*.
https://www.cncf.io/reports/cncf-annual-survey-2023/

[2] CoreOS. (2016). *Introducing Operators: Putting Operational Knowledge into Software*.
https://coreos.com/blog/introducing-operators.html

[3] CNCF Operator Working Group. (2021). *Operator White Paper v1.0*.
https://github.com/cncf/tag-app-delivery/blob/main/operator-whitepaper/v1/Operator-WhitePaper_v1-0.md

[4] Google, Microsoft, Amazon Web Services. (2024). *kro: Kubernetes Resource Orchestrator*.
https://kro.run

[5] Orkestra Project. (2026). *Production metrics: process_resident_memory_bytes 49,176,576*.
Internal measurement, live deployment.

[6] Datadog. (2024). *State of Kubernetes Security 2024*.
https://www.datadoghq.com/state-of-kubernetes-security/
