# Kubernetes, operators, and why Orkestra

For engineers who are new to Kubernetes or want a clear mental model before writing their first Katalog. If you already know what CRDs, operators, and reconciliation are, go straight to [Learning to Orkestrate](./index.md).

---

## What Kubernetes is

Kubernetes is a platform for running software in containers. You describe what should exist, and Kubernetes continuously works to make the cluster match that description.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  # ...
```

Kubernetes reads that file and keeps three copies of `my-app` running. If one crashes, Kubernetes starts a new one. If you change `replicas` to five, Kubernetes starts two more. You declare the desired state — Kubernetes figures out how.

---

## Built-in resource types

Kubernetes ships with a set of built-in kinds:

| Kind | What it is |
|---|---|
| `Deployment` | A set of identical pod replicas that Kubernetes keeps running |
| `Service` | A stable network address that routes to a set of pods |
| `ConfigMap` | Configuration data that pods can read |
| `Secret` | Sensitive data — passwords, tokens — that pods can read |
| `Namespace` | A logical boundary for grouping resources |
| `Pod` | One or more containers running together on a node |

Each kind is stored in Kubernetes, validated against a schema, and watched by a built-in controller that keeps the cluster in the desired state.

---

## What a CRD is

`CRD` stands for Custom Resource Definition. It is how you add your own resource types to Kubernetes.

Suppose you are building a platform that manages databases. You want your users to write:

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

For this to work, Kubernetes needs to know what a `Database` is — what fields it accepts, how it is stored, how it should be validated. That is what a CRD does:

> "There is a new kind called `Database`. Here is its schema. Store it, validate it, serve it over the API like any other resource."

After you apply the CRD, `kubectl get databases` works. The object is stored. Kubernetes treats it as a first-class resource.

What Kubernetes does **not** do is act on it. It stores the object and waits. Something else has to watch for `Database` objects and do the actual work. That is an **operator**.

---

## What an operator is

An operator is a program that:

1. Watches a CRD for new, changed, or deleted objects
2. Reads the declared desired state
3. Creates or modifies Kubernetes resources to make that state real
4. Continuously checks that the desired state is maintained

For the `Database` example, the operator would create a `StatefulSet`, a `Service`, and a `PersistentVolumeClaim` when a `Database` appears — update them when the spec changes — clean them up when the object is deleted.

The loop it runs — watch, compare, act — is the **reconcile loop**.

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
        └─── repeat on next change or resync
```

This loop is level-triggered, not edge-triggered. The operator does not track what it did last time — it looks at the current desired and actual state and makes them match. An interrupted reconcile is safe to retry.

---

## Why operators are hard to write

The pattern is simple. The implementation is not.

To write an operator from scratch, you need to:

- Register your Go types with the Kubernetes API scheme
- Set up an informer that watches the API server for changes
- Create a workqueue that buffers and deduplicates rapid changes
- Write a reconcile function that handles create, update, and delete
- Add a worker pool for concurrent processing
- Manage finalizers to prevent dirty deletion
- Handle retries with exponential backoff
- Emit Kubernetes events so users can see what happened
- Expose Prometheus metrics
- Set up leader election
- Write a Dockerfile, build a binary, push an image
- Write a Helm chart or deployment manifests

All of this before you write a single line of business logic. The community answer to this is controller-runtime (the library) or Kubebuilder / Operator SDK (the scaffolding tools). They reduce the boilerplate — you still write Go, manage a project layout, maintain generated code, and own the reconcile loop.

---

## Where Orkestra fits

The list above is the cost of writing an operator with controller-runtime or Kubebuilder. Those are the standard tools — you write Go, own the reconcile function, manage generated code and a build pipeline.

Orkestra is a runtime that implements the reconcile loop for you. You write a Katalog — a YAML declaration of what CRD your operator manages, what resources it creates per CR, and what status transitions look like. Orkestra reads the Katalog and runs a full operator from it — informers, workqueue, leader election, metrics, events.

```yaml
# katalog.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: database-operator
spec:
  crds:
    database:
      group: platform.myorg.io
      version: v1alpha1
      # ...
  operatorBox:
    reconciler:
      statefulSet:
        template: ./templates/statefulset.yaml
      service:
        template: ./templates/service.yaml
      pvc:
        template: ./templates/pvc.yaml
```

`ork run` starts the operator locally against a real cluster. No image build, no Helm chart.

The file count is lower because the structure is declared rather than implemented — a Katalog, a CRD, and one or two templates covers most operators. When the pattern cannot be expressed declaratively, Orkestra lets you attach Go logic at specific points in the reconcile pipeline without rewriting the whole controller.

---

## Next

- [Learning to Orkestrate](./index.md) — the map of all runnable examples
- [Writing your first Katalog](../03-writing-your-first-katalog.md) — go from nothing to a running operator in one file
- [Migration Guide](./07-migration.md) — if you have an existing controller-runtime operator
