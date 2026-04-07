---
title: "Vision"
weight: 51
---

# The Vision

OrkestraRegistry is infrastructure for an ecosystem that does not fully exist
yet. This page explains where it is going and why the direction is right.

---

## Where the operator ecosystem is today

OperatorHub.io lists hundreds of operators. Each is a separate binary. Each
runs as a separate process. Each has its own release cycle, its own metrics
format, its own health story, its own memory footprint. Installing ten
operators means ten deployments consuming gigabytes of memory for the control
plane work of watching CRDs.

The ecosystem grew this way because the operator pattern was defined as
one-binary-per-CRD. Every operator is a software project. Every software
project is a maintenance burden. Platform teams spend enormous effort
maintaining operators that were not central to their business — they just
needed a database, or a certificate manager, or an ingress controller.

This is the problem the OrkestraRegistry solves at scale.

---

## What the registry changes

When operator behavior is a declaration rather than a binary, the economics
of the operator ecosystem change fundamentally.

**Publishing an operator becomes as easy as publishing a Helm chart.** You
write a Katalog, package it as an OCI artifact, push it. There is no binary
to compile, no image to build, no runtime to maintain. The Orkestra runtime
is already installed. Your Katalog tells it what to do.

**Consuming an operator becomes as easy as pulling a container image.** One
line in a Komposer. The runtime fetches the artifact, merges it, and starts
managing the CRD. No Helm releases. No CRD installation separate from
operator installation. No RBAC to configure manually.

**Operating ten operators becomes the same as operating one.** They all run
inside a single Orkestra process. One health endpoint. One metrics endpoint.
One deployment to monitor. One upgrade to perform. Ten operators, one
operational surface.

---

## The Promotion Ladder

The registry is designed around a progression. Use cases that need Go today
will not always need Go.

```
Typed extension (Go hooks)
    |
    |   when the pattern is understood and widely used
    ↓  
Core Katalog (declarative YAML)
    |
    |   when the pattern is general enough to be built-in
    ↓  
Orkestra feature (built-in template capability)
```

This is not a requirement imposed on contributors. It is the natural trajectory
of successful patterns. A typed extension that handles 95% of use cases
declaratively, with hooks only for the remaining 5%, invites the question:
what would it take to express that 5% in YAML? When the answer is found, the
typed extension becomes a core Katalog.

This is how Orkestra's declarative capability grows — not through top-down
design, but through community use revealing what is truly general.

{{< callout type="note" >}}
Today, Orkestra's declarative templates cover resource creation, drift
correction, dependency ordering, and multi-version conversion. The next
layer — expressing conditional logic and external calls declaratively —
is where the next generation of promotions will come from.
{{< /callout >}}

---

## Discoverability through Artifact Hub

Artifact Hub is the discovery layer for the cloud-native ecosystem. Helm
charts, OPA policies, Falco rules, and Kyverno policies are all discoverable
through it. OrkestraRegistry patterns will be too.

Each pattern will have an Artifact Hub entry with:

- A description of what CRD it manages
- The versions available
- The fields it accepts
- Which overrides are recommended for production
- Links to the source Katalog and example CRs

This makes operator discovery consistent with how teams already find Helm
charts. The difference is what they get when they import one: not a binary,
but a declaration.

---

## Private Registries

The OCI distribution model means the registry is not a single point of
failure or a single point of control. Any OCI-compatible registry can host
Orkestra patterns.

Platform teams at large organizations can maintain private registries — a
curated set of approved, internally tested patterns. They push their own
versions of the postgres Katalog with organization-specific defaults applied.
Application teams import from the private registry instead of the public one.
The security posture is the same as container image policy: control which
registries your cluster can pull from.

```yaml
# Internal komposer — references private registry
sources:
  registry:
    - url: //ghcr.io/orkestra-sh/registry/postgres:v14
      oci: true
      auth:
        type: basic
        usernameFromEnv: REGISTRY_USER
        passwordFromEnv: REGISTRY_PASSWORD
```

The mechanics are identical to public consumption. Only the registry URL
and authentication differ.

---

## The Long Term

The OrkestraRegistry vision converges with the Orkestra project's broader
direction: if operators can be data, and data can be distributed through
standard package infrastructure, the operator ecosystem can grow at the
pace of Helm — not at the pace of binary software releases.

The registry is the mechanism. The patterns are the content. The runtime
is the executor. When all three are in place, building a platform operator
becomes a declaration: what CRD, what behavior, which version. The platform
makes it so.

{{< callout type="tip" title="Contributing" >}}
The registry repository is open for contributions at
`github.com/orkestra-sh/registry`. The contribution guide
explains how to structure a new pattern, the five required files, and
the review process for promotion to core status.
{{< /callout >}}

---

<!-- **Next:** [Technical Documentation →](../technical-docs/registry-technical-docs.md) -->
