---
title: "Operator Authoring Vs Infrastructure Controlplane"
weight: 68
---

# Operator Authoring vs Infrastructure Control Plane

Orkestra and Crossplane are often mentioned in the same conversation. Both
are Kubernetes-native. Both are declarative. Both deal with CRDs. Engineers
evaluating one will encounter the other — and with Crossplane v2 explicitly
expanding to cover applications, the question of how they differ deserves
a precise and honest answer.

They are solving fundamentally different problems from fundamentally different
positions. Understanding the distinction is not about competition. It is about
choosing the right tool with clarity.

---

## What Crossplane is

Crossplane is a control plane composition framework. It provides a model for
assembling resources — cloud infrastructure, Kubernetes objects, third-party
services — into higher-level abstractions that platform teams expose to
application developers.

You write a Composition that describes how to assemble resources. A
CompositeResourceDefinition (XRD) defines the API users interact with. When
a user creates a Composite Resource (XR), Crossplane's composition engine
renders the Composition and creates the declared resources.

Crossplane v1 was designed primarily for external infrastructure — RDS
databases, S3 buckets, GKE clusters, DNS records. Crossplane v2, released
August 2025, deliberately expanded this model to include Kubernetes-native
resources like Deployments and Services as first-class composition targets.
This is genuine and significant progress. The Crossplane team acknowledged
directly that v1's architecture had become overly opinionated and rebuilt
with everything they had learned.

---

## What Orkestra is

Orkestra is an operator authoring runtime. It provides a model for building
Kubernetes operators for your domain CRDs without writing Go code.

You write a Katalog that declares what your CRD should do. When a user creates
a CR, Orkestra's reconcile loop runs — creating child resources, correcting
drift, managing finalizers, writing status, emitting events. The operator
behavior is entirely declared. No Composition to write. No function to deploy.
No framework machinery in the user-facing API.

---

## The question Crossplane v2 asks

Crossplane v2 is a sincere attempt to address a real limitation. Their own
framing from the v2 announcement:

*"Crossplane v2 is better suited to building control planes for applications,
not just infrastructure."*

Their v2 example of an application composite resource:

```yaml
apiVersion: example.crossplane.io/v1
kind: App
metadata:
  namespace: default
  name: my-app
spec:
  image: nginx
  crossplane:
    compositionRef:
      name: app-kcl
    compositionRevisionRef:
      name: app-kcl-41b6efe
  resourceRefs:
    - apiVersion: apps/v1
      kind: Deployment
      name: my-app-9bj8j
    - apiVersion: v1
      kind: Service
      name: my-app-bflc4
```

The Crossplane team explicitly moved framework machinery under `spec.crossplane`
to make it easier for users to identify which fields matter to them. This is
good ergonomic progress.

But notice what cannot be removed: `spec.crossplane` is still there. The
`compositionRef`, `compositionRevisionRef`, and `resourceRefs` are Crossplane
concepts that every user of this API encounters. The CR is Crossplane's CR —
shaped by Crossplane's model, carrying Crossplane's machinery.

---

## The question Orkestra asks

Orkestra asks a different question entirely.

Not: *how do we make Crossplane's model work for applications?*

But: *what does a user's CR look like if the framework is completely invisible?*

```yaml
apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: my-site
spec:
  image: nginx:1.25
  replicas: 2
  port: 8080
```

No `spec.crossplane` or `spec.orkestra` fields. No compositionRef. No resourceRefs. No framework
machinery of any kind. The CR is pure domain. Your users think about websites,
not about operators. The framework's job is to disappear.

This is what *your CRD is enough* means — not as a slogan, but as a technical
guarantee. The user's API surface is defined entirely by your CRD schema.
Orkestra contributes nothing to it.

---

## Five things Crossplane v2 still does not do

Acknowledging Crossplane v2's genuine progress does not mean the gap closed.
Five capabilities that matter to operator authors remain absent:

**1. Declarative status management**

Orkestra writes a standard `Ready` condition after every reconcile
automatically. Declarative status fields are resolved from the live CR
and propagated from child resource status:

```yaml
status:
  fields:
    - path: readyReplicas
      value: "{{ readyReplicas .children.deployment }}"
    - path: endpoint
      value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

After every successful reconcile:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
  readyReplicas: "2"
  endpoint: my-site.default.svc.cluster.local
```

Crossplane Composite Resources have framework-managed conditions, but
declarative status fields resolved from child resource state are not a
Crossplane feature. Your users' status experience is defined by what
Crossplane exposes, not what you declare.

**2. Admission-time validation and mutation**

Orkestra declares deny/warn rules in the Katalog that intercept `kubectl apply`
synchronously when webhooks are enabled, and run at reconcile time regardless:

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "images must come from the internal registry"
    action: deny

mutation:
  - field: spec.replicas
    default: "2"
```

One declaration. Two enforcement points. Crossplane has no equivalent
declarative admission policy model for the APIs you expose through XRDs.

**3. Declarative multi-version CRD conversion**

Orkestra handles CRD version conversion in the Katalog with no Go code:

```yaml
conversion:
  storageVersion: v1
  paths:
    - from: v1alpha1
      to: v1
      spec:
        image: "{{ .spec.image }}"
        seo:
          enabled: false
```

Production result: 62 conversions, 0 failures, 0.5 ms average latency.

Crossplane manages Composition revisions — how a Composition evolves over time.
Kubernetes-level CRD version conversion for the XRDs you define is not handled
declaratively by Crossplane.

**4. Live operational observability**

Orkestra's `/katalog` endpoint exposes the live operational truth of every
managed CRD — queue depth, p95 reconcile latency, worker utilisation, active
validation warnings, conversion statistics:

```json
{
  "name": "website",
  "healthy": true,
  "queueDepth": 0,
  "reconcileTotal": 8312,
  "reconcileErrors": 3,
  "conversion": {"p95LatencyMs": 1.2}
}
```

`ork status` queries this API. Dashboards are built on it. There is no
equivalent live per-CRD operational API in Crossplane.

**5. Continuous reconciliation — not composition rendering**

The deepest difference is architectural. Crossplane is a composition engine
that renders Compositions. Orkestra is a reconcile loop that continuously
enforces desired state. These are different primitives with different guarantees.

When a Deployment is manually deleted in a Crossplane-managed stack, the
composition is not automatically re-reconciled to recreate it. When a
Deployment is manually deleted in an Orkestra-managed stack with
`reconcile: true`, it reappears within one resync interval. This is drift
correction — the operator continuously enforces the declared state, every
cycle, whether or not anything changed.

This is what Kubernetes controllers do. It is what `kube-controller-manager`
has always done for Deployments, ReplicaSets, and Jobs. Orkestra brings that
model to custom resources. Crossplane's composition model is different in kind.

---

## The concept model after v2

Crossplane v2 reduced concept count. Claims are no longer required.
Provider-kubernetes Objects are no longer needed for Kubernetes resources.
These were genuine pain points and the team addressed them.

The Crossplane v2 concept model:

- Provider
- Managed Resource
- CompositeResourceDefinition (XRD)
- Composition + Composition Functions (KCL, CEL, Python, or Go)
- Composite Resource (XR)
- Package (Configuration, Provider)
- Composition Revisions

Orkestra's complete concept model:

- Katalog (what to do)
- Komposer (how to compose — optional)

If you know Kubernetes, you can write a working Orkestra Katalog in under an
hour. The Crossplane learning investment remains measured in days to weeks —
not because Crossplane is badly designed, but because it is solving a more
general and genuinely complex problem.

---

## Kubernetes principles first

There is a philosophical dimension worth naming directly.

Crossplane introduces new primitives — XRDs, Compositions, XRs — that live
alongside Kubernetes but are not native to it. Engineers learn a model that
maps to Kubernetes concepts but is not the same as them. The abstraction is
powerful, but it is Crossplane's abstraction.

Orkestra uses Kubernetes primitives throughout. The informer is a standard
Kubernetes informer. The reconcile loop follows the pattern of
`kube-controller-manager`. Owner references, finalizers, events, conditions —
all standard Kubernetes. The Katalog declares what those standard mechanisms
should do. There is nothing to unlearn.

When Kubernetes operators are eventually a native platform primitive — when
`kubectl get katalogs` works without installing anything extra — the path
from Orkestra is a straight line. The concepts are already Kubernetes concepts.
The primitives are already Kubernetes primitives. The only change would be
where the runtime lives.

This is what it means to place Kubernetes first: not just to run on Kubernetes,
but to extend it by strengthening what it already is, rather than building a
new model on top of it.

---

## When to use each

**Use Crossplane when:**

- You are provisioning external infrastructure — cloud databases, storage,
  networks, managed services from cloud providers
- You need the Provider ecosystem (AWS, GCP, Azure, and hundreds of others)
- You are building a platform where engineers self-service infrastructure
  through Kubernetes CRs and those resources live outside the cluster

**Use Orkestra when:**

- You have domain CRDs that should produce Kubernetes child resources
- You want operators without writing Go
- You need admission policy, multi-version conversion, and declarative status
  from a single YAML declaration
- You want governance over existing Kubernetes resources (built-in kind enrichment)
- You want your users' CRs to contain only domain fields — no framework machinery

**Use both when:**

Crossplane provisions the external infrastructure. Orkestra manages the
application-layer operators that consume it. Crossplane creates the RDS
instance. Orkestra manages the `Database` CR that creates the connection Secret,
configures the Deployment, and keeps everything in sync. Each tool does what
it does best. Neither intrudes on the other.

---

## Summary

| | Crossplane v2 | Orkestra |
|---|---|---|
| **Primary purpose** | Infrastructure and application control plane composition | Kubernetes operator authoring |
| **User's CR** | Carries framework machinery (`spec.crossplane`) | Pure domain — framework invisible |
| **Logic expression** | Composition Functions (KCL, CEL, Python, Go) | YAML templates, evaluated at runtime |
| **Concept count** | 7+ | 2 |
| **Learning curve** | Days to weeks | Hours |
| **Reconciliation model** | Composition rendering | Continuous drift-correcting loop |
| **Declarative status** | Limited | Three layers including child propagation |
| **Admission policy** | No | Yes — deny/warn, two enforcement points |
| **Multi-version conversion** | No | Yes — declarative paths, in-process |
| **Live operational API** | No | Yes — `/katalog` per CRD |
| **Kubernetes model** | New model on top of Kubernetes | Kubernetes model throughout |
| **Memory (15 CRDs)** | 500 MB+ (core + providers) | ~47 MB |

Crossplane v2 is a serious project that improved substantially. The expansion
toward applications is evidence that the Kubernetes ecosystem knows the
operator authoring problem is unsolved. The Crossplane team has been building
honestly and openly for seven years and their progress is real.

But Crossplane v2 is still a composition model that requires learning Crossplane.
It still places framework machinery in the user-facing API. It still renders
compositions rather than continuously reconciling desired state.

Orkestra places Kubernetes first, your CRD first, and your users first.
The framework disappears. The operator appears.

That is the difference.