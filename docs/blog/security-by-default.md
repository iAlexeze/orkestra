Here’s a sharp, opinionated blog post that builds directly on your philosophy and the RBAC feature — written to land with real practitioners.

---

# Why Operators Are Over-Permissioned — And How We Fixed It

*Orkestra Project — March 2026*

---

## The uncomfortable truth

Most Kubernetes operators are massively over-permissioned.

Not slightly. Not accidentally.

**Structurally.**

If you inspect the RBAC of many production operators, you’ll find something like this:

```yaml
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

Cluster-wide. Full access.

And it’s not because engineers are careless.

It’s because the system makes it easier to be unsafe than correct.

---

## How we got here

The Kubernetes operator model evolved around one assumption:

> an operator is a program

You write Go code.
You use frameworks like Kubebuilder.
You scaffold RBAC alongside your controller.

Somewhere in that process, you add permissions.

Then you add more.

Then you hit a permissions error in production and do the fastest thing possible:

> widen the scope until it works

And it stays that way.

Not because it’s right — but because:

* RBAC is painful to maintain
* operators evolve over time
* no one wants to debug permissions at 2am

So teams converge on a pattern:

> **over-permission once, never touch it again**

---

## The real problem isn’t RBAC

RBAC is doing exactly what it was designed to do.

The problem is this:

> **RBAC is disconnected from intent**

Permissions are written manually.

But operators are dynamic systems:

* they reconcile different resources over time
* they evolve
* they compose with other systems

So the RBAC drifts.

And drift in security always goes in one direction:

> **more access, not less**

---

## It gets worse with scale

This problem compounds as operator count grows.

A typical cluster might run:

* 10–20 community operators
* 5–10 internal operators

Each with:

* its own RBAC
* its own assumptions
* its own blast radius

No one has a complete picture anymore.

At that point, security becomes:

> “we trust these operators”

Not:

> “we understand what they can do”

---

## The missing idea

The industry has treated RBAC as something you *write*.

But what if that’s wrong?

What if:

> **permissions should be derived, not declared**

Because the truth is:

An operator already tells you everything you need to know:

* what CRDs it watches
* what resources it creates
* what resources it updates

That *is* the permission model.

We just haven’t been using it.

---

## How Orkestra approaches this

Orkestra flips the model.

Instead of writing RBAC, you write a **Katalog** — a declarative definition of your operator:

```yaml
spec:
  crds:
    - name: website
      apiTypes:
        group: demo.orkestra.io
        version: v1
        kind: Website
      reconciler:
        onCreate:
          deployments:
            - image: "nginx"
              reconcile: true
```

From this, Orkestra knows:

* it needs access to `websites.demo.orkestra.io`
* it needs to manage `deployments`
* it may create `services`, `configmaps`, etc.

So instead of asking you to write RBAC…

It generates it.

```bash
ork generate rbac --katalog katalog.yaml
```

Result:

```yaml
- apiGroups: ["demo.orkestra.io"]
  resources: ["websites"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

Nothing more.

---

## Why this matters

### 1. Security becomes the default

You don’t need to think about RBAC to be secure.

You just define what your operator does.

The permissions follow automatically.

---

### 2. No silent privilege expansion

In traditional setups:

* dependencies change
* operators evolve
* RBAC gets patched ad hoc

With Orkestra:

* change the Katalog → permissions update deterministically

No drift.

---

### 3. Composition stays safe

Orkestra runs multiple CRDs in one runtime.

Without precise RBAC, that would be dangerous:

* one CRD could unintentionally expand access for another

Derived permissions prevent that.

Each Katalog defines a bounded capability.

---

### 4. It matches how operators actually work

Operators are not arbitrary programs.

They are:

* watchers
* reconcilers
* resource managers

Their behavior is already declarative in nature.

Orkestra simply extends that:

> if behavior is declarative, permissions should be too

---

## The deeper shift

This isn’t just about RBAC.

It’s about changing the unit of trust.

Traditional model:

> trust the operator binary

Orkestra model:

> trust the declaration of behavior

That’s a much smaller, clearer surface.

---

## Why this hasn’t been done before

Because most operator frameworks are built around code.

And once you start from code:

* permissions feel external
* RBAC becomes configuration
* drift is inevitable

Orkestra starts from declarations.

So permissions are not an afterthought.

They are a **derivative**.

---

## A better default

The industry standard today is:

> start permissive, tighten later (maybe)

Orkestra takes the opposite approach:

> start minimal, expand only when required

That’s the model Kubernetes itself uses internally.

We’re just applying it to operators.

---

## Final thought

Operators were meant to encode domain knowledge.

But somewhere along the way, they became:

* heavy
* over-privileged
* hard to reason about

Fixing that isn’t about better RBAC templates.

It’s about aligning permissions with intent.

---

**Orkestra doesn’t ask you to secure your operator.**
**It makes your operator secure by construction.**
