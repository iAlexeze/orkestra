# operatorBox

Defines the reconciliation strategy for a CRD. Controls whether Orkestra uses the GenericReconciler with declarative templates, Go hooks, or a fully custom reconciler.

```yaml
operatorBox:
  default: true              # use GenericReconciler (true) or custom reconciler (false)

  finalizers:
    - example.io/cleanup

  # Declarative templates (default: true, dynamic mode)
  onCreate:
    ...
  onReconcile:
    ...
  onDelete:
    ...

  # Go hooks (default: true, typed or dynamic)
  hooks:
    location: github.com/example/operator
    function: DatabaseHooks
    alias: dbhooks
    resources:
      - statefulsets
      - services

  # Custom reconciler (default: false only)
  constructor:
    location: github.com/example/operator
    function: NewDatabaseReconciler
    alias: dbreconciler
    resources:
      - statefulsets
      - services

  status:
    ...               # → status.md

  when:
    ...               # → when-conditions.md

  rollBackOnError: false
  autoscale:
    ...
```

## `default`

| Value | Behaviour |
|-------|-----------|
| `true` (default) | GenericReconciler handles reconciliation. Use `onCreate`, `onReconcile`, `onDelete` for declarative templates, and `hooks` for Go hooks. |
| `false` | Fully custom reconciler. Set `constructor` to provide it. Templates and hooks are ignored. |

## `finalizers`

Per-CRD finalizers. Overrides `spec.finalizers` for this CRD only.

## `hooks`

A Go function invoked by the GenericReconciler. Implements typed reconcile hooks (`OnCreate`, `OnUpdate`, `OnDelete`). Used when you need Go logic that the GenericReconciler calls instead of declarative templates.

```yaml
hooks:
  location: github.com/example/operator   # Go module path
  function: DatabaseHooks                  # exported function name
  alias: dbhooks                           # import alias (auto-derived if omitted)
  resources:                              # RBAC verbs claimed for this hook
    - statefulsets
    - services
    - cronjobs
```

Requires typed mode (`apiTypes.location` set) and `ork generate registry`.

## `constructor`

Replaces the GenericReconciler entirely. Requires `default: false`.

```yaml
operatorBox:
  default: false
  constructor:
    location: github.com/example/operator
    function: NewDatabaseReconciler
    alias: dbreconciler
    resources:
      - statefulsets
      - services
```

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
