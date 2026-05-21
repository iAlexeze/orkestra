# 02 — Status Propagation

A `DataPipeline` creates a `Connector` child CR with `hasStatus: true`. After the Connector is reconciled, Orkestra reads back its `status.phase` and surfaces it in the DataPipeline's own status as `connectorPhase`.

This is the foundation of observable operator composition: parent operators can reflect the health of their children without any custom code.

---

## What This Shows

1. `DataPipeline` is created.
2. Orkestra creates `Connector` child CR with `hasStatus: true`.
3. The Connector operator reconciles the child and sets `status.phase = Running`.
4. On the next DataPipeline reconcile, Orkestra fetches the child's status and makes it available in `.children.custom`.
5. The DataPipeline status field `connectorPhase` is populated from `{{ (index .children.custom "my-pipeline-connector").status.phase }}`.

---

## New Concepts Introduced

### `hasStatus: true`

When set on a child CR block, Orkestra fetches the child resource's status after creation/update and stores it in a map accessible during status field template evaluation.

```yaml
onCreate:
  custom:
    - apiVersion: data.example.io/v1alpha1
      kind: Connector
      metadata:
        name: "{{ .metadata.name }}-connector"
        namespace: "{{ .metadata.namespace }}"
        namespaced: true
      spec:
        source: "{{ .spec.source }}"
        destination: "{{ .spec.destination }}"
        format: "{{ .spec.format }}"
      hasStatus: true   # <-- fetch and expose child status
```

| Value | Behaviour |
|---|---|
| `hasStatus: true` | Child status is fetched and available in `.children.custom` |
| `hasStatus: false` (default) | Child status is not fetched — faster, use when you don't need it |

### Referencing child status in parent status fields

The `.children.custom` map is keyed by the **resolved** child name (template expressions evaluated). Because the parent CR in this example is named `my-pipeline`, the child name `{{ .metadata.name }}-connector` resolves to `my-pipeline-connector`.

Use the `index` function for dynamic key lookup:

```yaml
status:
  fields:
    - path: connectorPhase
      value: '{{ (index .children.customs (printf "%s-connector" .metadata.name)).status.phase }}'
```

This pattern works regardless of what the parent CR is named at runtime.

---

## Prerequisites

- `kubectl` configured to a running cluster (Kind works)
- Ork CLI:
  ```bash
  curl get.orkestra.sh | bash
  ```

---

## Run the Example

### 1. Apply the CRDs

```bash
kubectl apply -f crd-datapipeline.yaml
kubectl apply -f crd-connector.yaml
```

### 2. Start the operator

```bash
ork run
```

### 3. Apply the DataPipeline CR

```bash
kubectl apply -f cr.yaml
```

### 4. Watch status propagation

```bash
kubectl get datapipelines -n default -w
```

Within a few reconcile cycles you will see `ConnectorPhase` change from empty to `Running` as the Connector operator updates its own status and the DataPipeline reconciler reads it back.

### 5. Inspect the full status

```bash
kubectl get datapipeline my-pipeline -n default -o yaml | grep -A 10 "^status:"
```

Expected:

```yaml
status:
  phase: Running
  pipelineStatus: Active
  connectorPhase: Running
```

### 6. Inspect the child Connector

```bash
kubectl get connector my-pipeline-connector -n default -o yaml | grep -A 10 "^status:"
```

```yaml
status:
  phase: Running
```

---

## What to Observe

- `connectorPhase` in the DataPipeline status mirrors `status.phase` from the Connector.
- If you manually patch the Connector's status to `Failed`, the next DataPipeline reconcile picks it up and reflects it in `connectorPhase: Failed`.
- Setting `hasStatus: false` on the child block would make `connectorPhase` evaluate to an empty string — no status read is performed.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
