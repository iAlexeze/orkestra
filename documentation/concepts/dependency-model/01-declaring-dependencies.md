# Declaring Dependencies

In your Katalog, you declare dependencies with `dependsOn`. Three formats are supported — use whichever fits your style.

---

## Format 1 — Simple list

The most common format. Each entry is a CRD name. The condition defaults to `started`.

```yaml
crds:
  database:
    crdFile: database-crd.yaml

  application:
    crdFile: application-crd.yaml
    dependsOn:
      - database      # condition: started (default)
```

---

## Format 2 — Key-value map

When you need different conditions per dependency:

```yaml
crds:
  application:
    dependsOn:
      database: started
      cache: ready
```

---

## Format 3 — Full map

The most explicit form. Useful when tooling or schema validation requires structured objects:

```yaml
crds:
  application:
    dependsOn:
      database:
        condition: started
      cache:
        condition: ready
```

All three formats are interchangeable. Mix them freely across different CRD entries in the same Katalog.

---

## Condition semantics

### `started`

The dependency's informer has performed its initial list and the worker pool is running. The dependency is not necessarily processing CRs successfully — only that it has started. This is the minimum signal and the default when no condition is specified.

Use `started` when the downstream CRD only needs the upstream CRD to exist and be registered.

### `ready`

The dependency has passed `started` and additionally reports `ready: true` on its health endpoint. This means at least one reconcile cycle has succeeded.

Use `ready` when the downstream CRD creates resources that depend on the upstream having successfully reconciled at least one CR.

### `healthy`

The dependency has passed `ready` and has no active degradation — zero consecutive reconcile failures, queue depth below the degrade threshold, no error rate spike.

Use `healthy` for the strictest ordering guarantee, typically in production environments where partial readiness is unacceptable.

| Condition | Meaning |
|---|---|
| `started` | Informer synced, workers launched |
| `ready` | At least one successful reconcile |
| `healthy` | No consecutive failures, no degradation |
