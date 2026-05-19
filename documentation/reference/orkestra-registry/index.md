# Orkestra Registry

OrkestraRegistry is the operator ecosystem for Orkestra — a collection of declarative operator patterns and reusable motifs that can be imported, composed, and overridden in any Orkestra runtime.

Traditional operator ecosystems distribute binaries. OrkestraRegistry distributes **behavior** — YAML Katalogs that declare how a CRD should reconcile, what resources it should create, and how it should convert between versions. The Orkestra runtime interprets these patterns. Multiple patterns compose into a single, efficient process.

```yaml
imports:
  registry:
    - url: ghcr.io/orkspace/orkestra-registry/postgres@v14
    - url: ghcr.io/orkspace/orkestra-registry/redis@v7
    - url: ghcr.io/myorg/patterns/interna-platform@v2
```

That is a complete multi-operator platform. No separate deployments. No separate processes. One Orkestra runtime interprets all of them.

---

## What the registry contains

**Patterns** — complete, declarative operators expressed as Katalogs. Import a pattern and Orkestra manages that CRD for you: create, update, delete, conversion, status, notifications. No Go code required from the consumer.

**Motifs** — reusable building blocks imported at the CRD level in a Katalog. A motif declares named inputs and contributes resources, status configuration, and admission rules (validation and mutation) to any CRD that imports it. Motifs are how complex operators are composed from smaller, well-tested pieces.

**Typed extensions** — Go hooks for use cases that cannot yet be expressed declaratively. Versioned Go modules that any Katalog can reference. The bridge between what YAML can express today and what it will express tomorrow.

---

## Distribution: OCI artifacts

Patterns and motifs are published as OCI artifacts. This means:

- Any OCI-compatible registry can host them: GHCR, Docker Hub, private Artifactory, AWS ECR
- Standard tooling (`oras`, `crane`, `docker`) works with them
- Semantic versioning and immutable tags apply exactly as they do for images
- Version pins in Komposers never change behavior without a version bump

```bash
# Pull a pattern to inspect it
ork registry pull postgres:v14

# Reference it in a Komposer — Orkestra fetches and caches automatically
imports:
  registry:
    - url: ghcr.io/orkspace/orkestra-registry/postgres@v14
```

---

## The promotion ladder

The registry is built around a deliberate progression:

```
Typed extension (Go hooks)
    │
    │   use case becomes well understood
    ↓
Core pattern (declarative Katalog)
    │
    │   pattern proves general enough
    ↓
Orkestra built-in (template capability)
```

This is how Orkestra's declarative model grows. Community patterns exercise the frontier. Successful patterns illuminate the next feature. Orkestra absorbs them. Operators that needed Go today will not always need Go.

---

## Quick start

```bash
# Install
brew install orkspace/tap/ork

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

imports:
  registry:
    - url: ghcr.io/orkspace/orkestra-registry/postgres@v14
  files:
    - ./local-overrides.yaml

spec:
  crds:
    postgres:
      workers: 4    # override — wins over the pattern's default
```

```bash
ork validate --file komposer.yaml
ork run --file komposer.yaml
```

---

→ [How it works](how-it-works.md) — the three layers in detail: internal resource library, public registry, OCI distribution, and the `ork registry` CLI
