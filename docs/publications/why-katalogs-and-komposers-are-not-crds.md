# Why Katalog and Komposer Are Not CRDs

When engineers first encounter Orkestra, they ask a reasonable question: why aren't
the Katalog and Komposer themselves Kubernetes Custom Resource Definitions? Every
other operator management tool — Operator Lifecycle Manager, kro, KubeVela — defines
its management objects as CRDs. Orkestra does not. This document explains why that
choice is not an oversight. It is the founding design decision from which everything
else in Orkestra follows.

---

## The problem with a CRD watching CRDs

If the Katalog were a Kubernetes CRD, the following would be true:

A CRD (the Katalog kind) would be required to watch and manage other CRDs. You would
install the Katalog CRD before you could describe your operator. You would apply a
Katalog CR to define the operator for your CRD. Orkestra would watch the Katalog CR
and act on it to manage your actual CRD.

This creates a layered indirection that contradicts the thing Orkestra is trying to
say: **your CRD is enough**.

The moment the Katalog becomes a CRD, your CRD is no longer enough. You need the
Katalog CRD too. And the Komposer CRD. You have traded one operator binary for a
set of management CRDs that require their own operator to function. The complexity
moves rather than disappearing.

There is also a subtler problem. A CRD that watches other CRDs creates a dependency
on Orkestra's own presence in the cluster before any of your CRDs can be managed.
The Orkestra runtime must be healthy for your operator to do anything. This is the
same problem that OLM creates — the management infrastructure becomes a dependency
of every operator it manages. If the management infrastructure degrades, everything
it manages degrades with it.

Orkestra rejects this model entirely.

---

## Katalog and Komposer as intent, not objects

Katalog and Komposer are YAML documents. They are files. They live on disk, in Git,
in Helm charts, in OCI artifacts. They are not Kubernetes objects and they are not
stored in etcd. They do not have a lifecycle in the cluster. They are not watched by
any controller.

They are the medium through which you communicate your intent to Orkestra.

When you write a Katalog, you are saying: *here is what I want to happen when someone
creates a `Website` resource.* When you write a Komposer, you are saying: *here are
the sources I want to compose into one runtime.* These are declarations of intent —
not deployable artifacts, not managed objects.

Orkestra reads them once at startup. It understands what you mean. Then it goes to
work in the cluster, speaking native Kubernetes — creating informers, registering
watches, managing child resources with owner references, emitting events, exposing
metrics. From that point forward, the Katalog is not running anywhere. The cluster
is running. Orkestra is running. Your CRD is running.

The distinction is important: **the Katalog describes the behavior, Orkestra
produces the behavior, the CRD is the behavior.**

---

## Human intent → Kubernetes

Orkestra's job is translation.

You understand your domain. You know what a `Website` resource should do — it should
produce a Deployment and a Service. You know what constraints it should have — the
image must come from your registry. You know what defaults it should carry — two
replicas if none are specified. You know what should happen when it is deleted — a
cleanup Job should run first.

Kubernetes understands none of this. Kubernetes understands Deployments, Services,
Pods, watch events, and control loops. It does not understand what a Website is or
what it should do.

The Katalog is the place where you speak. Orkestra listens. Then Orkestra speaks to
Kubernetes in the language Kubernetes understands. This is the translation layer:
human intent → Orkestra → native Kubernetes.

The Katalog does not need to be a Kubernetes object because it is not speaking to
Kubernetes. It is speaking to Orkestra. Orkestra is the interpreter.

If the Katalog were a CRD, Orkestra would need to watch for Katalog CRs to understand
your intent, which means you would be speaking to Kubernetes and Kubernetes would be
passing your message to Orkestra. You would be encoding your human intent in a format
designed for machines and then having a machine decode it. The translation would go:
human intent → Kubernetes format → Orkestra → Kubernetes. The intermediate step adds
nothing except complexity.

By keeping the Katalog as a plain YAML document, the translation is direct:
human intent → Orkestra → Kubernetes.

---

## The CRD remains the focus

This design keeps the focus exactly where it belongs: on your CRD.

Your CRD is the API. It is the thing your users interact with. It is the schema that
describes your domain. When someone in your organisation creates a `Website` resource,
they are not thinking about Katalogs or Composors. They are thinking about websites.
The Katalog is invisible to them.

If the Katalog were a CRD, it would be visible. Users would need to understand the
difference between a `Website` (their resource) and a `Katalog` (the operator
definition). They would see both in `kubectl get crds`. They would encounter both
in RBAC policies. They would need to understand the relationship between them.

Orkestra refuses to create that confusion. The Katalog is an implementation detail
of how Orkestra works, not a user-facing concept in the cluster. Users see:

```bash
kubectl get websites
kubectl describe website my-site
kubectl get events --field-selector involvedObject.name=my-site
```

They do not see:

```bash
kubectl get katalogs           # this does not exist
kubectl describe katalog ...   # this does not exist
```

The cluster's surface area is clean. Your domain objects are the only objects. The
operator infrastructure is invisible — which is exactly what good operator infrastructure
should be.

---

## Opportunities without limits

There is a practical consequence of this design that is easy to miss.

Because the Katalog is a file rather than a CRD, you can put it anywhere. In a Git
repository. In an OCI artifact. In a Helm chart. In a local file. In a Komposer that
references all of the above. You can version it. You can branch it. You can diff it.
You can review it in a pull request. You can validate it in CI without a running cluster.

```bash
ork validate --katalog katalog.yaml   # no cluster needed
```

A CRD-based approach cannot do this cleanly. A Katalog CR requires a cluster with
Orkestra installed to be valid. A Katalog YAML requires nothing — it is valid or
invalid based on its own structure.

This is what makes the OrkestraRegistry possible. Patterns are OCI artifacts
containing YAML files. They are distributed through the same infrastructure as
container images. They are consumed with a URL reference. None of this would be
natural if patterns were CRDs — you cannot distribute CRDs as OCI artifacts and
pull them with a version reference.

Because the Katalog is data, the ecosystem built around it is a data ecosystem.
Version control. Package registries. Composition. Overrides. All the patterns that
the software world uses to manage shared code apply directly to Katalog patterns,
because Katalogs are text files.

The CRD-as-CRD approach would make Orkestra yet another closed ecosystem, where
patterns can only be shared within the constraints of the management CRD format.
The file-as-data approach makes Orkestra an open ecosystem, where patterns can
be shared through any infrastructure that can host a file.

---

## The long view

In the long-term vision for Orkestra, Katalog and Komposer become native Kubernetes
resource kinds — registered by the cluster itself, RBAC-controlled, auditable. But
this is not a contradiction of the current design. It is its natural completion.

When Orkestra is part of Kubernetes core, the Katalog kind is a first-class concept
in the Kubernetes API, not a management overlay on top of it. The Kubernetes API
server does not need an operator to manage Katalog objects — it manages them natively,
the way it manages Deployments and Services. At that point, a CRD that watches a
Katalog is the right model, because the Katalog is no longer managed by a separate
operator. It is managed by the cluster itself.

Until then, the Katalog is a file. It is the right model for now because it is
the model that keeps the cluster surface clean, keeps the user's focus on their CRD,
and keeps the ecosystem open.

Your CRD is enough. Everything else is how Orkestra serves it.