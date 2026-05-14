# 05 — forEach Sharding

A `ShardedStore` CRD creates N `Shard` child CRs using `forEach` over `spec.shards`. Updating the list — adding or removing entries — causes corresponding child CRs to be created or deleted on the next reconcile cycle.

---

## What This Shows

- A single `ShardedStore` CR with 3 shards creates 3 `Shard` child CRs.
- Each Shard operator then creates its own Deployment for local storage.
- Adding a fourth shard entry to the `ShardedStore` spec creates a new `Shard` CR without touching existing ones.
- Removing a shard entry from the list causes Orkestra to delete the corresponding `Shard` CR.

---

## New Concepts Introduced

### `forEach:` on child CR blocks

`forEach` turns one child CR block into a template that is instantiated once per element in a list field of the parent spec.

```yaml
onCreate:
  custom:
    - apiVersion: storage.example.io/v1alpha1
      kind: Shard
      forEach:
        field: spec.shards   # the list field to iterate over
        as: shard            # local variable name for the current element
      metadata:
        name: "{{ .metadata.name }}-{{ .shard.name }}"
        namespace: "{{ .metadata.namespace }}"
        namespaced: true
      spec:
        shardName: "{{ .shard.name }}"
        region: "{{ .shard.region }}"
        size: "{{ .shard.size }}"
        replicationFactor: "{{ .spec.replicationFactor }}"
      hasStatus: false
```

Inside the block, template expressions have access to:

| Variable | Source |
|---|---|
| `.metadata`, `.spec` | The parent ShardedStore CR |
| `.shard` | The current iteration element from `spec.shards` |

So for a parent named `global-store` with a shard `{ name: shard-eu, region: eu-west-1, size: 500Gi }`, the resolved child name is `global-store-shard-eu`.

### Dynamic topology from list fields

`forEach` allows the operator to represent dynamic, variable-length topologies in pure YAML. There is no code to write — just declare the shape of each child and let Orkestra fan it out:

- 1 entry in `spec.shards` → 1 Shard CR
- 100 entries → 100 Shard CRs
- Remove an entry → that Shard CR is deleted

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
kubectl apply -f crd-shardedstore.yaml
kubectl apply -f crd-shard.yaml
```

### 2. Start the operator

```bash
ork run -f katalog.yaml 
```

### 3. Apply the ShardedStore CR

```bash
kubectl apply -f cr.yaml
```

### 4. Watch the fan-out

```bash
kubectl get shardedstores,shards -n default
```

Expected output (three Shard CRs created from one ShardedStore):

```
NAME                                         SHARDS   REPLICATIONFACTOR   PHASE   AGE
shardedstore.storage.example.io/global-store   3        2                   Ready   10s

NAME                                         SHARDNAME   REGION           SIZE    PHASE   AGE
shard.storage.example.io/global-store-shard-eu   shard-eu    eu-west-1        500Gi   Ready   8s
shard.storage.example.io/global-store-shard-us   shard-us    us-east-1        500Gi   Ready   8s
shard.storage.example.io/global-store-shard-ap   shard-ap    ap-southeast-1   250Gi   Ready   8s
```

Also check Deployments — each Shard created its own:

```bash
kubectl get deployments -n default
```

### 5. Add a new shard

```bash
kubectl patch shardedstore global-store -n default --type=merge -p '{
  "spec": {
    "shards": [
      {"name": "shard-eu", "region": "eu-west-1", "size": "500Gi"},
      {"name": "shard-us", "region": "us-east-1", "size": "500Gi"},
      {"name": "shard-ap", "region": "ap-southeast-1", "size": "250Gi"},
      {"name": "shard-sa", "region": "sa-east-1", "size": "100Gi"}
    ]
  }
}'
```

Watch the new `global-store-shard-sa` appear within the resync window:

```bash
kubectl get shards -n default -w
```

### 6. Remove a shard

```bash
kubectl patch shardedstore global-store -n default --type=merge -p '{
  "spec": {
    "shards": [
      {"name": "shard-eu", "region": "eu-west-1", "size": "500Gi"},
      {"name": "shard-us", "region": "us-east-1", "size": "500Gi"}
    ]
  }
}'
```

The `global-store-shard-ap` and `global-store-shard-sa` Shard CRs are deleted.

---

## What to Observe

- The number of `Shard` CRs always equals the number of entries in `spec.shards`.
- Each Shard is independently reconciled by its own operator goroutine.
- All Shards have owner references pointing to the `global-store` ShardedStore.
- Deleting the ShardedStore deletes all three Shards (and their Deployments) via cascading garbage collection.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
