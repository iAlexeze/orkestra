# status

Orkestra writes status in two layers — one automatic, one declarative.

**Layer 1 — automatic (always on).** After every reconcile Orkestra patches:
- `status.conditions[type=Ready]` — `True` on success, `False` on error
- `status.observedGeneration` — mirrors `metadata.generation`

No declaration required. Every managed CR gets this.

**Layer 2 — declarative fields.** Declared under `operatorBox.status`. Written after reconcile templates complete.

## Wire format

```yaml
operatorBox:
  status:
    conditions: true    # optional — default true

    fields:
      - path: phase
        value: "Running"

      - path: replicas
        type: int
        value: "{{ toInt .spec.replicas }}"

      - path: database.host    # dot-notation → status.database.host
        value: "{{ .spec.host }}"

      - path: endpoint
        value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

      - path: phase
        value: "Pending"
        when:
          - field: status.phase
            operator: notExists
```

## `conditions`

| Value | Description |
|-------|-------------|
| `true` (default) | Write the standard `Ready` condition and `observedGeneration` after every reconcile. |
| `false` | Opt out. Use only when the CRD's status schema forbids a `conditions` field, or you manage conditions entirely in Go hooks. |

## `fields`

A **list** of field declarations. Resolved in declaration order — later entries win on path conflict.

```yaml
fields:
  - path: <dot.notation.path>     # required
    value: "<string or template>"  # required
    type: string                   # optional
    clearOnFalse: false            # optional
    when:                          # optional — AND conditions
      - ...
    anyOf:                         # optional — OR conditions
      - ...
```

### Field properties

| Field | Required | Description |
|-------|----------|-------------|
| `path` | yes | Dot-notation path relative to `status`. `"phase"` → `status.phase`. `"db.host"` → `status.db.host`. |
| `value` | yes | Value to write. Supports Go template expressions. Static strings skip parsing. |
| `type` | no | Cast the resolved value before writing. Defaults to `string`. |
| `when` | no | List of conditions — **all must pass** (AND). Field is skipped if any fails. |
| `anyOf` | no | List of conditions — **at least one must pass** (OR). |
| `clearOnFalse` | no | When `true` and the `when:`/`anyOf:` condition evaluates to false, write `""` to the field instead of leaving the previous value. Use for transient fields (crash reasons, warning messages) that should disappear when the triggering condition clears. No effect when no conditions are declared. |

When both `when` and `anyOf` are declared, both blocks must pass.

### `type` values

| Value | Written as |
|-------|------------|
| `string` / `str` / `""` (default) | string |
| `int` / `integer` | integer |
| `float` | float64 |
| `bool` / `boolean` | boolean |
| `auto` | inferred from the resolved value |

### Template variables

Values are Go templates evaluated against the full CR object map:

| Expression | Description |
|------------|-------------|
| `{{ .metadata.name }}` | CR name |
| `{{ .metadata.namespace }}` | CR namespace |
| `{{ .spec.* }}` | Any spec field |
| `{{ .status.* }}` | Existing status fields |
| `{{ .children.deployment }}` | Child Deployment status map |
| `{{ .children.statefulset }}` | Child StatefulSet status map |
| `{{ .children.service }}` | Child Service status map |
| `{{ .children.job }}` | Child Job status map |
| `{{ .children.custom }}` | Child Custom Resource status map |

### Kubernetes helper functions

Functions from the Orkestra note library useful in status values:

**Replicas**

| Function | Returns | Description |
|----------|---------|-------------|
| `allReplicasReady .children.deployment` | bool | `true` when desired == ready |
| `readyReplicas .children.deployment` | int | Number of ready replicas |
| `desiredReplicas .children.deployment` | int | Desired replica count |
| `availableReplicas .children.deployment` | int | Available (not just ready) replicas |
| `updatedReplicas .children.deployment` | int | Replicas on current pod template |
| `rolloutComplete .children.deployment` | bool | `true` when rollout is fully done |

**Services**

| Function | Returns | Description |
|----------|---------|-------------|
| `serviceClusterIP .children.service` | string | Cluster IP of the service |
| `serviceNodePort .children.service "portName"` | int | NodePort for the named port |
| `serviceLoadBalancerIP .children.service` | string | Load balancer IP |
| `serviceLoadBalancerHost .children.service` | string | Load balancer hostname |
| `endpointsReady .children.endpointslice` | bool | `true` when endpoints are ready |

**Jobs**

| Function | Returns | Description |
|----------|---------|-------------|
| `jobSucceeded .children.job` | bool | `true` when job completed successfully |
| `jobFailed .children.job` | bool | `true` when job failed |

**Type casting**

| Function | Description |
|----------|-------------|
| `toInt .spec.replicas` | Cast to integer |
| `toFloat .spec.ratio` | Cast to float |
| `toBool .spec.enabled` | Cast to bool |
| `toString .spec.count` | Cast to string |

### `when` and `anyOf` conditions

Each condition targets a dot-notation field path and applies an operator.

```yaml
when:
  - field: status.phase      # required — dot-path into the CR object
    equals: "Running"        # shorthand operators (see below)

  - field: spec.replicas
    operator: gt             # explicit operator form
    value: "0"
```

**Shorthand fields** (instead of `operator` + `value`):

| Shorthand | Equivalent |
|-----------|-----------|
| `equals: "v"` | `operator: equals, value: "v"` |
| `notEquals: "v"` | `operator: notEquals, value: "v"` |
| `contains: "v"` | `operator: contains, value: "v"` |
| `prefix: "v"` | `operator: prefix, value: "v"` |
| `suffix: "v"` | `operator: suffix, value: "v"` |
| `greaterThan: "v"` | `operator: gt, value: "v"` |
| `lessThan: "v"` | `operator: lt, value: "v"` |
| `exists: true` | `operator: exists` |
| `notExists: true` | `operator: notExists` |

**All operators:**

| Operator | Description |
|----------|-------------|
| `equals` | Field exactly equals value (string comparison). |
| `notEquals` | Field does not equal value. |
| `contains` | Field contains value as a substring. |
| `prefix` | Field starts with value. |
| `suffix` | Field ends with value. |
| `exists` | Field is present and non-empty. |
| `notExists` | Field is absent or empty. Use for first-reconcile detection. |
| `gt` | Field is numerically greater than value. Absent field treated as 0. |
| `lt` | Field is numerically less than value. |
| `in` | Field is one of a comma-separated list: `value: "Pending,Running"`. |
| `typeOf` | Field's runtime type equals value (`map`, `slice`, `string`, `number`, `bool`, `null`). |
| `typeMap` | Field is a YAML object. |
| `typeList` | Field is a YAML array. |
| `typeString` | Field is a string. |
| `typeNumber` | Field is a number. |
| `typeBool` | Field is a boolean. |
| `typeNull` | Field is null or missing. |

## Example: declarative state machine

```yaml
status:
  fields:
    - path: phase
      value: "Pending"
      when:
        - field: status.phase
          operator: notExists

    - path: phase
      value: "Running"
      when:
        - field: status.phase
          equals: "Pending"
        - field: children.deployment.status.readyReplicas
          operator: gt
          value: "0"

    - path: phase
      value: "Succeeded"
      when:
        - field: status.phase
          equals: "Running"
        - field: children.job.status.succeeded
          operator: gt
          value: "0"

    - path: ready
      type: bool
      value: "{{ allReplicasReady .children.deployment }}"

    - path: endpoint
      value: "{{ serviceLoadBalancerHost .children.service }}"
```

---
