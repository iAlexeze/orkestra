# Why Orkestra?

Kubernetes has always promised declarative infrastructure.
You describe what you want. The platform makes it so.

That promise holds everywhere — until you need to extend Kubernetes itself.

The moment you need a custom resource, you leave the declarative world.
You write Go. You scaffold controllers. You wire informers and schemes.
You manage reconcile loops, retries, and backoff. You build images. You
write deployment manifests for your controller. You maintain a project
whose primary purpose is to watch another project.

Every major operator framework to date has accepted this as the cost of
entry. Kubebuilder, Operator SDK, Metacontroller — they each make the Go
easier. None of them make it unnecessary.

This creates a paradox: **to make Kubernetes more declarative, you must
write imperative code.**

Orkestra breaks that paradox.

---

## How we got here

Understanding why Orkestra exists requires understanding how the operator
pattern evolved — not as a criticism of what came before, but as an honest
account of how each improvement introduced the problem the next one tried
to solve.

**The original operator pattern** (2016) was elegant in concept: a controller
that watches a custom resource and reconciles the cluster toward the desired
state declared in it. The implementation required deep familiarity with
client-go internals — informers, workqueues, schemes, RESTMappers. Most of
that code was identical across operators. Only the business logic differed.

**Kubebuilder and Operator SDK** solved the boilerplate problem. Code
generation, scaffolding, and controller-runtime reduced hundreds of lines
to dozens. The operator developer could focus on the reconcile function
rather than the plumbing. This was real progress.

But something subtle happened: the abstraction layer got thicker. The
operator was now a project with a Makefile, a generated directory, a
controller-runtime dependency, a test suite for framework scaffolding
before any business logic was written. The cost of entry dropped — but
it did not disappear. You still needed to know Go. You still needed to
build and push an image. You still needed a deployment manifest for your
controller.

**Admission webhooks** arrived to solve the policy problem. When a user
applied a CR with invalid fields, there was no mechanism to reject it
synchronously — the API server had no visibility into domain constraints.
Webhooks provided synchronous interception: the API server calls an external
HTTP endpoint during admission, which validates or mutates the object before
it is stored.

This worked. It also required deploying a separate webhook server,
provisioning TLS certificates, registering webhook configuration objects,
and maintaining that server as a dependency of the API server itself. Policy
enforcement became a second project alongside the operator.

**Policy engines** — OPA, Kyverno, Gatekeeper — emerged to solve this in
the general case. Rather than each team writing their own webhook server, a
shared policy engine accepts declarations and enforces them across the cluster.
Progress again. But now the platform team maintains three things: the CRD
definition, the operator, and the policy declarations. Three separate systems.
Three separate observability stories. Three separate failure modes.

Each step in this progression was correct given the constraints of its time.
The progression itself reveals something important: the fundamental problem
was never the Go, or the boilerplate, or the webhook infrastructure.

**The fundamental problem was that Kubernetes had no permanent observer
watching what was being provisioned.** Every tool that emerged — operator
frameworks, webhook servers, policy engines — existed to compensate for
that absence.

Orkestra is that observer.

---

## The core idea

Orkestra treats operators the same way Kubernetes treats Deployments — as
a declaration of intent. You write a Katalog. You describe the CRDs you
want managed, the resources that should exist for each CR, and the
constraints that must hold. You run `ork run`. The operator is live.

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    - name: website
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      validation:
        - field: spec.image
          prefix: "myorg/"
          message: "image must be from the myorg registry"
          action: deny
        - field: metadata.labels.team
          operator: exists
          message: "all resources should declare a team owner"
          action: warn
      mutation:
        - field: spec.replicas
          default: "2"
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
```

This is a complete platform operator. It validates. It defaults. It creates
and drift-corrects child resources. It emits Kubernetes events, exposes
Prometheus metrics, supports leader election, and handles graceful shutdown.

No Go. No code generation. No webhook server. No policy engine. No admission
controller. No additional deployment.

---

## Why this is different

Other frameworks lower the barrier to writing Go operators. Orkestra removes
the barrier entirely for the common case. That is not a reduction in
complexity — it is a different category of tool.

**Accessibility.** Platform engineers, SREs, and application teams can now
build and own operators without learning Go. The person who understands the
domain can write the operator. The institutional knowledge lives in the
Katalog, not in a Go binary that only the original author understands.

**Portability.** A Katalog is YAML. It can be versioned in Git, rendered
with Helm, promoted across environments, diffed in pull requests, and shared
with other teams like any other manifest. Operators become infrastructure
artifacts rather than software projects.

**Composability.** A Komposer composes Katalogs from files, Helm charts, and
remote URLs. Platform teams publish standard CRD definitions with built-in
policy. Application teams consume and override what they need. The same
pattern that Helm brought to deployment manifests now applies to operator
behavior. Operators become composable.

**A unified policy interface.** Because Orkestra watches every CR it manages,
validation and mutation run inside the reconcile loop — not in a separate
webhook server. Denials block reconciliation and emit Warning events. Warnings
advise without blocking and surface as active violations on the health API.
The entire policy model is declared in the same Katalog as the operator
behavior and observable through the same metrics endpoint, without any
additional infrastructure.

**Consistency.** Every CRD managed by Orkestra behaves the same way. Same
health API. Same metrics. Same dependency ordering. Same lifecycle. An
organisation running twenty Orkestra operators has twenty operators with
identical operational characteristics — not twenty systems each invented
from scratch.

---

## The bigger picture

Admission webhooks exist because the API server has no operator watching
your CRDs — it must delegate validation externally. Policy engines exist
because operator frameworks have no built-in policy model. Both are correct
solutions to real problems. Both exist because of the same underlying
absence: no permanent, trusted observer.

Orkestra is that observer. Because it watches every CR it manages before
any child resources are created, the external delegation becomes unnecessary.
Validation runs in-process. Policy is declared alongside behavior. The
feedback loop — from CR creation to Warning event — is sub-second. No
network hop. No TLS. No external availability dependency.

This matters now, for the teams adopting Orkestra today.

It matters more in the longer term for what it suggests about the future
of Kubernetes extensibility. If an operator can be a declaration, and
policy can be a declaration, and composition can be a declaration — then
the observer that interprets those declarations belongs inside Kubernetes
itself. A native meta-controller that understands Katalogs and Komposers
would mean every cluster has an operator runtime without installation.
For built-in resources — Deployments, Namespaces, Pods — the Kind alone
is enough. Kubernetes already knows the schema. For custom resources, the
CRD definition provides it. Either way, the Katalog is the interface and
Kubernetes is the runtime.

Platform teams write Katalogs. Kubernetes manages them. Less tooling. Fewer
CRD types per resolution. One native interface for both admission and CRD
management — unified, composable, and observable by default.

That is not where we are today. But the solution will speak for itself.
Every Katalog written today, every operator simplified, every webhook server
decommissioned, is evidence that the direction is right.

---

## What Orkestra is

**A seer** — it watches everything it manages. CRDs, built-in resources,
validation outcomes, reconcile history. It knows the managed surface better
than any individual operator because it manages the whole surface simultaneously.

**A balancer** — per-CRD worker pools, queue management, dependency ordering,
per-CRD resync intervals. It allocates reconciliation resources across CRD
types based on actual demand.

**An organizer** — it composes operator behavior from multiple sources,
enforces policy across the managed surface, and provides the runtime that
makes declarative operators real.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.
It was always possible.
It just needed someone to build it.