# 21 — PVC Notes

PVC notes read PersistentVolumeClaim phase, capacity, storage class, and access modes. Enriched notes navigate the bound PV (via `enrich: [pvc]`) or the StorageClass object (via `enrich: [storageclass]`).

---

## Reference

### `pvcBound`

Returns `true` when `status.phase == "Bound"`.

```yaml
when:
  - field: "{{ pvcBound .children.pvc }}"
    equals: "true"
```

---

### `pvcPhase`

Returns `status.phase` — `"Bound"`, `"Pending"`, `"Released"`, or `"Lost"`.

```yaml
- path: pvcPhase
  value: "{{ pvcPhase .children.pvc }}"
# → "Bound"
```

---

### `pvcCapacity`

Returns `spec.resources.requests.storage`.

```yaml
- path: capacity
  value: "{{ pvcCapacity .children.pvc }}"
# → "10Gi"
```

---

### `pvcStorageClass`

Returns `spec.storageClassName`, or `""` when not set (cluster default applies).

```yaml
- path: storageClass
  value: "{{ pvcStorageClass .children.pvc }}"
# → "standard"
```

---

### `pvcAccessModes`

Returns a comma-separated list of `spec.accessModes`.

```yaml
- path: accessModes
  value: "{{ pvcAccessModes .children.pvc }}"
# → "ReadWriteOnce"
```

---

### `pvcProvisioner`

Returns the provisioner annotation from the bound PV (`pv.kubernetes.io/provisioned-by`). Requires `enrich: [pvc]`.

```yaml
- path: provisioner
  value: "{{ pvcProvisioner .children.pvc }}"
# → "ebs.csi.aws.com"
```

---

### `pvcVolumeMode`

Returns the `volumeMode` of the bound PV — `"Filesystem"` or `"Block"`. Requires `enrich: [pvc]`.

```yaml
- path: volumeMode
  value: "{{ pvcVolumeMode .children.pvc }}"
# → "Filesystem"
```

---

### `pvcStorageClassProvisioner`

Returns `_storageClass.provisioner` — the provisioner declared on the StorageClass itself. Requires `enrich: [storageclass]`.

```yaml
- path: scProvisioner
  value: "{{ pvcStorageClassProvisioner .children.pvc }}"
# → "ebs.csi.aws.com"
```

---

### `pvcStorageClassReclaimPolicy`

Returns `_storageClass.spec.reclaimPolicy` — `"Delete"` or `"Retain"`. Requires `enrich: [storageclass]`.

```yaml
- path: reclaimPolicy
  value: "{{ pvcStorageClassReclaimPolicy .children.pvc }}"
# → "Delete"
```

---

## Quick reference

| Note | Signature | Returns | Enrichment |
|------|-----------|---------|------------|
| `pvcBound` | `(obj any)` | `bool` | none |
| `pvcPhase` | `(obj any)` | `string` | none |
| `pvcCapacity` | `(obj any)` | `string` | none |
| `pvcStorageClass` | `(obj any)` | `string` | none |
| `pvcAccessModes` | `(obj any)` | `string` | none |
| `pvcProvisioner` | `(obj any)` | `string` | `enrich: [pvc]` |
| `pvcVolumeMode` | `(obj any)` | `string` | `enrich: [pvc]` |
| `pvcStorageClassProvisioner` | `(obj any)` | `string` | `enrich: [storageclass]` |
| `pvcStorageClassReclaimPolicy` | `(obj any)` | `string` | `enrich: [storageclass]` |

---

**Next →** [22 — Node Notes](22-node.md)
