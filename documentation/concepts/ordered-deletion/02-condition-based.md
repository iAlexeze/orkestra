# Condition-Based Deletion

In addition to hard ordered deletion, Orkestra supports a declarative sequencing model using `when:` and `or:` conditions on `onDelete:` blocks. Each deletion step becomes eligible only when its condition evaluates to true.

This model is non-blocking: the CR's finalizer is never held, the CR never gets stuck.

---

## YAML

```yaml
onDelete:
  # Step 1 — runs unconditionally
  jobs:
    - name: "{{ .metadata.name }}-drain"

  # Step 2 — runs only after the Job is gone
  deployments:
    - name: "{{ .metadata.name }}"
  when:
    - "{{ not (resourceExists .children.job) }}"

  # Step 3 — runs when either:
  #   - the Deployment is gone, OR
  #   - the CR has already entered a Failed phase
  services:
    - name: "{{ .metadata.name }}-svc"
  or:
    - "{{ not (resourceExists .children.deployment) }}"
    - "{{ eq .status.phase \"Failed\" }}"

  # Step 4 — runs only after the Service is gone
  secrets:
    - name: "{{ .metadata.name }}-credentials"
  when:
    - "{{ not (resourceExists .children.service) }}"
```

---

## How it works

1. Orkestra evaluates deletion blocks in the order they appear
2. A block with no conditions runs immediately
3. A block with `when:` runs only when **all** conditions are true
4. A block with `or:` runs when **any** condition is true
5. If conditions never become true, the block is skipped
6. The CR's finalizer is not held — deletion is non-blocking

---

## When to use condition-based vs hard ordered

Use **hard ordered deletion** when cleanup *must* complete before the CR disappears — backups, migrations, cloud infrastructure teardown where incomplete teardown leaves orphaned resources.

Use **condition-based deletion** when cleanup is optional, dynamic, or should not block deletion — soft dependencies, best-effort sequencing, or when you want to express deletion logic using the same condition grammar as `onCreate` and `onReconcile`.
