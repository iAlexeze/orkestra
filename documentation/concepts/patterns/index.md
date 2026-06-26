# Orkestra Patterns

Operators are not unique. Across hundreds of teams building Kubernetes operators, the same problems appear repeatedly: gate a Deployment on an upstream health check, inject live config from an external service, enforce ordered deletion, propagate CRD status from child resource state, compose operators across teams without coupling them.

These are not one-off requirements. They are **patterns** — recurring solutions to recurring problems in the operator world.

Orkestra formalizes this. Every file you write — `katalog.yaml`, `komposer.yaml`, `simulate.yaml` — is an Orkestra Pattern: a versioned, named, distributable artifact with a specific kind and a specific job.

---

## The Pattern kinds

| Kind | File | Job |
|------|------|-----|
| **Katalog** | `katalog.yaml` | Declares a CRD and its full operator behavior — resources, status, admission, lifecycle |
| **Motif** | `motif.yaml` | A reusable resource blueprint imported by Katalogs — security posture, RBAC, probes, retry policy |
| **Komposer** | `komposer.yaml` | Wires multiple Katalogs from registry, file, or Helm sources into a running platform |
| **E2E** | `e2e.yaml` | Verifies operator behavior against a real cluster — declarative, no test framework |
| **Simulate** | `simulate.yaml` | Verifies reconciler behavior in memory — no cluster, sub-second |

Each kind solves one problem. You compose them as needed.

---

## How they relate

Motifs are imported by Katalogs. Katalogs are imported by Komposers. E2E and Simulate are the testing layer — they reference Katalogs and CRs, and they compose into suites with `imports:`.

A small platform might be one Katalog. A large one might be a Komposer pulling three Katalogs from the registry, each built from shared Motifs, with a Simulate suite for CI and an E2E suite for pre-deploy verification.

---

## Patterns are portable

A Pattern is not tied to the team that wrote it. It is an OCI artifact, versioned and distributable:

```text
ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
```

A team publishes a Katalog that knows how to manage PostgreSQL. Another team imports it in a Komposer, overrides the parts specific to their environment, and runs it. The behavior travels with the artifact — not with a binary, not with a Helm chart, not with tribal knowledge.

This is the same shift containers made for applications. Patterns make operator behavior portable.

---

## Why not "document"?

YAML files are documents. Patterns are something more specific: they encode a solution to a named problem that recurs in the operator world. The name reflects intent — not format.

---

## Pages

| Page | What it covers |
|------|---------------|
| [Scaffolding a pattern](scaffold.md) | `ork create pattern` — generate the full file set to build, test, and publish |

---

→ See the [Pattern kinds in the registry](../../orkestra-registry/index.md) — examples of Katalogs, Motifs, and Komposers in practice.
