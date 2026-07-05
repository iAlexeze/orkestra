# operatorBox

Defines the reconciliation strategy and lifecycle configuration for a CRD. Controls which reconciler implementation runs and how resources, status, admission, autoscaling, and rollback behave.

```yaml
operatorBox:
  # reconciler: determines which reconciler implementation runs.
  # Omit entirely for declarative-only CRDs (GenericReconciler is the default).
  reconciler:
    default: true              # true → GenericReconciler | false → custom constructor

    # Go hooks (default: true, typed mode)
    hooks:
      location: github.com/example/operator
      function: DatabaseHooks
      alias: dbhooks
      resources:
        - kind: StatefulSet
        - kind: Service

    # Custom reconciler (default: false)
    constructor:
      location: github.com/example/operator
      function: NewDatabaseReconciler
      alias: dbreconciler
      resources:
        - kind: StatefulSet
        - kind: Service

  finalizers:
    - example.io/cleanup

  # Declarative templates (GenericReconciler only)
  onCreate:
    ...
  onReconcile:
    ...
  onDelete:
    ...

  status:
    ...               # → status.md

  when:
    ...               # → when-conditions.md

  rollBackOnError: false
  autoscale:
    ...
```

## `reconciler`

Groups the reconciler identity fields. Omit for declarative-only CRDs — GenericReconciler is the default.

### `reconciler.default`

| Value | Behaviour |
|-------|-----------|
| `true` (default) | GenericReconciler handles reconciliation. Use `onCreate`, `onReconcile`, `onDelete` for declarative templates, and `reconciler.hooks` for Go hooks. |
| `false` | Fully custom reconciler. Set `reconciler.constructor` to provide it. Templates and hooks are ignored. |

### `reconciler.hooks`

A Go function invoked by the GenericReconciler. Implements typed reconcile hooks (`OnCreate`, `OnUpdate`, `OnDelete`). Used when you need Go logic that the GenericReconciler calls instead of declarative templates.

```yaml
operatorBox:
  reconciler:
    hooks:
      location: github.com/example/operator   # Go module path
      function: DatabaseHooks                  # exported function name
      alias: dbhooks                           # import alias (auto-derived if omitted)
      runHooksFirst: false                     # see below
      resources:                               # RBAC verbs claimed for this hook
        - kind: StatefulSet
        - kind: Service
        - kind: CronJob
```

Requires typed mode (`apiTypes.location` set) and `ork generate registry`.

#### `reconciler.hooks.runHooksFirst`

Controls the order in which the hook and declared templates run within the same reconcile cycle.

| Value | Order |
|-------|-------|
| `false` (default) | Declared templates run first, then the hook. Use when the hook is additive — the ServiceAccount or Deployment already exists when the hook runs. |
| `true` | Hook runs first, then declared templates. Use when the hook creates resources that declared templates depend on. |

```yaml
reconciler:
  hooks:
    runHooksFirst: true   # hook → then declared templates
                          # false (default): declared templates → then hook
```

### `reconciler.constructor`

Replaces the GenericReconciler entirely. Requires `reconciler.default: false`.

```yaml
operatorBox:
  reconciler:
    default: false
    constructor:
      location: github.com/example/operator
      function: NewDatabaseReconciler
      alias: dbreconciler
      resources:
        - kind: StatefulSet
        - kind: Service
```

## `finalizers`

Per-CRD finalizers. Overrides `spec.finalizers` for this CRD only.

## `onCreate` / `onReconcile` / `onDelete`

Declarative resource templates evaluated during reconcile phases:

| Hook | When it runs |
|------|-------------|
| `onCreate` | CR transitions from Pending → Active (first reconcile) |
| `onReconcile` | Every reconcile cycle — drift correction |
| `onDelete` | Before finalizer is removed — cleanup |

```yaml
onCreate:
  deployments:
    - name: "{{ .Name }}-server"
      image: postgres:14
      env:
        - name: POSTGRES_DB
          value: "{{ .Spec.Database }}"
  services:
    - name: "{{ .Name }}-svc"
      port: 5432

onDelete:
  jobs:
    - name: "{{ .Name }}-cleanup"
      image: postgres:14
      command: ["./cleanup.sh"]
```

Available resource types: `deployments`, `services`, `configmaps`, `secrets`, `jobs`, `cronjobs`, `statefulsets`, `ingresses`, `serviceaccounts`, `roles`, `rolebindings`, `pvcs`, `pdbs`, `hpas`, `namespaces`.

Templates are Go templates evaluated against the CR object. Use `{{ .Name }}`, `{{ .Namespace }}`, `{{ .Spec.* }}`, `{{ .Status.* }}`.

## `rollBackOnError`

Zero-config rollback on reconcile failure. Restores the previous known-good state when a reconcile cycle errors.

```yaml
rollBackOnError: true
```

## `autoscale`

Dynamically adjusts worker count, queue depth, and resync interval based on conditions.

```yaml
autoscale:
  interval: 15s
  cooldown: 2m
  conditions:
    when:
      - field: status.queueDepth
        operator: gt
        value: "100"
        valueType: int
  do:
    workers: 5
    queueDepth: 500
    resync: 10s
```

| Field | Description |
|-------|-------------|
| `interval` | Evaluation frequency (default: `15s`) |
| `cooldown` | Min time conditions must be false before restoring baseline (default: `2m`) |
| `conditions.when` | AND conditions — all must be true |
| `conditions.anyOf` | OR conditions — at least one must be true |
| `do.workers` | Override concurrent goroutines when conditions are met |
| `do.queueDepth` | Override max queue depth |
| `do.resync` | Override resync interval |

---
