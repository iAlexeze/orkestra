---
title: "Kubernetes Basics"
weight: 1
description: "This page is for engineers who are new to Kubernetes or who want a clear"
---

!!! tip "Skip if you know Kubernetes"
    This page is for engineers who are new to Kubernetes or who want a clear
    mental model before reading about Orkestra. If you already know what CRDs,
    operators, and reconciliation are, go straight to
    [Understanding Orkestra](./understanding-orkestra.md).

---

## What Kubernetes is

Kubernetes is a platform for running software in containers. You tell it what you
want — three copies of this application, always available, updated without downtime
— and it makes it happen. When something breaks, Kubernetes notices and fixes it.
When you change your mind, Kubernetes adjusts.

The key idea is **desired state**: you describe what should exist, and Kubernetes
continuously works to make the cluster match that description.

```yaml
# You write this
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  # ...
```

Kubernetes reads that file and makes sure three copies of `my-app` are always
running. If one crashes, Kubernetes starts a new one. If you change `replicas` to
five, Kubernetes starts two more. You do not tell Kubernetes how to do this. You
tell it what you want.

---

## Built-in resource types

Kubernetes ships with a set of built-in resource types. The most common:

| Kind | What it is |
|---|---|
| `Deployment` | A set of identical pod replicas that Kubernetes keeps running |
| `Service` | A stable network address that routes to a set of pods |
| `ConfigMap` | Configuration data that pods can read |
| `Secret` | Sensitive data (passwords, tokens) that pods can read |
| `Namespace` | A logical boundary for grouping resources |
| `Pod` | One or more containers running together on a node |

Each of these is stored in Kubernetes, validated against a schema, served over an
API, and watched by a built-in controller that keeps the cluster in the desired state.

---

## What a CRD is

`CRD` stands for **Custom Resource Definition**. It is how you add your own resource
types to Kubernetes.

Suppose you are building a platform that manages databases. You want your users to
be able to write:

```yaml
apiVersion: platform.myorg.io/v1
kind: Database
metadata:
  name: production-db
spec:
  engine: postgres
  version: "14"
  storage: 100Gi
```

For this to work, Kubernetes needs to know what a `Database` is — what fields it
accepts, how it is stored, how it should be validated. That is what a CRD does.

A CRD is a declaration that tells Kubernetes:

> "There is a new kind of object called `Database`. Here is its schema. Store it,
> validate it, and serve it over the API like any other resource."

After you apply the CRD, `kubectl get databases` works. `kubectl apply -f database.yaml`
works. The object is stored in etcd. Kubernetes treats it like a first-class resource.

What Kubernetes does **not** do is tell you what the `Database` should actually create.
It stores the object and waits. Something else has to watch for `Database` objects and
act on them. That something is an **operator**.

---

## What an operator is

An operator is a program that:

1. Watches a specific CRD for new, changed, or deleted objects
2. Reads what the object declares (the desired state)
3. Creates or modifies Kubernetes resources to make that desired state real
4. Continuously checks that the desired state is maintained

For our `Database` example, the operator would:

- Watch for new `Database` objects
- When one appears, create a `StatefulSet`, a `Service`, a `PersistentVolumeClaim`
- When the `Database` spec changes, update those resources
- When the `Database` is deleted, clean up the resources it created

The operator runs as a pod in the cluster, continuously watching and reconciling.
The loop it runs — watch, compare, act — is called the **reconcile loop**.

---

## The reconcile loop

The reconcile loop is the fundamental pattern of Kubernetes operators.

```
Desired state (what the CR says)
        │
        ▼
Actual state (what exists in the cluster)
        │
  Are they the same?
        │
    No ─┤─ Yes → do nothing
        │
        ▼
  Make them the same
        │
        ▼
Emit an event, update metrics
        │
        └─── repeat on next change or resync
```

This loop runs continuously. It is **level-triggered**, not edge-triggered. The
operator does not track what it did last time — it looks at the current desired
state and the current actual state and makes them match. This means a reconcile
that is interrupted partway through is safe to retry — the next run will pick up
where it left off.

---

## Why operators are hard to write

The pattern is simple. The implementation is not.

To write an operator from scratch, you need to:

- Register your Go types with the Kubernetes API scheme
- Set up an informer that watches the API server for changes to your CRD
- Create a workqueue that buffers events and deduplicates rapid changes
- Write a reconcile function that handles create, update, and delete
- Add a worker pool that processes the queue concurrently
- Manage finalizers to prevent dirty deletion
- Handle retries with exponential backoff
- Emit Kubernetes events so users can see what happened
- Expose Prometheus metrics
- Set up leader election so only one instance reconciles at a time
- Write a Dockerfile, build a binary, push an image
- Write a Helm chart or deployment manifests

All of this before you write a single line of your actual business logic. This is
the problem Orkestra solves.

---

## Next: Understanding Orkestra

Now that you have the foundation, the next page explains exactly what Orkestra
automates and how.

[Understanding Orkestra →](./understanding-orkestra.md)