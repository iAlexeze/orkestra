# The Core — Runtime, Gateway, and Control Center

The first and most consequential reuse in Orkestra is the core infrastructure itself.

Every operator built on Orkestra runs on the same engine. The reconcile loop, event watching, retry behaviour, leader election, health tracking, dependency ordering — none of this is written by the operator author. It is supplied by the runtime, once, and shared by every operator that runs on it.

This means an operator author's entire responsibility is: what should happen when a CR arrives or changes. The runtime handles everything before and after that point.

---

## One runtime, many operators

The same runtime processes events for every CRD it is configured to watch. A cluster can run operators for databases, pipelines, tenant namespaces, and feature deployments — all on the same engine, with the same operational model.

**One upgrade, all operators.** When the runtime gains a capability — gateway integration, per-target routing, autoscaling, dependency ordering — every operator using it gains it immediately, without changes to operator code.

**One operational model.** Health endpoints, reconcile stats, dependency graphs, graceful shutdown — identical across every operator. A platform team running ten operators does not learn ten operational models; they learn one.

**One test surface.** The runtime's behaviour is tested once. Operator authors test their own logic — not the retry mechanics, not informer cache semantics, not the workqueue. Those come from the shared layer and are not the operator's concern.

---

## The gateway is shared infrastructure

The gateway — intent translation, token validation, schema catalog, cluster routing, surface conflict detection — is part of the shared layer. An operator author does not build a REST API for their CRD. They declare `serve:` in their Katalog and the gateway handles every caller concern.

This means the same gateway instance serves every CRD in every Katalog the runtime loads. One token, one schema catalog endpoint, one `ork serve apply` command — regardless of which operator or which CRD is being called.

The gateway is also where targets become visible to callers. A caller picks a named surface. The gateway stamps the routing decision onto the CR before apply. The runtime reads it and reconciles accordingly. From the caller's side, there is one API and one schema. The routing is transparent.

---

## The Control Center spans all runtimes

The Control Center connects to any number of running operators — local or remote, across clusters. It reads the same endpoint every Orkestra runtime exposes and surfaces reconcile activity, health state, CR listings, and dependency graphs for all of them in one place.

An operator becomes visible in the Control Center the moment it starts. Nothing to register, no plugin to write, no dashboard to configure. The shared runtime contract does it automatically.

This is reuse at the observability level: one interface, any number of operators, zero per-operator configuration.

---

## Where the operator author's work begins

The runtime delivers a CR to the operator at the point of reconcile. Everything that led to that moment — watching for changes, deduplicating events, managing retries, ordering dependencies — was the runtime's responsibility. Everything from that point forward is the operator author's.

Expressed in the Katalog alone, that boundary is a set of declarations: what to validate, what to mutate, what status fields to compute, what external calls to make, when to gate reconciliation. No Go code required.

When Go is needed, the operator author writes hooks or a constructor. Hooks are called by the runtime for each reconcile event. A constructor produces a reconciler that the runtime runs. In both cases, the runtime owns the loop and the operator owns the logic.

---

## Related topics

- [Orkestra Core](../../orkestra-core/index.md) — runtime, gateway, and Control Center in depth
- [OperatorBox](../operatorbox/index.md) — the execution unit each CRD becomes inside the runtime
- [Live API](../live-api/index.md) — the HTTP contract every runtime exposes for the Control Center and ONCOP
