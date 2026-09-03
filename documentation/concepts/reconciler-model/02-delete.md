# Delete

When a CR receives a deletion timestamp, Orkestra switches to the delete path.

---

## What happens

1. **Deletion timestamp detected** — the CR is in the cache with `.metadata.deletionTimestamp` set
2. **`onDelete:` runs** — hooks and templates in the `onDelete:` block execute. If `ordered: true`, groups run sequentially. See [Ordered Deletion](../ordered-deletion/)
3. **Finalizer removed** — after `onDelete:` completes, Orkestra removes its finalizer from the CR
4. **Kubernetes GC** — with the finalizer gone, Kubernetes proceeds with deletion. Child resources that have owner references pointing to the CR are garbage-collected automatically

---

## Owner references

Every child resource created by Orkestra's templates gets the parent CR as its owner reference. This means child resources are cleaned up by Kubernetes itself when the parent is deleted — Orkestra's `onDelete:` block is only needed when you have cleanup that Kubernetes cannot handle: external infrastructure, Jobs that must run before deletion, credentials that must be revoked.

---

## What the finalizer protects

The finalizer exists to guarantee that `onDelete:` runs before the CR disappears. Without it, deleting a CR would immediately start Kubernetes GC, potentially tearing down resources while an `onDelete:` Job is still running.

If `onDelete:` is not declared, Orkestra still adds the finalizer and removes it immediately after confirming there is nothing to clean up.
