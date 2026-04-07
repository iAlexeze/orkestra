---
title: "No Trace Runtime"
weight: 63
---

# The Runtime That Leaves No Trace

*Why a continuous operator should exist without becoming part of your cluster.*

---

## The Mistake We Stopped Seeing

Kubernetes taught us to think declaratively.

You declare a resource.
The system converges toward it.
The API remains the source of truth.

And then—right at the point where behavior begins—we break that model.

To make a CRD *do something*, we install an operator:

* a deployment
* a controller
* RBAC rules
* webhook configurations
* lifecycle objects
* metrics endpoints

The system that interprets the declaration becomes something that must itself be declared, deployed, upgraded, and eventually removed.

Over time, clusters accumulate these interpreters.

They stay.
They stack.
They outlive their purpose.

The result is not just operational overhead.
It is a violation of the original model:

> The thing that interprets the declaration becomes part of the state being declared.

---

## The Alternative: A Runtime, Not a Resident

Orkestra takes a different position:

> The interpreter should not become part of the system it interprets.

It is a **continuous runtime**—not a job, not a one-off process—but it refuses to become a persistent resource inside the cluster.

While running, it behaves like a full operator system:

* it watches resources
* it reconciles continuously
* it exposes metrics
* it participates in leader election
* it manages lifecycle and drift

But it does so **without installing itself into the cluster’s API**.

No CRDs.
No configuration objects.
No runtime state stored in Kubernetes.

The cluster knows nothing about Orkestra.

And that is intentional.

---

## Full Presence, Zero Ownership

While Orkestra is running, it is fully present.

You can observe it through:

* logs
* metrics
* health endpoints
* the effects it produces in the cluster

It is not invisible in behavior.
It is invisible in **ownership**.

It does not introduce a parallel control plane.
It does not claim territory in the API server.
It does not require the cluster to model *its existence*.

Everything it needs, it derives:

* from CRDs (structure)
* from CRs (desired state)
* from the Katalog (behavior)

The cluster is the source of truth.
Orkestra is only the interpreter.

---

## Precision Instead of Presence

Orkestra exposes only what your declarations require—and nothing more.

Even when capabilities are enabled, they are not materialized unless used:

* `/convert` exists **only when conversion paths are declared**
* `/validate` exists **only when validation rules exist**
* `/mutate` exists **only when mutation rules are defined**

:::important[The Philosophy]
No declaration → no endpoint

No endpoint → no webhook

No webhook → no surface area
:::

This is not optimization.
It is alignment with intent.

The runtime does not generalize.
It specializes exactly to what you declared.

---

## Continuous, but Not Persistent

Orkestra is not ephemeral in execution.
It is continuous.

It runs control loops.
It performs reconciliation.
It maintains convergence over time.

But it is ephemeral in **identity**.

It does not persist as an object in the system.
It does not leave behind configuration that represents itself.
It does not bind the cluster to its existence.

This creates a clean separation:

* **State lives in the cluster**
* **Behavior lives in the runtime**

And the two are not entangled.

---

## The Moment It Stops

When Orkestra shuts down, it does something most systems cannot:

> It removes every trace of its presence—without affecting the system it managed.

* webhook configurations → deleted
* conversion endpoints → removed
* admission hooks → gone
* internal servers → stopped

What remains:

* your CRDs
* your custom resources
* all managed Kubernetes objects

Everything it created *as part of reconciliation* remains.
Everything it created *to exist as a runtime* disappears.

The cluster is left:

* structurally unchanged
* operationally intact
* free of artifacts

No uninstall procedures.
No cleanup scripts.
No residual infrastructure.

---

## No Ownership, No Lock-In

Because Orkestra does not embed itself into the cluster:

* it does not own your resources
* it does not own your lifecycle
* it does not own your future

You can stop it at any time.

You can restart it later—it resumes from cluster state.

You can replace it with another system—the cluster remains valid.

There is no migration step.
There is no extraction process.

:::tip
The system does not depend on Orkestra to *exist*.
Only to *converge*.
:::

---

## Why This Matters

Clusters are long-lived systems.

Runtimes should not be.

Every persistent control plane component introduces:

* upgrade coupling
* failure surface
* operational burden
* long-term ownership

Most of these persist not because they are needed—but because they cannot leave cleanly.

Orkestra changes that constraint.

It proves that a system can:

* run continuously
* provide full operator behavior
* integrate deeply with Kubernetes

…and still leave **no trace when removed**.

---

## The Clean Boundary

This model restores a clean boundary that Kubernetes originally implied:

* **The cluster stores truth**
* **The runtime enforces it**

But the runtime is not part of that truth.

It is replaceable.
It is restartable.
It is optional.

And critically:

> It is forgettable.

---

## The Principle

Orkestra is built on a simple constraint:

> The runtime must not become part of the system it operates.

From that constraint, everything else follows:

* no CRDs for itself
* no stored state
* no persistent identity
* no residue on exit

What remains is a system that behaves like a full operator platform while running—

and like nothing was ever there when it stops.

---

## The Guarantee

Orkestra leaves your cluster exactly as it found it—

with one exception:

Everything you asked it to build is still there.

<!-- Nothing else is. -->
No residue.
No footprint.
No friction.

Just a system that worked,
and a cluster that never had to remember why.
