# Why Orkestra?

Kubernetes has always promised declarative infrastructure.
You describe what you want. The platform makes it so.

That promise holds everywhere — until you need to extend Kubernetes itself.

The moment you need a custom resource, you leave the declarative world.
You write Go. You scaffold controllers. You wire informers and schemes.
You manage reconcile loops, error handling, retries, and backoff.
You build images, write tests, maintain boilerplate that nobody asked for.

Every operator framework to date has accepted this as the cost of entry.
Kubebuilder. Operator SDK. Metacontroller. They all make the Go easier.
None of them make it unnecessary.

This creates a paradox: **to make Kubernetes more declarative, you must
write imperative code.**

Orkestra breaks that paradox.

---

## What changed

Orkestra treats operators the same way Kubernetes treats deployments —
as a declaration of intent.

You write a Katalog. You describe the CRDs you want to manage, the resources
that should exist for each CR, and how they should stay in sync. You run
`ork run`. The operator is live.

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
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

```bash
ork run --katalog website.yaml
```

That is the entire operator. Apply a `Website` CR and Orkestra creates the
Deployment and Service. Change a field and Orkestra reconciles. Delete the
CR and Orkestra cleans up. No Go. No code generation. No compilation.

The operator also has a health API, emits Prometheus metrics, supports leader
election, and handles graceful shutdown. Not because you wrote any of that —
because Orkestra provides it for every CRD, always, for free.

---

## Why this is different

Other frameworks lower the barrier to writing Go operators. Orkestra removes
the barrier entirely for the common case. That is not a reduction in
complexity — it is a different category of tool.

**Accessibility.** Platform engineers, SREs, and application teams can now
build and own operators without learning Go. The person who understands the
domain — what a `Website` should do — can write the operator.

**Portability.** A Katalog is YAML. It can be versioned in Git, rendered
with Helm, promoted across environments, diffed in PRs, and shared with
other teams as easily as any other manifest. Operators become artifacts, not
codebases.

**Composability.** A Komposer composes Katalogs from files, Helm charts,
remote URLs, and environment-specific overrides. Platform teams publish
standard CRD definitions. Application teams consume and override them.
The same pattern that Helm brought to deployment manifests now applies to
operator configuration.

**Consistency.** Every CRD managed by Orkestra behaves the same way.
Same health endpoints. Same metrics. Same dependency ordering. Same
lifecycle semantics. An organisation running ten Orkestra operators has
ten operators with identical operational characteristics — not ten different
systems each invented from scratch.

**Escape hatches.** When a CRD needs Go — typed spec access, complex state
machines, external API calls — Orkestra provides Go hooks and custom
constructors. You write only what you need. The framework handles everything
else.

---

## The philosophy

**Declarative first.** If Kubernetes can express it declaratively, Orkestra
should too. Templates, dependencies, finalizers, drift correction — all
declared, none written.

**Composition over code.** Operators should be assembled from declarations
and reusable registry implementations, not programmed from scratch.

**Runtime over build-time.** Behavior should be interpreted at runtime, not
baked into binaries. A change to a Katalog changes operator behavior without
a build, a push, or a deploy.

---

## The bigger picture

The patterns that make Orkestra work — declarative reconciliation, dependency
graphs, the registry model, multi-source composition — are not just
simplifications of existing patterns. They are a different way of thinking
about what an operator is.

An operator is not a controller. It is not a Go binary. It is a declaration
of how a domain concept maps to Kubernetes resources and how that mapping
should be maintained over time.

Orkestra is the runtime that makes that declaration executable.

The [whitepaper](./declarative-operators-whitepaper.md) explores these
patterns and their implications in depth.