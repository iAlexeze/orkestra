---
title: "Katalog Schema Status"
weight: 96
---

# Katalog Schema — Status Block

---

## `reconciler.status`

Declares how Orkestra manages the CR's `/status` subresource after each
reconcile cycle.

{{< callout type="note" title="CRD prerequisite" >}}
Status management requires `subresources: status: {}` in the CRD definition.
Without it, Orkestra's status patches are silently ignored by the API server.
Reconciliation is not affected — the CRD simply shows no status.
{{< /callout >}}

```yaml
operatorBox:
  status:
    conditions: bool          # default: true
    fields:
      - path: string          # required — dot-notation, relative to status
        value: string         # required — supports template expressions
```

### `reconciler.status.conditions`

**Type:** `bool` | **Default:** `true`

When true (the default), Orkestra writes a standard Kubernetes `Ready`
condition and `observedGeneration` to the CR's status after every reconcile,
regardless of outcome:

```yaml
status:
  conditions:
    - type: Ready
      status: "True" | "False"
      reason: ReconcileSucceeded | ReconcileError
      message: "" | "<error details>"
      lastTransitionTime: "<RFC3339>"
      observedGeneration: <generation>
  observedGeneration: <generation>
```

Set to `false` to opt out. Only needed when the CRD schema explicitly
forbids a `conditions` field, or when conditions are managed entirely by
Go hooks.

### `reconciler.status.fields[]`

**Type:** `[]StatusFieldSpec` | **Required:** no | **Default:** `[]`

A list of status fields to write after every successful reconcile. Fields
are not written on reconcile failure — only the `Ready` condition is updated
on failure. This prevents misleading status when the cluster state is partial.

| Field | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | Dot-notation path relative to `status`. `phase` → `status.phase`. `database.host` → `status.database.host`. |
| `value` | string | yes | The value to write. Supports Go template expressions evaluated against the CR. Static strings are returned as-is. |

**Template context:**

The full CR object is available — same context as `onCreate` templates:

```
{{ .metadata.name }}        CR name
{{ .metadata.namespace }}   CR namespace
{{ .spec.<field> }}         any spec field (dynamic mode)
{{ .children.deployment }}  first Deployment created by this CR (Layer 3)
{{ .children.service }}     first Service created by this CR (Layer 3)
```

**Child resource access (Layer 3):**

After onCreate/onReconcile templates create child resources, Orkestra reads
them back and makes their status available under `.children`:

```yaml
fields:
  # Singular shorthand — first/only resource of this type
  - path: readyReplicas
    value: "{{ .children.deployment.status.readyReplicas }}"

  # Plural — multiple resources, accessed by Kubernetes name
  - path: apiReadyReplicas
    value: "{{ (index .children.deployments \"my-site-api\").status.readyReplicas }}"
```

Available child keys:

| Singular | Plural | Resource |
|---|---|---|
| `.children.deployment` | `.children.deployments` | Deployment |
| `.children.service` | `.children.services` | Service |
| `.children.secret` | `.children.secrets` | Secret |
| `.children.configMap` | `.children.configMaps` | ConfigMap |
| `.children.job` | `.children.jobs` | Job |
| `.children.cronJob` | `.children.cronJobs` | CronJob |
| `.children.pod` | `.children.pods` | Pod |
| `.children.serviceAccount` | `.children.serviceAccounts` | ServiceAccount |

Missing child status fields resolve to `""` — expected on the first reconcile
before Kubernetes has populated child resource status.

---

## Full status example

```yaml
- name: website
  apiTypes:
    group: demo.orkestra.io
    version: v1alpha1
    kind: Website
    plural: websites

  operatorBox:
    default: true

    status:
      # conditions: true ← default
      fields:
        # Layer 2 — from the CR spec and metadata
        - path: phase
          value: "Running"
        - path: observedReplicas
          value: "{{ .spec.replicas }}"
        - path: endpoint
          value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

        # Layer 3 — from child resource status
        - path: readyReplicas
          value: "{{ .children.deployment.status.readyReplicas }}"
        - path: availableReplicas
          value: "{{ .children.deployment.status.availableReplicas }}"

        # Nested status — becomes status.database.host
        - path: database.host
          value: "{{ .spec.host }}"
        - path: database.port
          value: "{{ .spec.port }}"

    onCreate:
      deployments:
        - image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          reconcile: true
```

---

## Error reference

### Status errors

```
warning: status: failed to build resolver — declarative fields skipped
```
The resolver could not be built from the CR object. Should not occur in
normal operation. Check that the CR is valid unstructured JSON.

```
warning: status: some field expressions failed to resolve
  status.readyReplicas: executing "f": ...
```
A template expression in a status field failed to evaluate. The field is
skipped; other fields proceed. Check the expression syntax and the CR fields
it references.

```
debug: status: patch failed — CRD may not have status subresource declared
```
The CRD does not declare `subresources: status: {}`. Add it to the CRD
definition and reapply. The reconcile is not affected by this error.
```
