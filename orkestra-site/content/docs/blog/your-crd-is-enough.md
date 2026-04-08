---
title: "Your Crd Is Enough"
weight: 8
---

# Your CRD Is Enough

*The engineering argument for why the operator should have always been the CRD.*

---

## What CRDs Were Actually Introduced to Do

In 2017, Kubernetes shipped Custom Resource Definitions as the stable mechanism
for extending its API. Before CRDs, extension required forking the Kubernetes
source code or running a separate API server. The community wanted something
simpler: a way to teach Kubernetes about new resource types without touching
its internals.

The design that emerged was deliberate. A CRD is not a plugin. It is a
declaration. You tell Kubernetes: "there is a new kind of object in the world,
it belongs to this group, it looks like this schema, and here is what to call
it." Kubernetes responds by doing everything it knows how to do with objects:
storing them, serving them, watching them, validating them, handling updates
and deletions. All of the platform's existing machinery applies automatically
to your new resource type.

The question CRDs left unanswered was intentional: *what happens when someone
creates one of these objects?* Kubernetes deliberately said nothing. That was
the operator's job.

So the relationship was established:

- **Kubernetes** answers "how does this object exist?"
- **The operator** answers "what should happen when it does?"

This is the correct division of responsibility. Kubernetes is a general-purpose
platform. It should not know what a `Website` or a `Database` means. The
operator encodes that domain knowledge.

---

## What Kubernetes Actually Needs

Here is the question the ecosystem failed to ask: when the API server receives
a `Website` CR, what does it actually need from the operator?

Not much.

The API server needs to know when something changed. It already does — that is
what the watch mechanism is for. The API server needs to know the schema for
validation. You already provided that in the CRD. The API server needs the
object stored. It already stores it.

What the operator adds is reconciliation: watching for changes and calling the
Kubernetes API to bring the actual state of the cluster into alignment with
the declared state. That is the entire job.

Now ask: what does reconciliation actually require?

1. A watch on the resource — the informer
2. A queue for changes — the workqueue
3. Workers to process that queue
4. Logic that reads the CR and creates/updates/deletes child resources

Items 1, 2, and 3 are identical for every operator ever written. They differ
only in which resource is being watched and how many workers to run. The
plumbing is fungible. **Only item 4 contains any domain-specific logic.**

This is the insight that Orkestra is built on. The plumbing can be provided
by a runtime. The logic can be expressed as a declaration. The operator becomes
a Katalog.

---

## What Kubernetes Already Has

When you apply a CRD to a cluster, Kubernetes immediately knows:

```
Group:    demo.orkestra.io
Version:  v1alpha1
Kind:     Website
Plural:   websites
Scope:    Namespaced
Schema:   { spec: { image: string, replicas: integer, port: integer } }
```

It knows this because you declared it. It stores this in
`customresourcedefinitions.apiextensions.k8s.io`. It is available through the
discovery API at any moment, to any process that asks.

Orkestra asks. At startup, when it reads a Katalog entry:

```yaml
- name: website
  apiTypes:
    kind: Website
```

it queries the cluster's discovery API and retrieves everything it needs to
watch that resource. The group, version, plural, scope — all of it. The user
does not need to declare these because the cluster already has them. The CRD
definition is the source of truth. Orkestra reads it rather than making the
user duplicate it.

This is not a convenience feature. It is a consequence of taking seriously what
CRDs were designed to do. CRDs are declarations. Everything downstream of a
declaration should be able to derive what it needs from that declaration, not
require the author to repeat it.

---

## What Operators Were Forced to Duplicate

Every operator framework that came before Orkestra required the user to
re-declare what the CRD had already declared.

You wrote the CRD:

```yaml
group: demo.orkestra.io
version: v1alpha1
kind: Website
```

Then you wrote a Go struct:

```go
type Website struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              WebsiteSpec `json:"spec"`
}

type WebsiteSpec struct {
    Image    string `json:"image"`
    Replicas int    `json:"replicas"`
    Port     int    `json:"port"`
}
```

Then you registered it in a scheme:

```go
var SchemeBuilder = &scheme.Builder{GroupVersion: schema.GroupVersion{
    Group: "demo.orkestra.io", Version: "v1alpha1",
}}
```

Then you generated deep-copy functions. Then you registered the scheme with
the manager. Then you wired the reconciler to watch this type.

All of this to tell the operator what the CRD had already told the cluster.
The information was not new. It was already there. The framework demanded that
you say it again, in Go, with all the overhead that entails.

This is the root cause of operator complexity. Not the reconcile logic — that
is often straightforward. The complexity is in the mandatory ceremony of
re-stating what the CRD already declared.

Kubernetes stores every CRD object as `map[string]interface{}`. It does not
use typed Go structs internally. The typed layer in operator frameworks was
added for developer convenience — and became a dependency that everyone had
to satisfy.

Orkestra removes the dependency. It operates on unstructured objects because
that is what the cluster stores. There is nothing to re-declare. The CRD is
the declaration.

---

## The Operator as a Data Problem

The core insight: reconciliation is not a programming problem. It is a data
problem.

The reconciler receives an object. The object has a spec. The spec declares
the desired state. The reconciler must translate that desired state into
Kubernetes API calls. Every reconciler ever written does exactly this.

The translation — "`spec.image` maps to `deployment.spec.containers[0].image`,
`spec.replicas` maps to `deployment.spec.replicas`" — is not logic. It is a
mapping. **Mappings are data. Data can be declared.**

```yaml
reconciler:
  default: true
  onCreate:
    deployments:
      - image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
```

This is the mapping. The template expressions are not programs. They are
references: "the value here comes from here." Orkestra's template resolver
evaluates them against the live object and produces the literal values that
the registry needs.

The registry — OrkestraRegistry — is the standard implementation of
"given this spec, make this Kubernetes resource exist." It handles create,
update, delete, owner references, idempotency, drift detection. It is
the same code that every operator would have written, extracted into a
shared library.

The operator is therefore: a mapping (the Katalog) applied to a runtime
(Orkestra) using a library (OrkestraRegistry). No language required.

---

## What a Complete Operator Actually Is

When a `Website` CR is applied and Orkestra reconciles it, the following
happens — automatically, for free, with no code written:

The informer sees the new object and enqueues it. A worker picks it up and
reads the CR from the informer cache — zero API calls. Finalizers are added
via a merge patch so the CR cannot be deleted until cleanup completes. The
template resolver evaluates each field in the Katalog against the CR's
unstructured map. The registry creates a Deployment with the resolved image,
replica count, and owner reference. The registry creates a Service with the
resolved port and the correct pod selector. Both resources are marked for
drift correction — if someone manually changes the Deployment's image, the
next reconcile cycle will restore it. A `Reconciled` event is emitted on the
CR. Five Prometheus metrics are incremented. The health endpoint reflects the
successful reconcile. When the CR is eventually deleted, the finalizer holds
it in terminating state while the registry deletes the Deployment and Service,
then removes the finalizer, and the CR is gone.

That is a complete, production-grade operator. The user wrote twelve lines
of YAML. Nothing else.

---

## The Super-Operator Model

There is a common assumption that "zero code" means "lightweight." The
opposite is true.

In a traditional framework, an operator has whatever you built into it. You
wrote a reconcile function. It does what you told it to do. If you forgot
finalizers, there are no finalizers. If you forgot metrics, there are no
metrics. If you forgot events, there are no events.

In Orkestra, every CRD gets the full operator stack — not because you asked
for it, but because the runtime provides it unconditionally:

- Its own informer with configurable resync
- Its own workqueue with independent depth and backoff
- Its own worker pool — no other CRD can consume its workers
- Its own health endpoint at `/katalog/{crd}/health`
- Five Prometheus metrics labeled by full GVK
- Kubernetes event emission on every operation
- Finalizer management
- Owner references on every child resource
- Cascade deletion
- Drift correction
- Leader election participation
- Graceful shutdown with in-flight reconcile completion

This is more than most hand-written operators provide. Not because Orkestra
is doing something magical, but because the runtime can afford to give every
CRD everything when the cost of building it is shared.

Your CRD does not get a lightweight shim. It gets a super-operator.

---

## The Unfinished Promise

CRDs were introduced so that the Kubernetes API could be extended declaratively.
You declare a new resource type. Kubernetes handles the rest — storage,
validation, serving, watching.

The promise was never completed. Kubernetes handled the platform side. The
operator side required you to leave the declarative world and write a program.
The declaration ended at the CRD, and the imperative code began.

Orkestra completes the promise. You declare the CRD. You declare the Katalog.
Orkestra handles the rest — reconciliation, drift correction, lifecycle,
observability.

The declaration does not end at the CRD. The declaration *is* the operator.

That is what "your CRD is enough" means. Not that the CRD contains magic.
But that everything Kubernetes needs to manage your resource is already
present in the declaration you made. The runtime reads it. The platform
enforces it. The operator emerges from the combination.

No code required.