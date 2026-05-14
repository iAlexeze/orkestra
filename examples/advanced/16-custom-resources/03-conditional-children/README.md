# 03 — Conditional Children

An `AppEnvironment` CRD creates optional `CacheCluster` and `SearchIndex` child CRs based on feature flags in its spec. Dev environments run lean; prod environments enable everything.

This example shows that a single CRD can represent radically different topologies — without branching logic in Go code.

---

## What This Shows

- `cr-dev.yaml` creates an AppEnvironment with `cache.enabled: "false"` and `search.enabled: "false"`. No child CRs are created.
- `cr-prod.yaml` creates an AppEnvironment with both flags set to `"true"`. Both `CacheCluster` and `SearchIndex` children are created.
- Updating a dev environment to enable cache causes Orkestra to create the `CacheCluster` on the next reconcile — without any restart.

---

## New Concepts Introduced

### `when:` conditions on child CR blocks

A `when:` list gates whether a child CR is created. All conditions must be satisfied for the child to be provisioned.

```yaml
onCreate:
  custom:
    - apiVersion: platform.example.io/v1alpha1
      kind: CacheCluster
      when:
        - field: spec.cache.enabled
          equals: "true"
      metadata:
        name: "{{ .metadata.name }}-cache"
        namespace: "{{ .metadata.namespace }}"
        namespaced: true
      spec:
        size: "{{ .spec.cache.size }}"
        ttl: "{{ .spec.cache.ttl }}"
      hasStatus: false
```

| Field | Meaning |
|---|---|
| `when[].field` | Dot-path into the parent CR's spec/metadata |
| `when[].equals` | String value to match |

When the condition is false, the child CR is not created and is not deleted if it previously existed. When the condition becomes true, the child is created on the next reconcile cycle.

Multiple conditions in the list are ANDed together:

```yaml
when:
  - field: spec.tier
    equals: "prod"
  - field: spec.cache.enabled
    equals: "true"
```

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
kubectl apply -f crd-appenvironment.yaml
kubectl apply -f crd-cachecluster.yaml
kubectl apply -f crd-searchindex.yaml
```

### 2. Start the operator

```bash
ork run -f katalog.yaml 
```

### 3. Apply the dev environment (no children)

```bash
kubectl apply -f cr-dev.yaml
```

Check: no CacheCluster or SearchIndex should exist:

```bash
kubectl get appenvironments,cacheclusters,searchindices -n default
```

Expected:

```
NAME                                  TIER   CACHE   SEARCH   PHASE   AGE
appenvironment.platform.example.io/dev-env   dev    false   false    Ready   5s

No resources found for cacheclusters.
No resources found for searchindices.
```

### 4. Apply the prod environment (both children)

```bash
kubectl apply -f cr-prod.yaml
```

Check: both child CRs should appear:

```bash
kubectl get appenvironments,cacheclusters,searchindices -n default
```

Expected:

```
NAME         TIER   CACHE   SEARCH   PHASE
dev-env      dev    false   false    Ready
prod-env     prod   true    true     Ready

NAME                   SIZE    TTL    PHASE
prod-env-cache         large   7200   Running

NAME                   INDEXNAME         REPLICAS   PHASE
prod-env-search        prod-main-index   3          Running
```

### 5. Promote dev to enable cache

Edit the dev AppEnvironment to flip the cache flag:

```bash
kubectl patch appenvironment dev-env -n default --type=merge \
  -p '{"spec":{"cache":{"enabled":"true","size":"small","ttl":3600}}}'
```

Within the resync window (60s), Orkestra creates `dev-env-cache`:

```bash
kubectl get cachecluster dev-env-cache -n default
```

---

## What to Observe

- When `when:` evaluates to false, the child block is silently skipped — no error, no partial creation.
- When `when:` becomes true after a spec update, the child is created on the next reconcile without restarting the operator.
- Each child has an owner reference pointing to its parent AppEnvironment, so deleting the parent removes all children.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
