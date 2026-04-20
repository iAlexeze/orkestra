# Condition‑Based Ordered Deletion

In addition to hard ordered deletion (`ordered: true`), Orkestra supports a fully declarative sequencing model using **conditions**. Each deletion step becomes eligible only when its condition evaluates to true. Conditions may use both `when:` (AND) and `anyOf:` (OR), giving you fine‑grained control over deletion order without blocking the CR’s finalizer.

This model is ideal when deletion order matters, but cleanup should remain **non‑blocking**, **best‑effort**, and **fully declarative**.

---

## When to use it

Condition‑based sequencing is useful when:

- cleanup should **not** block CR deletion  
- deletion order depends on **resource existence**  
- deletion depends on **status**, **metrics**, or **cross‑cluster signals**  
- you want flexible branching using `when:` and `anyOf:`  
- you want to avoid finalizer deadlocks  
- you want to express deletion logic the same way as `onCreate` and `onReconcile`

This model is perfect for soft dependencies:

- “Delete this only after the Job is gone.”  
- “Delete this only if the Deployment exists.”  
- “Delete this only when the external condition is true.”  

---

## YAML Example: Condition‑Based Sequencing

```yaml
onDelete:
  # Step 1 — runs unconditionally
  jobs:
    - name: "{{ .metadata.name }}-drain"

  # Step 2 — runs only after the Job is gone
  deployments:
    - name: "{{ .metadata.name }}"
  when:
    - "{{ resourceExists (exists .children.job) }}"

  # Step 3 — runs when either:
  #   - the Deployment is gone, OR
  #   - the CR has already entered a Failed phase
  services:
    - name: "{{ .metadata.name }}-svc"
  anyOf:
    - "{{ not (resourceExists .children.deployment) }}"
    - "{{ eq .status.phase 'Failed' }}"

  # Step 4 — runs only after the Service is gone
  secrets:
    - name: "{{ .metadata.name }}-credentials"
  when:
    - "{{ not (resourceExists .children.service) }}"
```

Each block is evaluated in order.  
A block executes only when its condition(s) are satisfied.

---

## How it works

1. Orkestra evaluates deletion blocks in the order they appear.  
2. A block with no conditions runs immediately.  
3. A block with `when:` runs only when **all** conditions are true.  
4. A block with `anyOf:` runs when **any** condition is true.  
5. If conditions never become true, the block is skipped.  
6. The CR’s finalizer is **not** held — deletion is non‑blocking.

This is **soft ordered deletion**: sequencing without enforcement.

---

## Comparison with Hard Ordered Deletion

| Feature | Hard Ordered (`ordered: true`) | Condition‑Based (`when:` / `anyOf:`) |
|--------|--------------------------------|--------------------------------------|
| Blocks finalizer | ✔ Yes | ✘ No |
| Has timeout | ✔ Yes | ✘ No |
| Sequential guarantee | ✔ Enforced | ✘ Best‑effort |
| CR can get stuck | ✔ Yes | ✘ Never |
| Flexible conditions | Limited | ✔ Full condition grammar |
| Use case | Safety‑critical cleanup | Optional sequencing |

---

## When to choose which

Use **hard ordered deletion** when cleanup *must* complete before the CR disappears (backups, migrations, cloud teardown).

Use **condition‑based sequencing** when cleanup is optional, dynamic, or should not block deletion.
