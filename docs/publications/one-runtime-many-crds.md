# One Runtime, Many CRDs: A New Model for Kubernetes Operator Management

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operators have become the standard mechanism for extending cluster capabilities, yet their development remains bound to a long‑standing orthodoxy: one operator per CRD. This principle, codified in official best practices, emerged from sound engineering concerns—separation of concerns, encapsulation, and the UNIX philosophy of "do one thing and do it well". However, this architectural constraint has produced an unintended consequence: operator sprawl. Production clusters routinely run dozens of operators, each consuming memory, CPU, and API server resources, each with its own lifecycle, upgrade cadence, and failure domain.

This paper argues that the prohibition against multi‑CRD operators reflects constraints of implementation, not necessity. When the operator runtime is designed to manage multiple CRDs with proper isolation, the benefits of consolidation outweigh the risks. We present Orkestra, a runtime that treats each CRD as an isolated unit—with per‑CRD workers, queues, health endpoints, and failure domains—while sharing a single control plane. This model preserves the separation of concerns that operator frameworks advocate while eliminating the operational overhead of operator sprawl. We examine the original concerns that led to the one‑operator‑per‑CRD principle and demonstrate how a well‑architected multi‑CRD runtime addresses each of them. Finally, we show that this approach enables new capabilities: unified observability, declarative composition, dependency management across CRDs, and a novel declarative approach to multi‑version CRD conversion—features that are difficult or impossible to achieve with separate operators.

---

## 1. Introduction

Kubernetes operators have transformed how we manage complex applications on the platform. By encoding operational knowledge into custom controllers, operators automate tasks that would otherwise require human intervention: provisioning databases, managing certificates, configuring monitoring stacks, and maintaining storage systems.

The operator pattern has succeeded beyond its creators' expectations. Today, organizations routinely run 20, 50, or even 100 operators in a single cluster. Each operator represents a distinct binary, a distinct deployment, a distinct set of RBAC rules, a distinct upgrade path, and—crucially—a distinct failure domain. The cumulative overhead is substantial.

This proliferation is not accidental. It stems from a design principle that has been consistently reinforced by operator frameworks: "Avoid a design solution where more than one Kind is reconciled by the same controller". The Operator SDK best practices document elaborates: "Having many Kinds (such as CRDs) which are all managed by the same controller usually goes against the design proposed by controller‑runtime. Furthermore this might hurt concepts such as encapsulation, the Single Responsibility Principle, and Cohesion".

This guidance is sound in principle. A controller that manages multiple CRDs risks becoming a tangled monolith where changes to one resource inadvertently affect another. The UNIX philosophy of "do one thing and do it well" has proven its value across computing.

Yet the guidance was formulated in a specific context: the controller‑runtime framework and its assumptions about controller implementation. The question this paper explores is whether the principle is intrinsic to the operator pattern or incidental to the implementation frameworks that dominated early operator development.

We propose that the core insight of operator design is not "one controller per CRD" but rather "one reconciler per CRD, with proper isolation." When the runtime itself provides this isolation, a single controller process can safely manage many CRDs.

The remainder of this paper is organized as follows. Section 2 examines the original concerns that led to the one‑operator‑per‑CRD orthodoxy. Section 3 introduces the Orkestra model, a runtime designed for multi‑CRD management with strong isolation. Section 4 demonstrates how Orkestra addresses each of the original concerns. Section 5 discusses the additional capabilities enabled by this approach. Section 6 addresses limitations and future work. Section 7 concludes.

---

## 2. The One‑Operator‑Per‑CRD Orthodoxy

### 2.1 Origins of the Principle

The recommendation against multi‑CRD operators appears across Kubernetes operator documentation. The Operator SDK best practices document is explicit: "Avoid a design solution where more than one Kind is reconciled by the same controller". The rationale is grounded in software engineering fundamentals: separation of concerns, the Single Responsibility Principle, and cohesion.

The UNIX philosophy echoes this sentiment: "Do one thing and do it well". Applied to operators, this suggests each operator should manage exactly one kind of resource. The Operator SDK's own summary reinforces this: "One Operator per managed application".

### 2.2 The Stated Concerns

The official best practices articulate several specific concerns with multi‑CRD controllers:

- **Encapsulation:** When multiple CRDs are managed by the same controller, the controller necessarily knows about all of them. This coupling can make it difficult to reason about the system, test individual components, or reuse parts of the controller in other contexts.

- **Single Responsibility Principle:** A controller that manages multiple CRDs inherently has multiple reasons to change. A change to one CRD's behavior risks introducing bugs in another's.

- **Cohesion:** The operations performed for different CRDs may not be logically related. Grouping them in one controller may create artificial dependencies.

- **Unexpected Side Effects:** A change intended for one CRD might inadvertently affect others, especially if they share state or processing logic.

- **Testing Complexity:** Verifying that a multi‑CRD controller works correctly requires testing interactions between CRDs that might otherwise be independent.

### 2.3 The Unintended Consequence: Operator Sprawl

While the guidance aims to maintain clean architecture, it has produced a predictable operational problem: operator sprawl. Consider a typical production cluster:

| Application | Operator |
|-------------|----------|
| Prometheus | Prometheus Operator |
| Grafana | Grafana Operator |
| Cert Manager | Cert Manager Operator |
| Ingress | Nginx Ingress Operator |
| PostgreSQL | PostgreSQL Operator |
| Redis | Redis Operator |
| Kafka | Strimzi Operator |
| Monitoring | (multiple operators) |
| Security | (multiple operators) |

Each operator consumes memory (typically 50‑200 MB), consumes CPU (even when idle, due to informers), maintains its own watch cache, requires its own RBAC, and has its own upgrade schedule.

The problem extends beyond resource consumption. Observability becomes fragmented: each operator exposes its own metrics (or none at all), health is assessed per operator rather than per resource, and understanding the system's overall state requires aggregating data from dozens of sources.

### 2.4 Alternative Approaches to the Cluster‑Wide CRD Problem

A related challenge arises from the cluster‑wide nature of CRDs themselves. As noted in recent discussions, CRDs are cluster‑scoped resources, which means all tenants in a cluster share the same CRD definitions. This creates version compatibility issues: team A might need CRD version v1alpha2, while team B needs v1beta1, but only one can be installed.

The conventional response has been to create separate clusters per team—an expensive solution that introduces its own operational overhead. This problem highlights the need for better tooling around CRD management and isolation.

---

## 3. The Orkestra Model

Orkestra is a runtime designed for multi‑CRD management with strong isolation. Rather than treating each CRD as a separate operator process, Orkestra runs a single process that manages many CRDs while preserving separation of concerns.

### 3.1 Core Design Principles

- **Per‑CRD Isolation:** Every CRD in Orkestra receives its own:
  - Informer (with per‑CRD resync intervals)
  - Workqueue (with per‑CRD depth and backoff)
  - Worker pool (configurable concurrency)
  - Health endpoint and metrics
  - Failure domain (a panic in one reconciler does not affect others)

- **Declarative Configuration:** Users define CRDs in a Katalog—a YAML file that declares the CRD, its dependencies, and the resources it should create. No Go code is required.

- **Dependency Management:** CRDs can declare dependencies using `dependsOn`. Orkestra starts CRDs in topological order and shuts them down in reverse order.

- **Built‑in Observability:** Every CRD automatically exposes health endpoints (`/katalog/{crd}/health`) and metrics (reconcile count, latency, queue depth, worker count).

- **Runtime Composition:** Komposers allow composing multiple Katalogs from files, Helm charts, and remote URLs—with environment‑specific overrides.

- **Declarative Version Conversion:** Orkestra serves as a built‑in conversion webhook, applying declarative conversion rules defined in the Katalog. No Go code, no separate webhook deployment, no TLS management.

### 3.2 Implementation Architecture

Orkestra's implementation ensures isolation while sharing infrastructure:

```
┌─────────────────────────────────────────────────────────────────┐
│                       Orkestra Runtime                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │   Health    │ │   Kube-     │ │   Event     │               │
│  │   Server    │ │   client    │ │   Recorder  │               │
│  └─────────────┘ └─────────────┘ └─────────────┘               │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                Informer Factory                          │    │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │    │
│  │  │  Pod        │ │  Website    │ │  Database   │        │    │
│  │  │  Informer   │ │  Informer   │ │  Informer   │        │    │
│  │  │  resync:30s │ │  resync:5m  │ │  resync:1m  │        │    │
│  │  └─────────────┘ └─────────────┘ └─────────────┘        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Workqueue Registry                          │    │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │    │
│  │  │  Pod Queue  │ │ Website Q   │ │ Database Q  │        │    │
│  │  │ depth:1000  │ │ depth:500   │ │ depth:2000  │        │    │
│  │  └─────────────┘ └─────────────┘ └─────────────┘        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Worker Pools (per‑CRD)                       │    │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │    │
│  │  │ 3 workers   │ │ 2 workers   │ │ 5 workers   │        │    │
│  │  │ (Pod)       │ │ (Website)   │ │ (Database)  │        │    │
│  │  └─────────────┘ └─────────────┘ └─────────────┘        │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

Each component is designed for isolation:

- **Informer Factory:** Creates a separate informer for each CRD with its own resync interval. Informers share only the API server connection.
- **Queue Registry:** Maintains a separate workqueue per CRD, each with independent depth limits and backoff settings.
- **Worker Pools:** Each CRD has a dedicated pool of worker goroutines. A panic in one worker is caught and does not affect other CRDs.
- **Health Endpoints:** Each CRD has its own `/katalog/{crd}/health` endpoint, enabling fine‑grained monitoring.

### 3.3 The Komposer: Declarative Composition

A key innovation in Orkestra is the Komposer, which allows platform teams to publish Katalogs and application teams to compose them with environment‑specific overrides:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer
sources:
  files:
    - ./katalogs/project.yaml
    - https://raw.github.com/myorg/crds/main/katalog.yaml
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
spec:
  crds:
    - name: application
      workers: 8   # override for production
```

This enables organization‑wide standardization while preserving team autonomy. Platform teams define base Katalogs; application teams consume and override only what differs.

### 3.4 Declarative Version Conversion

Orkestra includes a built‑in conversion webhook that applies declarative rules defined in the Katalog. The user declares how fields map between versions, and Orkestra handles the conversion when the API server requests it.

```yaml
conversion:
  - kind: Website
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          autoscaling:
            enabled: false   # default for v1alpha1 resources
      - from: v1
        to: v1alpha1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          # autoscaling dropped — it didn't exist in v1alpha1
```

Orkestra's HTTPS server serves the `/convert` endpoint, and the CRD's `conversion` block points to it. No separate webhook deployment. No Go code. No TLS management (certificates are required). Conversion metrics are automatically exposed.

For a detailed treatment, see the companion paper: **Declarative Version Conversion for Kubernetes CRDs**.

---

## 4. Addressing the Original Concerns

We now examine each of the concerns raised against multi‑CRD controllers and show how Orkestra's architecture addresses them.

### 4.1 Encapsulation

The original concern: "Having many Kinds which are all managed by the same controller usually goes against the design proposed by controller‑runtime".

In Orkestra, each CRD's reconciler is encapsulated. The runtime dispatches events to the appropriate reconciler based on GVK, but the reconcilers themselves are independent. A change to one CRD's reconciler does not affect others.

Furthermore, Orkestra's per‑CRD configuration allows each CRD to be tuned independently. One CRD may have 10 workers and a 5‑minute resync; another may have 2 workers and a 30‑second resync. The operator that manages both is not a monolith; it is a platform that hosts independent reconcilers.

### 4.2 Single Responsibility Principle

The original concern: A controller that manages multiple CRDs has multiple reasons to change.

Orkestra's architecture inverts this. The runtime itself has a single responsibility: hosting reconcilers. Each reconciler maintains the Single Responsibility Principle at the level of its own code. The runtime is infrastructure; the reconcilers are logic.

This is analogous to Kubernetes itself. The kube‑controller‑manager runs dozens of controllers (Deployment controller, ReplicaSet controller, Endpoint controller, etc.) in a single process. No one argues that this violates the Single Responsibility Principle because each controller is a separate unit with its own responsibility.

### 4.3 Cohesion

The original concern: Operations for different CRDs may not be logically related.

In Orkestra, the runtime provides no logic beyond dispatching. Cohesion is maintained at the reconciler level. The runtime does not need to know what any CRD does—it merely provides the infrastructure for reconciliation.

Moreover, Orkestra enables *positive* cohesion that separate operators cannot achieve. When CRDs are managed by separate operators, there is no coordination between them. A database CRD and an application CRD that depends on it have no way to coordinate startup order. Orkestra's `dependsOn` feature provides this coordination declaratively.

### 4.4 Unexpected Side Effects

The original concern: A change intended for one CRD might inadvertently affect others.

Because Orkestra's reconcilers are isolated and communicate only through the shared runtime (via ready channels for dependencies), side effects are controlled. A panic in one reconciler is caught and logged; other reconcilers continue unaffected. Per‑CRD queues ensure that a backlog in one CRD's queue does not starve others.

### 4.5 Testing Complexity

The original concern: Testing a multi‑CRD controller requires testing interactions between CRDs.

Orkestra's isolation means each reconciler can be tested independently. The runtime itself can be tested in isolation using fake Kubernetes clients. The interactions that matter—dependencies—are explicit and can be tested with integration tests that mock ready channels.

---

## 5. New Capabilities Enabled by Multi‑CRD Management

Beyond addressing the original concerns, a well‑architected multi‑CRD runtime enables capabilities that are difficult or impossible with separate operators.

### 5.1 Unified Observability

With separate operators, observability is fragmented. Each operator exposes its own metrics (if it exposes any at all). There is no single view of what CRDs are present, how they are performing, or how they relate.

Orkestra provides a unified observability layer:

```bash
GET /katalog                 # All CRDs, health, config, dependency graph
GET /katalog/{crd}           # Single CRD — config, stats, reconciler info
GET /katalog/{crd}/health    # 200 healthy / 503 degraded
GET /metrics                 # Prometheus metrics for all CRDs
```

All metrics use the full GVK string as the `crd` label, enabling per‑CRD dashboards:

```
controller_reconcile_total{crd="demo.orkestra.io/v1alpha1, Kind=Website", result="success"} 1247
controller_workers_active{crd="demo.orkestra.io/v1alpha1, Kind=Website"} 3
controller_queue_depth{crd="apps/v1, Kind=Deployment"} 0
orkestra_conversion_requests_total{kind="Website", from_version="v1alpha1", to_version="v1", result="success"} 47
```

This unified view is essential for platform engineers who need to understand the health of all extensions to their cluster.

### 5.2 Cross‑CRD Dependencies

When operators are separate, dependencies between CRDs must be managed externally—if they are managed at all. Operators typically have no awareness of each other.

Orkestra allows CRDs to declare dependencies:

```yaml
crds:
  - name: application
    dependsOn:
      - database
```

The runtime starts CRDs in topological order and shuts them down in reverse order. Dependents block until dependencies are ready, even if dependencies appear after startup.

This feature is critical for complex systems where resources have ordering requirements. It is a natural extension of the operator pattern that separate operators cannot provide.

### 5.3 Declarative Composition

Platform engineering often requires that certain CRDs be present in every cluster, with specific configurations. Separate operators do not provide a mechanism for this composition; each operator must be installed independently.

Orkestra's Komposer allows platform teams to publish Katalogs that are composed at runtime:

```yaml
sources:
  files:
    - ./katalogs/namespaces.yaml
    - ./katalogs/rbac.yaml
    - ./katalogs/monitoring.yaml
```

Application teams can then consume these Katalogs and override only what they need:

```yaml
sources:
  files:
    - https://platform.internal/katalogs/production.yaml
spec:
  crds:
    - name: database
      workers: 8   # production needs more workers
```

This model enables GitOps workflows where the entire operator configuration is versioned in Git and promoted across environments.

### 5.4 Resource Efficiency

The resource efficiency of a consolidated runtime is substantial. A single Orkestra instance managing 5 CRDs with 170+ resources consumes:

| Metric | Value |
|--------|-------|
| Memory | 98 MB |
| CPU | 0.05 cores |
| Goroutines | 86 |

By comparison, 5 separate operators would likely consume 5–10 times the memory and CPU. Each operator would maintain its own informer cache, its own metrics endpoint, its own leader election mechanism.

### 5.5 Declarative Version Conversion

The standard approach to multi‑version CRDs requires writing a conversion webhook in Go, deploying it as a separate service, managing TLS certificates, and maintaining conversion logic across versions. This infrastructure overhead often outweighs the benefit of adding a new version.

Orkestra eliminates this complexity. Conversion rules are declared in the Katalog, the same place where reconciliation logic is defined. Orkestra's existing HTTPS server handles the `/convert` endpoint. Metrics for conversion requests, latency, and errors are automatically exposed.

```bash
# See conversion statistics per CRD
curl localhost:8080/katalog/website-v1 | jq '.conversion'
{
  "enabled": true,
  "total": 62,
  "success": 62,
  "failures": 0,
  "avgLatencyMs": 0.5,
  "p95LatencyMs": 1.2
}
```

This is the first time version conversion has been made declarative and observable.

---

## 6. Limitations and Future Work

The Orkestra model is not without limitations. While it addresses the concerns raised against multi‑CRD operators, it introduces new considerations.

**Shared Failure Domain.** While Orkestra's per‑CRD reconciler isolation ensures that a panic in one reconciler does not crash others, the runtime process itself is a single point of failure. In practice, this is mitigated by leader election and warm caches on follower pods. When the leader fails, a follower with an already‑warm cache takes over.

Critically, this architecture mirrors Kubernetes' own `kube-controller-manager`, which runs dozens of controllers in a single process. Kubernetes has operated this way for years without systemic failure. For production deployments, Orkestra recommends running at least two replicas with PodAntiAffinity rules to ensure pods are scheduled on different nodes, preventing node failure from affecting all replicas. This is identical to how `kube-controller-manager` is deployed in production clusters.

**Per‑CRD Lifecycle Management.** Separate operators can be upgraded independently. Orkestra is a single binary that manages all CRDs. This is both a feature and a constraint: upgrading Orkestra upgrades all CRD management logic at once, which may be desirable for consistency but may be problematic for teams that need to control upgrade cadence per CRD.

Future work includes exploring plugin architectures that would allow reconcilers to be loaded dynamically, enabling per‑CRD upgrades without restarting the runtime.

---

## 7. Conclusion

The recommendation against multi‑CRD operators emerged from sound engineering principles and the constraints of early operator frameworks. But the principle is not inherent to the operator pattern; it is a property of the implementation.

When the runtime itself provides proper isolation—per‑CRD informers, queues, workers, failure domains—a single process can safely manage many CRDs. This consolidation eliminates operator sprawl, reduces resource consumption, and enables new capabilities: unified observability, cross‑CRD dependencies, declarative composition, efficient resource utilization, and declarative version conversion.

Orkestra demonstrates that the core insight of operator design is not "one operator per CRD" but rather "one reconciler per CRD, with isolation." By providing that isolation in the runtime, Orkestra makes multi‑CRD management not just possible but preferable.

The Kubernetes ecosystem has long needed a way to manage operators without managing operator sprawl. Orkestra offers a path forward: one runtime, many CRDs, and the tools to make it work.

---

## References

[1] Operator SDK Documentation, "Common recommendations and suggestions," operator-sdk.netlify.app, 2024. [Online]. Available: https://operator-sdk.netlify.app/docs/best-practices/common-recommendation/

[2] OpenShift Documentation, "Limitations for multitenant Operator management," GitHub, 2024. [Online]. Available: https://raw.githubusercontent.com/openshift/openshift-docs/main/modules/olm-operatorgroups-limitations.adoc

[3] G. Kambalimath, "One Operator To Rule Them All? CRD Management Strategies for Cloud Native Apps," KubeCon + CloudNativeCon, 2026. [Online]. Available: https://www.classcentral.com/course/youtube-one-operator-to-rule-them-all-crd-management-strategies-for-cloud-native-apps-guna-kambalimath-479571

[4] Operator SDK, "Operator Best Practices," GitHub, 2018. [Online]. Available: https://github.com/operator-framework/operator-sdk/blob/d331456f64b6b14619881c45944862dffde27bdc/website/content/en/docs/best-practices/best-practices.md

[5] Operator SDK, "Customize generating operator framework," Programmer Sought, 2024. [Online]. Available: https://www.programmersought.com/article/860010821415/

[6] Operator Framework, "Operator Best Practices," GitHub, 2019. [Online]. Available: https://github.com/operator-framework/community-operators/blob/ba43453d3ed9b329694dac00d83c896be67c9a8c/docs/best-practices.md

[7] vCluster, "A solution to the problem of cluster-wide CRDs," vCluster Blog, 2024. [Online]. Available: https://www.vcluster.com/blog/solution-clusterwide-crds

[8] C. Schlotter and F. Pandini, "Kubernetes CRD Design for the Long Haul," KubeCon EU 2025. [Online]. Available: https://qiita.com/reoring/items/b8a880ceef1d766b4944

[9] Operator SDK Issue #3182, "Should I create two controllers for one CRD If their watching predicates are different?," GitHub, 2020. [Online]. Available: https://github.com/operator-framework/operator-sdk/issues/3182