---
title: "Index"
weight: 53
---

# Orkestra Registry

OrkestraRegistry is the operator ecosystem for Orkestra — a collection of
declarative operator patterns that can be imported, composed, and overridden
in any Orkestra runtime.

Traditional operator ecosystems distribute binaries. OrkestraRegistry
distributes **behavior** — YAML Katalogs that declare how a CRD should
reconcile, what resources it should create, and how it should convert between
versions. The Orkestra runtime interprets these patterns. Multiple patterns
compose into a single, efficient process.

```yaml
sources:
  registry:
    - url: ghcr.io/orkestra-sh/registry/postgres:v14
      oci: true
    - url: ghcr.io/orkestra-sh/registry/redis
      version: v7.2.0
    - url: ghcr.io/my-repo/my-registry/monitoring
        branch: main
```

That is a complete multi-operator platform. No binaries. No deployments.
No separate processes for each operator. One Orkestra runtime interprets
all of them.

---

## What the registry contains

OrkestraRegistry has three layers, each building on the last.

**Internal resource library** — the low-level Go implementations for
Deployments, Services, Secrets, ConfigMaps, Jobs, CronJobs, and more.
These live inside the Orkestra codebase and are the primitives the
GenericReconciler calls when it executes declarative templates. They are
stable, tested, and maintained as part of Orkestra core.

**Public Katalog registry** — versioned operator patterns maintained in
the `orkestra-sh/registry` repository and published as OCI
artifacts. Each pattern is a complete Katalog with the CRD definition,
the reconcile templates, example CRs, and documentation.

**Typed extensions** — Go hooks and custom reconcilers for use cases that
cannot yet be expressed declaratively. These are versioned modules that users
reference in their Katalogs. Successful typed extensions are candidates for
promotion to declarative core Katalogs over time.

---

## Distribution: OCI artifacts

Patterns in the public registry are published as OCI artifacts — the same
distribution format as container images. This means:

- Any OCI-compatible registry can host them: GHCR, Docker Hub, private
  Artifactory, AWS ECR
- Standard tooling (`oras`, `crane`) works with them out of the box
- Semantic versioning and immutable tags apply exactly as they do for images
- Discoverability through Artifact Hub follows the same conventions as Helm
  charts and other OCI-distributed artifacts

```bash
# Pull a pattern to inspect it
oras pull ghcr.io/orkestra-sh/registry/postgres:v14

# Reference it in a Komposer — Orkestra fetches and caches automatically
sources:
  registry:
    - url: hcr.io/orkestra-sh/registry/postgres:v14
    oci: true
```

{{< callout type="note" >}}
The `ork registry` CLI for push, pull, search, and list commands is
currently in development. OCI artifacts can be consumed today via direct
`oci: true` references in Komposers. Full CLI documentation will be published
when the commands reach stable.
{{< /callout >}}

---

## Key properties

**Composable.** Import multiple patterns from multiple sources — OCI refs,
Git registries, Helm charts, local files — and compose them in a single
Komposer. Override only what differs for your environment.

**Versioned.** Each pattern follows semantic versioning. Pin to a version,
track a branch, or reference an exact commit SHA. Declarative conversion
rules handle breaking changes between pattern versions.

**Overridable.** A pattern is a default, not a constraint. Override workers,
resync intervals, resource limits, or entire reconcile templates inline in
your Komposer without forking the pattern.

**Promotable.** Patterns start declarative or typed. Typed extensions that
prove their value become declarative core Katalogs. Core Katalogs that prove
general enough become built-in Orkestra features. The ecosystem evolves
upward.

---

## Quick start

Install Orkestra and use a pattern from the official registry in under
five minutes.

```bash
# Install
brew install iAlexeze/tap/ork

# Scaffold a project
ork init my-platform
cd my-platform
```

Create a `komposer.yaml`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: my-platform
sources:
  registry:
    - url: ghcr.io/orkestra-sh/registry/postgres:v14
      oci: true
  files:
    - ./local-overrides.yaml
spec:
  crds:
    postgres:
      workers: 4    # production override
```

```bash
ork validate --katalog komposer.yaml
ork run --katalog komposer.yaml
```

{{< callout type="tip" >}}
Run `ork validate` before `ork run`. It resolves OCI references, merges
all sources, enriches built-in Kinds, and surfaces configuration errors
before any reconciliation begins.
{{< /callout >}}

---

<!-- **Next:** [How It Works →](./how-it-works.md) -->
