# Why Katalog and Komposer Are Not CRDs

When engineers first encounter Orkestra, they ask a reasonable question: why aren't
the Katalog and Komposer themselves Kubernetes Custom Resource Definitions? Every
other operator management tool defines its management objects as CRDs. Orkestra does
not. This document explains why that choice is not an oversight or a limitation.

It is a deliberate design decision — and one that produces a superior operational
model.

---

## The wrong question

The instinct to make everything a CRD comes from a real place. CRDs give you
`kubectl get`, RBAC, audit logs, and GitOps compatibility. These are genuine
benefits. The question "why isn't the Katalog a CRD?" sounds like it is asking
about a missing feature.

But it is asking the wrong question. The right question is: **what is the most
direct path from human intent to cluster state, and what is the right interface
for observing running state?**

These are two different questions. Orkestra answers both separately and correctly.
A CRD-based approach conflates them and answers both worse.

---

## Two concerns, two interfaces

There are two things an operator platform must provide:

**Configuration** — how you declare what your operator should do. This belongs
in a file. Files are versioned in Git, validated in CI, reviewed in pull requests,
composed across teams, distributed as OCI artifacts. The Katalog is a file. You
declare intent once. Orkestra reads it at startup.

**Operational state** — what the operator is actually doing right now. This
belongs in a live API. Not a Kubernetes object showing you what was applied, but
a runtime response reflecting the current truth: which CRDs are healthy, which
have degraded, what the queue depth is, how many reconciles have succeeded,
what active warnings exist, what the conversion latency looks like.

Orkestra serves both through the right interface for each.

The `/katalog` endpoint is the operational state interface:

```bash
# All managed CRDs — live health, reconcile stats, dependency graph
curl localhost:8080/katalog | jq

# One CRD — full detail: workers, queue depth, error rate, active warnings
curl localhost:8080/katalog/website | jq

# Point health check — 200 healthy, 503 degraded
curl localhost:8080/katalog/website/health
```

`ork status` queries `/katalog`. `ork dashboard` will render it in the terminal.
Monitoring systems can scrape it. Out-of-the-box dashboards are on the roadmap.
All of this from a live API that reflects actual running state — not a Kubernetes
object that reflects the declared spec.

---

## Why `/katalog` is superior to `kubectl get katalogs`

A `kubectl get katalog` would show you the spec you applied. It would tell you
what you declared. It cannot tell you what is actually happening.

The `/katalog` endpoint tells you what is actually happening:

```json
{
  "name": "website",
  "gvk": "demo.orkestra.io/v1alpha1, Kind=Website",
  "healthy": true,
  "workers": 3,
  "workersActive": 2,
  "resourceCount": 47,
  "reconcileTotal": 8312,
  "reconcileErrors": 3,
  "queueDepth": 0,
  "conversion": {
    "total": 62,
    "failures": 0,
    "avgLatencyMs": 0.5,
    "p95LatencyMs": 1.2
  },
  "admission": {
    "validationTotal": 1204,
    "validationDenied": 9,
    "mutationApplied": 387,
    "webhooksEnabled": true
  }
}
```

This is the live truth of the operator. No Kubernetes object can carry this
information — it is runtime state, not configuration. Queue depth changes every
second. Reconcile totals increment on every cycle. Active warnings appear and
clear as CRs are created and corrected. p95 conversion latency reflects the
last thousand conversions.

A CRD-based Katalog would show you the spec. The `/katalog` endpoint shows you
the operator. These are not equivalent. The live API is strictly more valuable
than a Kubernetes object would be.

!!! note "ork status is built on /katalog"
    `ork status` is not a separate data source — it is the `/katalog` endpoint
    rendered for the terminal. Anything that can reach the health server can
    consume the same data. This is how out-of-the-box dashboards become possible:
    the API is already there. The dashboard is just a renderer.

---

## The configuration interface

The file is the right interface for configuration because configuration is a
human artifact. You write it, review it, version it, share it.

```bash
# Validate without a cluster — catches errors before they reach production
ork validate --file katalog.yaml

# In CI — exits non-zero on any configuration error
ork validate --file komposer.yaml
```

A Katalog file lives in Git. It is diffable. A pull request that changes a
validation rule from `action: warn` to `action: deny` shows exactly one line
changed. The reviewer sees it. The CI pipeline validates it. The change is
traceable from proposal to production.

!!! tip "Configuration in CI, state in the API"
    This separation is the pattern. Configuration changes go through Git and CI.
    Operational state is observed through `/katalog`. Neither steps on the other.
    The file is not authoritative about what is running — the API is.

A CRD-based Katalog applied to a cluster is harder to version, harder to diff
meaningfully, and requires a running cluster to validate. The file is simply
better for the job configuration needs to do.

The OrkestraRegistry is only possible because Katalogs are files. Patterns are
OCI artifacts containing YAML. They are distributed through the same infrastructure
as container images, pulled with version references, composed in Komposers. A
CRD-based pattern would require installing CRDs in the cluster to define how
other CRDs should behave — a recursive dependency that helps no one.

---

## The problem with a CRD watching CRDs

If the Katalog were a CRD, Orkestra would watch Katalog CRs and act on them.
A CRD watching CRDs to manage other CRDs.

This creates a dependency chain: your CRD cannot be managed until Orkestra is
running, Orkestra cannot configure itself until the Katalog CRD is installed,
the Katalog CRD may require Orkestra's webhooks to validate, and Orkestra's
webhooks require Orkestra to be running. The circularity is manageable with
careful ordering, but it is unnecessary complexity.

More importantly, it makes Orkestra's operational infrastructure visible in your
cluster. `kubectl get crds` shows the Katalog kind alongside your Website kind.
RBAC policies must cover the Katalog kind. Users encounter it when they did not
need to know it existed.

Orkestra refuses to add this noise. Your cluster's CRD surface belongs to your
domain — not to the infrastructure that serves it. When someone runs `kubectl get crds`
in a cluster managed by Orkestra, they see their CRDs. The operator infrastructure
is not there.

!!! note "The clean cluster principle"
    Orkestra manages resources. It does not add resources to represent itself.
    The only objects Orkestra creates in your cluster are the child resources it
    manages on behalf of your CRs, and the operational objects it needs to function
    (leader election lease, webhook configurations). The Katalog itself is not
    a cluster object. It is a file on disk.

---

## Human intent → Orkestra → Kubernetes

The deepest reason the Katalog is a file is what it reveals about the relationship
between you and Orkestra.

You write what you want. Orkestra makes it so.

The Katalog is not a configuration file in the traditional sense — it is a
declaration of intent. *Create a Deployment when a Website is created. Set
replicas to 2 when not declared. Reject images that do not come from our registry.*
These are human statements. Orkestra interprets them and speaks to Kubernetes in
Kubernetes's language: informers, workqueues, reconcile loops, owner references,
events, metrics.

If the Katalog were a CRD, you would be writing Kubernetes-format intent and
waiting for a controller to act on it. The distance between what you mean and how
you express it would grow. The file keeps that distance at zero.

Orkestra understands human intent and speaks native Kubernetes. The Katalog is
where you speak. The `/katalog` endpoint is where Orkestra reports back.

---

## The model in full

```
Katalog (file)          ← where you declare intent
    │
    ▼
ork validate            ← where errors are caught before the cluster sees them
    │
    ▼
Orkestra runtime        ← where intent becomes running operator
    │
    ▼
/katalog endpoint       ← where you observe the live operational truth
    │
    ▼
ork status / dashboards ← where operators see it, rendered
```

Each step has the right interface. The file has the interfaces of files — Git,
diff, CI, pull requests. The running state has the interfaces of running state —
HTTP, JSON, Prometheus, terminal UI, dashboards.

A CRD would collapse these into one interface and do both worse.

---

## The long view

The current design does not preclude Katalog and Komposer becoming native
Kubernetes kinds. In the long-term vision for Orkestra, they are registered by
the cluster itself — not managed by a user-space operator, but managed by
Kubernetes core the same way Deployments and Services are managed. At that point,
`kubectl get katalogs` is correct because the Katalog is a first-class Kubernetes
object understood by the platform, not an overlay managed by another operator.

Until then, the file is the right model. It keeps the cluster surface clean,
keeps configuration in version control, keeps the ecosystem open, and keeps
operational truth in a live API that reflects what is actually running.

The `/katalog` endpoint is not a workaround for the absence of a Katalog CRD.
It is the better answer to a different and more important question.

A CRD tells you what was applied. The live API tells you what is running.

Both matter. Orkestra provides both. Through the right interface for each.