# Ecosystem

---

## How does Orkestra compare to kro?

kro (Kubernetes Resource Orchestrator) was announced in 2024 by Google, Microsoft,
and AWS. It allows declaring `ResourceGraphDefinitions` that compose Kubernetes
resources declaratively. The core insight — operator behavior should be a declaration
— is the same insight Orkestra is built on.

The differences are significant:

| | kro | Orkestra |
|---|---|---|
| **Per-CRD isolation** | No — shared reconcile context | Yes — dedicated informer, queue, workers |
| **Multi-version CRDs** | No | Yes — declarative conversion paths |
| **Registry/distribution** | No | Yes — OCI artifacts, Artifact Hub |
| **Admission webhooks** | No | Yes — validation and mutation |
| **Health API** | No | Yes — per-CRD endpoints and Prometheus |
| **Observability** | No | Yes — Control Center, per-CRD health endpoints, Prometheus |
| **Hooks for external logic** | No | Yes — typed and dynamic Go hooks |

kro is a composability layer. Orkestra is a runtime. The fact that three major cloud
providers independently arrived at the same insight validates the direction. Orkestra
is the complete version of what they were reaching for.

---

## Can Orkestra manage third-party CRDs?

Yes — any CRD that Kubernetes accepts, Orkestra can watch and reconcile. No fork, no
reverse engineering, no changes to the CRD definition needed.

```yaml
spec:
  crds:
    prometheus:
      apiTypes:
        group: monitoring.coreos.com
        version: v1
        kind: Prometheus
        plural: prometheuses
      operatorBox:
        onCreate:
          # governance, companion resources, defaults
```

This is how governance patterns work — you apply Orkestra's validation and mutation
model to CRDs you did not write and cannot modify.

---

## What is the path to Kubernetes core?

See [Declarative Operators: A New Model for Kubernetes Extensibility](/publications/declarative-operators-whitepaper/)
for the full argument.

The short version: Orkestra is building toward CNCF Sandbox, then a Kubernetes
Enhancement Proposal, then alpha/beta/GA integration into `kube-controller-manager`.
The target timeline is five years. The prerequisite is production adoption at multiple
organisations, with metrics.

The Katalog and Komposer becoming native Kubernetes kinds — `kubectl get katalogs` —
is the end state. At that point, every cluster ships with a meta-controller that
understands declarative operator definitions. Platform teams write Katalogs. Kubernetes
manages them.

---

## Next

- **[Roadmap](../roadmap.md)** — what is shipped and what is coming
- **[Getting Started](../getting-started/index.md)** — your first operator in minutes
