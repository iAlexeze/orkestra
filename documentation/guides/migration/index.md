# Migration Guide

You have a working Kubernetes operator. The question is not whether controller-runtime works — it does. The question is what it costs.

Informers, workqueues, worker pools, leader election, status, finalizers, events, metrics, health endpoints, panic recovery — written from scratch for every CRD. The behaviour is always small. The machinery is always the same.

Orkestra removes the machinery. This guide shows the same WebApp operator expressed five ways so you can see what you are choosing between, and when to pick each.

---

## The pack

```bash
ork init --pack from-controller-runtime
```

Each directory is a self-contained, runnable step. Work through them in order or jump to the option you need.

---

## What you will learn

- What controller-runtime costs in practice, measured against a real operator
- How to go declarative — zero Go, no binary — when it fits
- How the hybrid pattern (90% declarative, 10% Go hook) works and when to use it
- How to drop all declarations and let a Go hook own every resource
- How to lift an existing `Reconcile` method into Orkestra with a one-line signature change
- How `pkg/resources` simplifies the Get / Create / Patch pattern
- How `ork migrate` automates the constructor path for an existing operator file

---

## The options

| Step | Directory | Go required | What you own |
|------|-----------|-------------|--------------|
| Baseline | `00-controller-runtime-baseline` | Yes — full | Everything: informers, manager, scheme, main.go |
| Declarative | `01-declarative` | No | Nothing — pure YAML |
| Hybrid | `02-hybrid` | Yes — hook only | The 10% templates can't express |
| Hooks only | `03-hooks-only` | Yes — all resources | All child resource specs in Go |
| Constructor — lift | `04-constructor-migration` | Yes — full reconciler | Reconcile logic; manager removed |
| Constructor — resources | `05-constructor-orkestra-resources` | Yes — full reconciler | Reconcile logic; resource ops simplified |
| ork migrate | `06-ork-migrate` | — | Automated constructor path from an existing file |

---

## Contents

| Page | What it covers |
|------|----------------|
| [The Baseline](./01-baseline.md) | What controller-runtime costs — line by line |
| [Declarative](./02-declarative.md) | Zero Go, zero binary — pure Katalog |
| [Hybrid](./03-hybrid.md) | 90/10: declare everything Orkestra handles, write Go for the rest |
| [Hooks only](./04-hooks-only.md) | When type-safe control over every resource matters more than YAML |
| [Constructor — lift](./05-constructor.md) | One signature change, every resource op untouched |
| [Constructor — resources](./06-constructor-resources.md) | `pkg/resources`: Get / Create / Patch → one Update call |
| [ork migrate](./07-ork-migrate.md) | Automate the constructor path for an existing operator file |

---

## Before you begin

The pack uses a WebApp CRD. When a CR is applied, the operator creates a Deployment and a Service, then writes status. The behaviour is intentionally small so the migration story is about the machinery, not the domain.

You do not need to understand all five options. Pick the first one that fits your situation and stop there.

**No cluster needed for simulate.** Every option has a `simulate.yaml` that runs in-memory — no cluster, no binary. Simulate before you deploy.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/01-declarative
ork simulate
ork run --dev
```

For typed operators (options 02–06), build first:

```bash
cd from-controller-runtime/02-hybrid
make registry && make build
./ork simulate
./ork run --dev
```

To try the automated path with your own operator:

```bash
cd from-controller-runtime/06-ork-migrate
# Follow the README
ork migrate ../00-controller-runtime-baseline/controller/webapp_controller.go -o ./output
```
