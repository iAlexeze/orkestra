# Reconciler Model

When you apply a CR to your cluster, Orkestra reconciles it. It reads your Katalog, follows your instructions, and makes sure the Kubernetes resources you declared exist and stay correct — automatically, without any code.

Orkestra creates one reconciler, one informer, and one worker pool **per CRD**. Each CRD in your Katalog gets its own independent reconcile loop — its own watch stream from the API server, its own workqueue, and its own configurable concurrency. Adding a second CRD does not share or contend with the first.

Think of it like a recipe: you write the recipe (Katalog), you provide the ingredients (CR spec), Orkestra follows the recipe (creates resources), and Orkestra keeps checking the result (drift correction).

---

## Three flows

**Create / Update** — the main path. A new CR appears or an existing one changes. Orkestra normalizes, mutates, validates, runs hooks, resolves templates, creates or updates resources, patches status, and records health.

**Delete** — a CR receives a deletion timestamp. Orkestra runs `onDelete:` hooks and templates, removes finalizers, and lets Kubernetes garbage-collect child resources via owner references.

**Drift correction** — a child resource is manually changed. Because the child emits a watch event that Orkestra's informer picks up, the parent CR is requeued and its `onReconcile:` templates run again, restoring the declared state.

---

## Where to go next

- [Create and Update](create-update/) — the 15-step reconcile pipeline
- [Delete](delete/) — finalizer lifecycle and cleanup sequencing
