# Standards Over Invention

Every platform project faces the same temptation: to build the thing you understand rather than use the thing that already exists. A new auth mechanism. A new wire format. A new credential store. Each one feels justified at the time — tighter integration, better control, fewer dependencies. Each one becomes a wall that users have to climb before they can do the thing they came to do.

Orkestra is deliberate about this. Where a standard exists and solves the problem, Orkestra uses it. Where one does not, Orkestra builds. The line between those two is not arbitrary — it is the question that shaped every major design decision in the system.

---

## What Orkestra uses

**`docker login` and `~/.docker/config.json`** — Orkestra has no `ork login` command. If you can push an image to a container registry, you can push a pattern to a pattern registry. The same credentials, the same credential store, the same rotation process. One auth surface for the entire ecosystem.

**`OCI`** — The pattern registry is OCI-compatible. Motifs, Katalogs, Komposers, and E2E suites are OCI artifacts with annotations that carry the quality signal — whether simulate and e2e passed before the artifact was published. Consumers can inspect that signal before they pull, using any OCI-compatible tool. Orkestra did not invent a format. It used the one the industry already agreed on.

**`kind`** — The E2E harness creates ephemeral clusters with `kind`. Orkestra did not write a cluster provisioner. `kind` solves that problem. The `e2e.yaml` describes what to test; the cluster it runs in is not a new concept to learn.

**`Prometheus`** — Operator metrics are exposed as Prometheus metrics. The autoscaler's condition engine is expressed in terms operators already understand. No new observability format, no new scrape target configuration.

**`controller-runtime`** — The constructor path gives operators exactly the interface they already use. Existing `controller-runtime` operators migrate without rewriting reconcile logic. The `pkg/kubeclient/merge.go` bridge is the only new surface between them.

**`text/template`** — The Orkestra expression language is Go's `text/template` extended, not replaced. The same delimiters, the same evaluation model, the same mental model anyone who has written a Helm chart or a Go template already has. Notes adds the operator-specific functions on top — `.children.*`, `.cross.*`, `randomBase64`, `quantityLessThan` — but the substrate is already familiar.

---

## Where Orkestra builds

`controller-runtime` is excellent at what it does. It is built for one operator, one binary, one process. Orkestra is built for many.

The question `controller-runtime` answers is: *how do I write an operator?* The question Orkestra answers is different: *how do I treat operators as behaviours rather than binaries? How do I share infrastructure across many operators instead of rebuilding it for each one?*

`client-go` provides the primitives that make that possible — and the choice matters beyond preference. When your runtime is built for one, the informer cache, the workqueue, and leader election are each operator's problem to solve. When it is built for many, one process pays that cost once, regardless of how many CRDs it serves. A panic in one CRD's reconciler is caught and isolated — the others keep running. Per-CRD autoscaling means each CRD gets its own worker pool tuned to its load, not a shared budget divided between them. One operator can read another's live state at template time through `.cross.*` — something that only makes sense when there is a runtime that owns the relationship between them.

None of that is possible when the runtime assumes it is one operator in one binary. `client-go` is not a lower-level choice — it is the choice that makes Orkestra a platform rather than scaffolding.

The name `Reconcile()` was kept. The problem was never the name. The problem was everything around it — the informer, the workqueue, the worker pool, the finalizer, the status management, the events, the metrics, the leader election, the webhooks, the security layer. All of it exists to support that one method. You write the `Reconcile()` — in whichever form fits your problem — and Orkestra provides everything else.

---

## The question

Every decision passed through the same filter: does this already exist? If yes, use it. If it exists but only partially, stand on what is there. If it does not exist, build it.

The result is that adopting Orkestra does not mean adopting a new auth system, a new distribution protocol, or a new CLI convention. It means adding the layer that was missing — the composition, the quality gates, the shared infrastructure — without replacing the tools that were already working.

You do not climb a wall to get in. You walk through the door that was already there.
