---
title: "Reconciler Model"
weight: 128
---

# How Reconciliation Works

This document explains what happens when you create, update, or delete a custom resource in Orkestra — without any code.

---

## The Big Picture

When you apply a CR to your cluster, Orkestra springs into action. It reads your Katalog, follows your instructions, and makes sure the Kubernetes resources you requested exist and stay correct.

Think of it like a recipe:

- **You** write the recipe (Katalog)
- **You** provide the ingredients (CR spec)
- **Orkestra** follows the recipe and cooks the meal (creates resources)
- **Orkestra** keeps checking the meal (drift correction)

---

## What Happens When You Create a CR

Let's say you create a `Website` CR with:

```yaml
apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: my-blog
spec:
  image: nginx:1.25
  replicas: 2
  port: 80
```

Here's what Orkestra does, step by step.

---

### 1. Orkestra Notices the Change

Orkestra is always watching. When you create, update, or delete a CR, Orkestra's watcher immediately detects it.

The change is placed in a queue — like a to‑do list — waiting to be processed.

---

### 2. A Worker Picks Up the Task

Orkestra runs multiple workers for each CRD. Each worker pulls tasks from the queue and processes them.

If one worker is busy, another picks up the next task. This means Orkestra can handle many changes at once.

---

### 3. Orkestra Reads the CR from Its Cache

Orkestra keeps a local copy of every CR it manages. This copy is always up‑to‑date.

When a worker processes a task, it reads from this local cache. It never goes to the Kubernetes API server — this makes reconciliation fast and reduces load on the cluster.

---

### 4. Orkestra Checks if the CR Is Being Deleted

If the CR has a deletion timestamp, Orkestra knows it's being deleted. It runs cleanup tasks before the CR is removed:

- Removes any finalizers it added
- Cleans up external resources (if you wrote a hook for that)
- Allows the CR to be garbage collected

If the CR is not being deleted, reconciliation continues.

---

### 5. Orkestra Adds a Finalizer

Finalizers are like safety locks. They prevent the CR from being deleted until Orkestra has finished its work.

When Orkestra first sees a CR, it adds its finalizer. This ensures that if you later delete the CR, Orkestra gets a chance to clean up before the CR disappears.

---

### 6. Orkestra Adds Tracking Labels and Annotations

Orkestra adds labels and annotations to your CR so you can see it's being managed:

```
labels:
  orkestra.konductor.io/managed: "true"

annotations:
  orkestra.konductor.io/managed-by: website-katalog
  orkestra.konductor.io/managed-since: 2026-03-25T10:30:45Z
```

These are purely informational — they help you understand which operator is managing your resources.

---

### 7. Orkestra Evaluates Conditions

Your Katalog might have conditions. For example, you might only want to create a public load balancer if `exposePublicly: true`.

Orkestra checks these conditions against your CR. If conditions are not met, it skips creating that resource.

---

### 8. Orkestra Runs Hooks (if you wrote any)

Hooks are optional Go functions. If you wrote one, Orkestra runs it now. Hooks can do anything Go can do — call external APIs, validate complex rules, or modify the CR before resources are created.

If you didn't write a hook, Orkestra skips this step.

---

### 9. Orkestra Resolves Templates

Your Katalog contains templates. Templates look like this:

```yaml
image: "{{ .spec.image }}"
replicas: "{{ .spec.replicas }}"
name: "{{ .metadata.name }}-app"
```

Orkestra replaces these templates with values from your CR:

| Template | Value from CR | Result |
|----------|---------------|--------|
| `{{ .spec.image }}` | `nginx:1.25` | `nginx:1.25` |
| `{{ .spec.replicas }}` | `2` | `2` |
| `{{ .metadata.name }}-app` | `my-blog` | `my-blog-app` |

After resolution, Orkestra has the exact specification for each resource it needs to create.

---

### 10. Orkestra Creates Resources

Orkestra creates the resources you defined:

- **Deployments** — with the right image, replicas, and ports
- **Services** — with the right type and ports
- **Secrets** — with your data or copied from other secrets
- **ConfigMaps** — with your configuration
- **Jobs** — for one‑time tasks
- **CronJobs** — for scheduled tasks

Each resource gets an owner reference pointing back to your CR. This means when your CR is deleted, Kubernetes automatically deletes all these child resources.

---

### 11. Orkestra Enables Drift Correction

If you marked a resource with `reconcile: true`, Orkestra will check it on every reconcile, not just on creation.

If someone manually changes the Deployment's image or replicas, Orkestra notices and changes it back. This keeps your resources in sync with your CR.

---

### 12. Orkestra Updates Status

Orkestra updates your CR's status so you can see what happened:

```yaml
status:
  phase: Ready
  lastReconcile: 2026-03-25T10:30:45Z
  readyReplicas: 2
```

You can see this with `kubectl describe website my-blog`.

---

### 13. Orkestra Records Metrics

Orkestra tracks:

- How many reconciliations happened
- How long each reconciliation took
- How many resources are being managed
- How deep the queue is
- How many workers are active

These metrics are exposed at `/metrics` and can be collected by Prometheus.

---

### 14. Orkestra Emits Events

Orkestra emits Kubernetes events for important actions:

```
$ kubectl get events
LAST SEEN   TYPE      REASON              OBJECT
2m          Normal    Reconciled          website/my-blog
5s          Normal    FinalizerAdded      website/my-blog
```

You can see these with `kubectl describe website my-blog`.

---

### 15. Orkestra Updates Health

Orkestra updates the health of your CRD. If everything worked, it's healthy. If there were errors, it may become degraded.

You can check health at:

```
GET /katalog/website/health
```

A healthy CRD returns `200 OK`. A degraded CRD returns `503 Service Unavailable`.

---

## What Happens When You Update a CR

When you change your CR — for example, updating the image from `nginx:1.25` to `nginx:1.26` — the same process happens:

1. Orkestra detects the change
2. A worker picks it up
3. Orkestra reads the updated CR from cache
4. It checks for deletion (not happening)
5. It ensures finalizers are present (they already are)
6. It runs hooks (if any)
7. It resolves templates (the new image appears)
8. It updates the Deployment (because `reconcile: true`)
9. It updates status
10. It records metrics
11. It emits events

The update happens automatically. You don't need to restart anything.

---

## What Happens When You Delete a CR

When you delete a CR:

1. Orkestra detects the deletion
2. A worker picks it up
3. Orkestra sees the deletion timestamp
4. It runs cleanup hooks (if you wrote any)
5. It removes its finalizers
6. Kubernetes garbage collects all child resources (Deployments, Services, etc.)

After cleanup, the CR disappears from the cluster.

---

## Summary

| Step | What Orkestra Does |
|------|-------------------|
| 1 | Detects change (create, update, delete) |
| 2 | Queues the task |
| 3 | Worker picks it up |
| 4 | Reads CR from cache |
| 5 | Checks for deletion |
| 6 | Adds finalizers (if not deleted) |
| 7 | Adds tracking labels and annotations |
| 8 | Evaluates conditions |
| 9 | Runs hooks (if any) |
| 10 | Resolves templates |
| 11 | Creates or updates resources |
| 12 | Corrects drift (if `reconcile: true`) |
| 13 | Updates status |
| 14 | Records metrics |
| 15 | Emits events |
| 16 | Updates health |
| 17 | Removes finalizers (if deleted) |

All of this happens automatically. You just write the Katalog. 🎼