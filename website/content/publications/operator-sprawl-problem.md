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

---

## 2. The per-operator cost structure

Each operator imposes a repeating cost in four dimensions.

### 2.1 Memory and CPU

Each operator is a Go binary running as a Kubernetes Deployment. A minimal
operator — informer, workqueue, reconcile loop, health endpoint — consumes
approximately 50–80 MB of resident memory at steady state.

For a cluster running twenty operators, this is 1–4 GB of memory allocated
purely to control plane processes.

### 2.2 API server load

Each operator maintains one or more informer watch connections to the API server.
Twenty operators with informers watching twenty different resource types means twenty parallel
streams of watch events flowing from the API server.

### 2.3 Development time

The CNCF Operator White Paper estimated three to six weeks for a minimally
viable operator using Kubebuilder or Operator SDK [3].

### 2.4 Operational burden

Different health endpoint paths and response formats. Different Prometheus
metric names and label conventions. Different log formats and verbosity levels.
Different upgrade procedures. Different failure modes. Different on-call
runbooks — twenty documents.

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

---

## 5. The Orkestra resolution

Orkestra's shared runtime model addresses the operator sprawl problem at its
root by replacing the per-binary cost structure with a per-declaration cost
structure.

**Applying the case study numbers:**

| | 20 separate operators | Orkestra |
|---|---|---|
| Memory (per cluster) | ~1.8 GB | ~50 MB |
| Memory (3 clusters) | ~5.4 GB | ~150 MB |
| Internal operator dev | 8 months | 2 weeks |
| Annual maintenance | 3 weeks | 0.5 weeks |
| Operational surfaces | 20 | 1 |

---

## 7. Conclusion

Operator sprawl is a predictable consequence of the per-binary operator model
applied at scale. The cost is real, measurable, and compounding.

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
