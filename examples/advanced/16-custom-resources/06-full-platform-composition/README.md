# 06 — Full Platform Composition

A `Platform` CRD uses a **motif** to provision a complete infrastructure stack — `MessageQueue`, `ObjectStore`, and `SearchCluster` — from a single CR. The motif packages the three-component composition so it can be imported and reused across any number of Katalogs without duplicating the child CR declarations.

This is the capstone example: motifs + custom resources + status propagation, all working together.

---

## What This Shows

- One `Platform` CR creates three child CRs: `MessageQueue`, `ObjectStore`, `SearchCluster`.
- The child topology is defined in `platform-motif.yaml`, not in `katalog.yaml`.
- Status from all three children is propagated back to the Platform.
- Two Platform CRs (`dev-platform`, `prod-platform`) share the same motif. A production Katalog would supply different input values for larger sizing.
- All child CRs are garbage-collected when the Platform CR is deleted.

---

## New Concepts Introduced

### Motifs with custom resources

A motif is a reusable template fragment. It can contain `resources.custom` blocks just like `operatorBox.onCreate.custom`. The motif receives inputs from the importing Katalog and resolves them inside the child CR specs.

```yaml
# platform-motif.yaml (excerpt)
inputs:
  - name: mqReplicas
    default: 1
    type: integer
  - name: searchNodes
    default: 1
    type: integer

resources:
  custom:
    - apiVersion: infra.example.io/v1alpha1
      kind: MessageQueue
      metadata:
        name: "{{ .metadata.name }}-mq"
        namespace: "{{ .metadata.namespace }}"
        namespaced: true
      spec:
        replicas: "{{ .inputs.mqReplicas }}"
        storage: "{{ .inputs.mqStorage }}"
      hasStatus: true
```

The importing Katalog supplies concrete values via the `with:` block:

```yaml
# katalog.yaml (excerpt)
imports:
  - motif: ./platform-motif.yaml
    with:
      mqReplicas: 1
      mqStorage: "20Gi"
      storeCapacity: "200Gi"
      searchNodes: 1
      searchHeapSize: "1g"
```

### Why motifs matter for custom resources

Without motifs, every Katalog that wants to provision a MessageQueue + ObjectStore + SearchCluster would need to copy-paste the three child CR blocks. With a motif:

- The composition is defined once and versioned independently.
- Inputs abstract the sizing differences — one motif for all tiers.
- Teams can publish motifs to the Orkestra registry and others can import them by URL.

### Multi-child status propagation

When all three children have `hasStatus: true`, the Platform status block can aggregate all of them:

```yaml
status:
  fields:
    - path: mqPhase
      value: '{{ (index .children.custom (printf "%s-mq" .metadata.name)).status.phase }}'
    - path: storePhase
      value: '{{ (index .children.custom (printf "%s-store" .metadata.name)).status.phase }}'
    - path: searchPhase
      value: '{{ (index .children.custom (printf "%s-search" .metadata.name)).status.phase }}'
```

A single `kubectl get platform dev-platform` tells you the health of the entire stack.

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
kubectl apply -f crd-platform.yaml
kubectl apply -f crd-messagequeue.yaml
kubectl apply -f crd-objectstore.yaml
kubectl apply -f crd-searchcluster.yaml
```

### 2. Start the operator

```bash
ork run -f katalog.yaml 
```

### 3. Apply the small platform

```bash
kubectl apply -f cr-small.yaml
```

Watch all three child CRs appear:

```bash
kubectl get platforms,messagequeues,objectstores,searchclusters -n default
```

Expected:

```
NAME                                    SIZE    OWNER          PHASE   AGE
platform.infra.example.io/dev-platform   small   platform-eng   Ready   12s

NAME                                           REPLICAS   STORAGE   PHASE     AGE
messagequeue.infra.example.io/dev-platform-mq   1          20Gi      Running   10s

NAME                                           CAPACITY   REPLICAS   PHASE     AGE
objectstore.infra.example.io/dev-platform-store 200Gi      1          Running   10s

NAME                                             NODES   HEAPSIZE   PHASE     AGE
searchcluster.infra.example.io/dev-platform-search 1       1g         Running   10s
```

### 4. Check the aggregated Platform status

```bash
kubectl get platform dev-platform -n default -o yaml | grep -A 10 "^status:"
```

Expected:

```yaml
status:
  phase: Ready
  mqPhase: Running
  storePhase: Running
  searchPhase: Running
```

### 5. Apply the large platform alongside

```bash
kubectl apply -f cr-large.yaml
```

A second independent stack (`prod-platform-mq`, `prod-platform-store`, `prod-platform-search`) appears without touching the `dev-platform` stack.

### 6. Delete one platform

```bash
kubectl delete platform dev-platform -n default
```

Only the `dev-platform-*` children are removed. The `prod-platform-*` children are unaffected:

```bash
kubectl get messagequeues,objectstores,searchclusters -n default
# Only prod-platform-* resources remain
```

---

## What to Observe

- A single two-line CR (`cr-small.yaml`) provisions three independent operators, each with their own Deployment and Service.
- The motif is the single source of truth for the stack topology. Changing `platform-motif.yaml` updates all future Platform CRs that import it.
- Status aggregation gives a single-pane-of-glass view of the entire stack health on the parent CR.
- Multiple Platform CRs run entirely independently; they share the operator binary but have isolated reconcile loops.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
