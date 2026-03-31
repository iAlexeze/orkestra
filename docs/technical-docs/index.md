# Technical Documentation

This section covers Orkestra's internals — the components, their responsibilities, how they fit together, and the design decisions behind them. It is written for engineers who are contributing to Orkestra, debugging production issues, or building advanced integrations.

If you are new to Orkestra, start with [Getting Started](../getting-started/index.md) and [Your CRD Is Enough](../blog/your-crd-is-enough.md) before reading this section.

---

## Architecture overview

Orkestra is a single Go binary. When `ork run --katalog katalog.yaml` is invoked, the following sequence happens:

```
ork run
  │
  ▼
konstructOrkestra()
  │  reads environment variables
  │  builds Merger
  │  calls KomposeKatalogFromYaml()
  │
  ▼
Merger
  │  resolves all sources (files, helm, registry)
  │  merges CRD entries by name
  │  applies inline overrides last
  │
  ▼
Katalog
  │  enriches built-in kinds from cluster discovery
  │  registers conversion rules → InMemoryConversionRegistry
  │  registers admission rules  → InMemoryAdmissionRegistry
  │  returns []CRDEntry
  │
  ▼
Orkestra runtime
  │  for each CRDEntry:
  │    creates informer
  │    creates workqueue
  │    creates worker pool
  │    creates GenericReconciler
  │  resolves dependency graph
  │  starts konductor election (leader election)
  │  starts components in dependency order
  │
  ▼
HealthServer
  │  serves /health, /ready, /katalog, /metrics (HTTP :8080)
  │  serves /convert, /validate, /mutate (HTTPS :8443) when enabled
  │
  ▼
Running — informers watch, workers reconcile, health reports
```

---

## Component map

| Component | Package | Responsibility |
|---|---|---|
| [konstructOrkestra](./konstructor.md) | `internal/` | Startup wiring — assembles all komponents, builds closures, hands to Orkestra |
| [HealthServer](./health-server.md) | `health/` | HTTP(S) server: health, metrics, katalog API, conversion, validation, mutation |
| [Merger](./merger.md) | `pkg/merger/` | Source resolution: files, helm, registry → unified CRD entry list |
| [Katalog](./katalog.md) | `pkg/katalog/` | Loading, enrichment, conversion registry, admission registry |
| [GenericReconciler](./generic-reconciler.md) | `pkg/reconciler/` | Per-CRD reconcile loop: lifecycle, finalizers, templates, hooks, metrics |
| [Informer + Queue](./informer-factory.md) | `pkg/informer/` | Per-CRD informer factory, typed and dynamic informers |
| [Kontroller](./kontroller.md) | `pkg/kontroller/` | DependencyKontroller, KontrollerRegistry, QueueRegistry, CRDHealth, HTTP handlers |
| [OrkestraRegistry](./orkestra-registry.md) | `pkg/orkestra-registry/` | Resource implementations: Deployment, Service, Secret, etc. |
| [Template Resolver](./orkestra-registry.md#the-resolver) | `pkg/orkestra-registry/template/` | Go text/template evaluation against live CR objects |
| [ork generate](./ork-generate.md) | `cmd/ork/` | Code generation for typed-mode CRDs, hooks, and constructors |
| Conversion | `health/conversion*.go` | ConversionReview handler, path lookup, spec resolution |
| Admission | `health/admission*.go` | AdmissionReview handler, validation and mutation at admission time |

---

## Key design invariants

These invariants hold across the entire codebase. Any change that violates them requires explicit justification.

**Per-CRD isolation.** Each CRD entry has its own informer, workqueue, worker pool, and reconciler closure. No two CRDs share workers or queue items. A failure in one CRD's reconciler does not affect others.

**Dynamic by default.** Orkestra operates on `*unstructured.Unstructured` for all CRDs unless `apiTypes.location` is set. The typed layer exists only for hooks that require Go struct access.

**Idempotent reconciliation.** Every reconcile operation is safe to retry. Create calls check for existence. Update calls apply desired state regardless of current state. The reconciler does not track "was this already created."

**Owner references on all child resources.** Every resource created by the OrkestraRegistry has owner references pointing to the CR. Cascade deletion is automatic.

**Single source of truth.** The Katalog declaration is the authority. Orkestra continuously reconciles the cluster toward what the Katalog declares. Manual changes to child resources are corrected on the next reconcile cycle when `reconcile: true` is set.

---

## Packages reference

```
github.com/ialexeze/orkestra/
  cmd/                      CLI entry points
    ork/                    main binary
  domain/                   Core interfaces (Object, Komponent, ReconcileHooks)
  health/                   HealthServer, conversion, admission handlers, stats
  pkg/
    katalog/                Katalog loading, enrichment, registries
    merger/                 Source resolution (files, helm, registry)
    reconciler/             GenericReconciler, conditions, validation, mutation
    kontroller/             Informer factory, queue registry, health tracking
    types/                  All public types (CRDEntry, APITypes, ValidationRule, etc.)
    orkestra-registry/      Resource implementations (Deployment, Service, etc.)
      template/             Template resolver
      deployments/
      services/
      secrets/
      configmaps/
      jobs/
      cronjobs/
      pods/
      serviceaccounts/
    konfig/                 Configuration constants (kind names, defaults)
    kubeclient/             Kubernetes client wrapper
    logger/                 Structured logging (zerolog)
    utils/                  File loading, auth
```
