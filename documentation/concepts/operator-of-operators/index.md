# Operator of Operators

The `custom:` block in an operatorBox creates Custom Resources — not Kubernetes primitives. This means one Orkestra operator can instantiate resources that are themselves managed by other Orkestra operators in the same Katalog.

---

## The pattern

```yaml
operatorBox:
  onCreate:
    custom:
      - apiVersion: platform.io/v1alpha1
        kind: CacheCluster
        when:
          - field: spec.cache.enabled
            equals: "true"
        metadata:
          name: "{{ .metadata.name }}-cache"
        spec:
          size: "{{ .spec.cache.size }}"
        hasStatus: false
        reconcile: true
```

The parent operator creates a `CacheCluster` CR when caching is enabled. The `cachecluster` reconciler in the same Katalog watches `CacheCluster` CRs and creates the actual Redis deployment. One spec change on the parent CR activates or deactivates an entire sub-operator.

---

## What `custom:` supports

Every feature available to other resource types is available to `custom:`:

- `when:` — conditional creation
- `anyOf:` — OR conditions
- `forEach:` — expand to multiple CRs from a list
- `reconcile: true` — drift correction
- `sleep:` — testing and chaos engineering
- `hasStatus:` — hint to skip status writes for CRDs without the status subresource

---

## `hasStatus`

Most CRDs expose a status subresource. Some do not. The `hasStatus` field controls whether Orkestra attempts status writes:

```yaml
hasStatus: false   # never write status — avoids API errors on CRDs without subresource
hasStatus: true    # always write status
# omit             # auto-detect via discovery
```

---

## Owner references

Every child CR created by `custom:` gets the parent CR as its owner reference. When the parent is deleted, Kubernetes garbage-collects all child CRs. Their operators then clean up their own resources.

---

## Naming convention

Prefix child CR names with the parent's name to keep them identifiable and avoid conflicts when multiple parent CRs exist in the same namespace:

```yaml
metadata:
  name: "{{ .metadata.name }}-cache"
```
