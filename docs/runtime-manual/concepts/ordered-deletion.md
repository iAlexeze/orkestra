# Ordered Deletion

When a CR is deleted, Orkestra runs the `onDelete:` block before removing the
finalizer. By default, all resource groups in that block execute concurrently —
deletion requests are submitted in parallel and the finalizer is removed
immediately after.

`ordered: true` changes this. Resource groups execute sequentially. Each group
must be fully deleted — every resource confirmed gone from the cluster — before
the next group starts. The finalizer is not removed until the last group clears.

---

## When you need it

Most operators do not need ordered deletion. Owner references and Kubernetes
garbage collection handle cascade deletion correctly for the common case: delete
the CR, delete the children, done.

Ordered deletion is for the cases where the sequence matters:

**A cleanup Job must complete before its target is deleted.** A database migration
operator might run a Job to drain connections, vacuum tables, and archive logs
before the database Deployment is terminated. If the Deployment disappears first,
the Job has nothing to connect to.

**Infrastructure must be deprovisioned before its configuration is removed.** An
operator that provisions cloud resources might delete the cloud resource in group 0
and the credentials Secret in group 1. Deleting the Secret first would leave the
cloud resource with no way to authenticate for its own teardown.

**Data safety before resource cleanup.** Group 0 takes a final backup. Group 1
deletes the workload. Group 2 cleans up configuration. Each step must complete
before the next.

---

## YAML

```yaml
operatorBox:
  default: true

  onDelete:
    ordered: true
    timeout: 10m       # optional — default 5m for the entire sequence
    groups:
      # Group 0 — runs first, must complete before group 1 starts
      - jobs:
          - name: "{{ .metadata.name }}-drain"
            image: "{{ .spec.image }}"
            command: ["./drain.sh"]

      # Group 1 — runs after group 0 is fully deleted
      - deployments:
          - name: "{{ .metadata.name }}"
        services:
          - name: "{{ .metadata.name }}-svc"

      # Group 2 — runs after group 1 is fully deleted
      - secrets:
          - name: "{{ .metadata.name }}-credentials"
        configMaps:
          - name: "{{ .metadata.name }}-config"
```

Multiple resource types within one group are deleted concurrently — only the
boundary between groups enforces ordering.

---

## How it works

For each group in sequence:

1. Delete all resources in the group concurrently (foreground deletion)
2. Poll the API server every 2 seconds until all resources return 404
3. Move to the next group

Foreground deletion (`DeletePropagationForeground`) means Kubernetes blocks
the delete API response until the object and its dependents are actually gone.
The poll confirms true absence — not just that the deletion was accepted.

---

## Timeout

The `timeout:` field applies to the entire ordered deletion sequence, not per
group. When the timeout is exceeded, the deletion stops and the finalizer is
**not** removed. The CR stays in a terminating state. The error is logged and
emitted as a Warning event on the CR.

```yaml
onDelete:
  ordered: true
  timeout: 15m    # give database drain jobs enough time
```

Default: `5m`. Set it high enough for your longest-running cleanup Job.

A CR stuck in terminating because a cleanup Job timed out is safer than a CR
whose finalizer was removed while cleanup was incomplete. The operator is telling
you something went wrong. Fix the underlying issue — the cleanup Job failure,
the network problem, the external API timeout — and the deletion will complete
on the next attempt.

---

## Without ordered

Without `ordered: true`, the `onDelete:` block behaves as it always has:

```yaml
onDelete:
  jobs:
    - name: "{{ .metadata.name }}-cleanup"
  deployments:
    - name: "{{ .metadata.name }}"
```

All deletions are submitted concurrently. The finalizer is removed immediately.
No polling, no sequencing. Appropriate for the common case where cleanup
resources have no dependencies on each other.

---

## What ordered deletion does not do

**It does not retry failed deletions.** If a resource cannot be deleted — RBAC
issue, API server error, finalizer on the child resource itself — the error is
logged and the group waits for the timeout. Fix the underlying issue rather than
relying on retry logic.

**It does not handle external state.** Ordered deletion manages Kubernetes
resources. If group 0 calls an external API to deprovision cloud infrastructure,
that is done via a Job (which ordered deletion waits for), not by Orkestra
directly.

**It does not guarantee that group 0's resources are fully reconciled before
deletion.** Ordered deletion triggers when the CR receives a deletion timestamp.
If group 0 contains a Job that was just created and has not yet pulled its image,
the Job is deleted in its not-yet-started state. Design cleanup Jobs to be
idempotent and tolerant of being deleted before completion.