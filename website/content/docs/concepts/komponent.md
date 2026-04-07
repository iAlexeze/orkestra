---
title: "Orkestra Komponents"
weight: 50
description: "Orkestra is made of several pieces, each with one job. They start in a specific order and shut down in reverse. This doc..."
---

Orkestra is made of several pieces, each with one job. They start in a specific order and shut down in reverse. This document explains each piece and how they fit together.

---

## The Pieces at a Glance

| Piece | What It Does |
|-------|--------------|
| **HealthServer** | Answers questions like "Is Orkestra alive?" and "Are my CRDs healthy?" |
| **Kubeclient** | Talks to Kubernetes — reads, writes, watches |
| **EventRecorder** | Creates events you see in `kubectl describe` |
| **QueueRegistry** | Holds a to-do list for each CRD |
| **DefaultWorkqueue** | A shared to-do list for simple CRDs |
| **SharedInformerFactory** | Watches for changes to your CRDs |
| **DependencyKordinator** | Makes sure CRDs start in the right order |
| **KonductorElection** | Picks one leader when multiple copies run |
| **Orkestra** | Starts and stops everything, in the right order |

---

## HealthServer

**First to start, last to stop.**

The HealthServer answers questions about Orkestra's health. It runs a small web server that responds to:

| Endpoint | What It Answers |
|----------|-----------------|
| `/health` | "Is Orkestra alive?" → Always 200 when running |
| `/ready` | "Is Orkestra ready to work?" → 200 after everything is started |
| `/metrics` | Numbers about what Orkestra is doing (for Prometheus) |
| `/katalog` | List of all CRDs, their health, and dependencies |
| `/katalog/{crd}` | Details about one CRD |
| `/katalog/{crd}/health` | Is this CRD healthy? |

These endpoints are created before the server starts. When you run `ork run`, Orkestra registers all the routes, then starts the server.

---

## Kubeclient

The Kubeclient is Orkestra's way of talking to Kubernetes. It can:

- Read and write any Kubernetes resource
- Watch for changes
- Talk to custom resources (your CRDs)
- Talk to built-in resources (Pods, Deployments, etc.)

Everything Orkestra does goes through the Kubeclient. It's started early because other pieces need it.

When you write a hook (Go code), Orkestra gives you access to the Kubeclient so you can read or write anything you need.

---

## EventRecorder

The EventRecorder creates Kubernetes events. Events appear in:

```
kubectl describe website my-blog
kubectl get events --watch
```

You'll see events when:

- A CR is reconciled successfully
- A CR fails to reconcile
- A finalizer is added or removed
- A leader is elected

Events help you understand what Orkestra is doing without looking at logs.

---

## QueueRegistry and DefaultWorkqueue

These are Orkestra's to-do lists.

**QueueRegistry** creates a separate to-do list for each CRD. If you have 5 CRDs, you have 5 queues. Each queue has its own depth (how many tasks are waiting) and retry behavior.

**DefaultWorkqueue** is one shared to-do list. If a CRD is simple and doesn't need its own queue, you can use the default one.

Each task in a queue has two pieces:

- **Key** — which resource to work on (like `default/my-blog`)
- **GVK** — what kind of resource it is (like `Website`)

When a worker picks up a task, it knows exactly which CRD and which CR to reconcile.

---

## SharedInformerFactory

The SharedInformerFactory watches for changes. It keeps a local copy of every CR it manages.

When something changes — a CR is created, updated, or deleted — the informer notices and adds a task to the queue.

The informer also keeps a local cache. This means workers never need to ask the Kubernetes API for the current state. They read from the cache, which is fast and reduces load on the cluster.

Each CRD can have its own resync interval. A resync is like a "just checking" event — even if nothing changed, the informer re‑adds the resource to the queue so Orkestra can make sure everything is still correct.

---

## KordinatorRegistry

The KordinatorRegistry is a lookup table. It maps a CRD (like `Website`) to:

- Its informer (how to watch it)
- Its reconciler (how to reconcile it)
- Its metadata (name, group, version, etc.)

When a worker picks up a task, it looks in this registry to find the right reconciler for that CRD.

---

## DependencyKordinator

This is the brain of Orkestra. It makes sure CRDs start in the right order.

### How It Starts

1. Reads all your CRDs and their dependencies
2. Figures out the order — dependencies first, dependents later
3. For each CRD in order:
   - Creates the reconciler (the code that does the work)
   - Waits for the informer to have the latest data
   - Starts workers (the number you set in `workers`)
   - Signals "ready" to any CRDs that depend on this one
4. Starts a background checker for CRDs that aren't installed yet
5. Marks itself ready when everything is running

### How It Processes Tasks

Workers pull tasks from the queue and call the reconciler. If the reconciler fails, the task goes back to the queue with a delay (exponential backoff). If it succeeds, the task is forgotten.

### What If a CRD Is Missing?

If a CRD is declared in your Katalog but not installed in the cluster:

- Orkestra doesn't start workers for it
- CRDs that depend on it wait
- A background checker keeps checking
- When the CRD appears, Orkestra starts its workers and unblocks the dependents

### How It Stops

When Orkestra shuts down:

1. Stops accepting new tasks
2. Waits for current tasks to finish
3. Stops CRDs in reverse order (dependents first, dependencies last)

---

## KonductorElection

If you run multiple copies of Orkestra (for high availability), only one should do the work. The KonductorElection picks one leader.

All copies watch for changes (informers run everywhere). But only the leader runs the workers.

If the leader dies, another copy becomes leader. Because all copies have warm caches, the new leader can start immediately.

When a leader is elected, it prints a banner so you know which pod is in charge.

---

## Orkestra (the Manager)

This piece starts and stops everything else. It's not a component itself — it's the conductor.

```bash
ork run --katalog my-katalog.yaml
```

When you run this command:

1. Orkestra creates all the components
2. It starts them in the right order
3. It waits for a signal to stop (like Ctrl+C)
4. When it stops, it shuts everything down in reverse order

If any component fails to start, Orkestra stops and tells you what went wrong.

---

## How They Fit Together

```
1. HealthServer starts — routes are registered, server isn't listening yet
2. Kubeclient starts — can now talk to Kubernetes
3. EventRecorder starts — can now create events
4. QueueRegistry starts — to-do lists are ready
5. DefaultWorkqueue starts — shared to-do list is ready
6. SharedInformerFactory starts — begins watching for changes
7. DependencyKordinator starts — waits for dependencies, starts workers
8. KonductorElection starts — picks a leader, runs the workers on the leader
9. HealthServer starts listening — now ready to answer health checks
```

When you press Ctrl+C:

```
1. KonductorElection stops — leader releases the lock
2. DependencyKordinator stops — workers finish and stop
3. SharedInformerFactory stops — stops watching for changes
4. Queues stop — no new tasks accepted
5. EventRecorder stops — flushes remaining events
6. Kubeclient stops — closes connections
7. HealthServer stops — no longer answers health checks
```

---

## What This Means for You

You don't need to know the details of how these components work. You just write a Katalog. Orkestra handles:

- Watching your CRDs
- Queueing changes
- Processing them in order
- Handling dependencies
- Recovering from failures
- Shutting down cleanly

The components exist so you don't have to build them yourself. 🎼