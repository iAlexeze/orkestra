# 13 — Job Notes

Job notes read batch Job lifecycle state. They gate dependent resources on Job completion, failure, or active execution — the most common pattern in multi-step workflows where a migration or init job must finish before the main workload starts.

---

## Reference

### `jobSucceeded`

Return `true` when `status.succeeded > 0` — at least one pod has completed successfully.

```yaml
# Gate the main deployment on a migration job:
when:
  - field: "{{ jobSucceeded .children.migrationjob }}"
    equals: "true"
```

---

### `jobFailed`

Return `true` when `status.failed > 0` — at least one pod has failed (and not been retried to success).

```yaml
# Require both: not failed AND succeeded before proceeding:
when:
  - field: "{{ jobFailed .children.migrationjob }}"
    equals: "false"
  - field: "{{ jobSucceeded .children.migrationjob }}"
    equals: "true"
```

---

### `jobActive`

Return `true` when `status.active > 0` — at least one pod is currently running.

```yaml
# Block cleanup until the job is no longer running:
when:
  - field: "{{ jobActive .children.cleanupjob }}"
    equals: "false"
```

---

## Complete workflow pattern

```yaml
status:
  fields:
    - path: migrationSucceeded
      value: "{{ jobSucceeded .children.migrationjob }}"
    - path: migrationFailed
      value: "{{ jobFailed .children.migrationjob }}"
    - path: migrationRunning
      value: "{{ jobActive .children.migrationjob }}"

resources:
  - kind: Deployment
    name: app
    when:
      - field: status.migrationSucceeded
        equals: "true"
      - field: status.migrationFailed
        equals: "false"
```

---

## Quick reference

| Note | Signature | Returns |
|------|-----------|---------|
| `jobSucceeded` | `(obj any)` | `bool` |
| `jobFailed` | `(obj any)` | `bool` |
| `jobActive` | `(obj any)` | `bool` |

---

**Next →** [14 — Service Notes](14-service.md)
