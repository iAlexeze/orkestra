---
title: "Validation Is Not Enough"
weight: 5
---

# Validation Isn’t Enough — Why Kubernetes Needs Continuous Enforcement

*Orkestra Project — March 2026*

---

## The assumption Kubernetes makes

Kubernetes assumes something simple:

> If a resource is valid when it is created, it will remain valid.

That assumption is baked into the API model.

* You define validation rules in a CRD
* The API server enforces them at admission time
* If the object passes, it is stored
* From that point on, Kubernetes trusts it

For many workloads, this works.

But at scale — and over time — it breaks down.

---

## The gap no one talks about

Validation in Kubernetes is **momentary**.

It happens:

* at `kubectl apply`
* at update
* at admission webhook execution

And then it stops.

What Kubernetes does *not* do:

* re-evaluate resources as the system evolves
* detect when a resource becomes invalid later
* surface drift between desired and acceptable states

That creates a blind spot.

---

## A simple example

Imagine a CRD:

```yaml
spec:
  replicas: integer (must be >= 1)
```

At creation:

```yaml
replicas: 3   # valid
```

Six months later:

* business rules change
* minimum replicas is now 5

What happens?

Kubernetes:

> does nothing

The resource remains in the cluster:

* technically valid at creation time
* invalid by current standards

There is no signal. No warning. No correction.

---

## This is not an edge case

Over time, systems change:

* APIs evolve
* constraints tighten
* infrastructure assumptions shift
* teams reinterpret what “valid” means

But Kubernetes validation does not evolve with them.

It is **static enforcement at a single point in time**.

---

## Validation vs enforcement

This is the core distinction:

| Concept         | When it runs | What it guarantees                 |
| --------------- | ------------ | ---------------------------------- |
| **Validation**  | At admission | Resource is valid *now*            |
| **Enforcement** | Continuously | Resource remains valid *over time* |

Kubernetes gives you validation.

It does not give you enforcement.

---

## The missing layer

To maintain correctness over time, you need:

* continuous evaluation
* visibility into violations
* the ability to react

This is not a webhook problem.

It is a **runtime problem**.

---

## How Orkestra approaches this

Orkestra treats validation as part of reconciliation — not just admission.

Every resync cycle:

* the resource is evaluated
* validation rules are applied
* mutations (if defined) can be enforced
* results are recorded

This turns validation into a continuous process.

---

## What this looks like in practice

Instead of a one-time decision:

> allow or deny

You get a live system view:

```json
{
  "admission": {
    "validationAllowed": 47,
    "validationDenied": 0,
    "validationWarned": 3,
    "validationTotal": 50
  }
}
```

This answers questions Kubernetes cannot:

* How many resources are drifting?
* Which rules are being violated over time?
* Are mutations correcting state or masking problems?

Validation becomes observable.

---

## Why this matters

### 1. Drift becomes visible

Without continuous validation:

* invalid resources accumulate silently

With it:

* you see them immediately

---

### 2. Systems become self-correcting

Mutation rules can:

* fix non-compliant resources
* enforce defaults retroactively
* align old objects with new standards

---

### 3. APIs can evolve safely

You no longer need to rely entirely on:

* version bumps
* conversion logic

You can:

* introduce stricter rules
* let the system converge over time

---

## The three layers of validation

Not all validation is the same.

The right model is layered:

---

### 🟢 Use CRD schema validation when:

* you need simple field validation
* you define defaults
* you follow standard Kubernetes patterns

This is:

* fast
* native
* always recommended

---

### 🟡 Use admission webhooks when:

* you need strict enforcement at write time
* you want to block invalid resources immediately

This is:

* powerful
* but operationally heavier

---

### 🔵 Use continuous validation when:

* you care about correctness over time
* you want visibility into drift
* you need system-wide guarantees

This is where Orkestra operates.

---

## Why Kubernetes doesn’t do this

Kubernetes is an API server, not a policy engine.

It:

* validates inputs
* stores state
* exposes resources

It does not:

* continuously reinterpret correctness
* enforce evolving constraints

That responsibility has always been left to controllers.

---

## Orkestra’s view

Orkestra extends the controller model:

> validation is just another form of reconciliation

If a controller ensures:

* Deployments match desired state

It can also ensure:

* CRDs match valid state

Same loop. Same model.

Different outcome.

---

## The deeper shift

This changes how you think about APIs.

From:

> validity is decided once

To:

> validity is continuously maintained

---

## Final thought

Kubernetes made validation declarative.

But it stopped at admission.

That was enough when systems were small.

It is not enough at scale.

---

**Validation tells you a resource was correct.**
**Enforcement ensures it stays correct.**

Orkestra brings that missing layer into the runtime.

---
