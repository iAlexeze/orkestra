# Orkestra Doesn’t Auto-Sync — By Design

Most modern infrastructure tools are built around one idea:

> If the source changes, apply the change.

This works well for deployments. It breaks down for APIs.

Orkestra makes a deliberate choice:
**it does not automatically re-fetch or re-apply your Katalog when a remote source changes.**

That’s not a missing feature. It’s the point.

---

## CRDs Are Not Deployments

A Deployment is disposable.
A CRD is not.

CRDs define:

* long-lived state (stored in etcd)
* API contracts
* reconciliation behaviour

Changing a CRD isn’t “shipping a new version.”
It’s **changing the rules of the system while it’s running**.

Auto-syncing that kind of change is dangerous:

* you can introduce breaking schema changes
* reconciliation logic can silently shift
* behaviour can diverge across environments

In practice, this leads to instability disguised as automation.

---

## What Orkestra Does Instead

When Orkestra starts, it:

1. Pulls all sources (files, GitHub, APIs, Helm, OCI)
2. Merges them into a single configuration
3. Validates everything
4. Starts the runtime

And then it stops listening.

No polling.
No background updates.
No surprise changes.

From that moment on:

> The configuration is fixed.
> Only the cluster state evolves.

---

## Why This Matters

A single Orkestra runtime can combine multiple sources:

* public baselines
* internal platform CRDs
* security APIs
* compliance catalogs
* Helm charts

This is not one source of truth.
It’s a *composed system*.

If Orkestra auto-synced all of these:

* one bad commit could corrupt the entire runtime
* partial updates could leave the system inconsistent
* network issues could create drift between replicas

When you manage multiple CRDs in one process:

> Input integrity becomes critical to system stability.

So Orkestra treats configuration as **immutable during execution**.

---

## Production Is the Default

Most systems treat production as a special mode.

Orkestra treats every run as production.

Starting Orkestra is a deployment:

* configuration is resolved
* state is locked in
* reconciliation begins

Updating Orkestra is also a deployment:

* you change inputs
* you restart
* you take responsibility for the change

Nothing happens implicitly.

---

## This Is About Control

Orkestra puts a clear boundary in place:

* **Reconciliation is automatic**
* **Configuration changes are intentional**

If you want to update:

* you bump a version
* you modify a Katalog
* you restart the runtime

That’s it.

No hidden loops. No background mutation.

---

## The Trade-Off

Yes, you lose:

* automatic updates from Git
* continuous sync behaviour

But you gain:

* deterministic runtime behaviour
* stable APIs
* clear deployment boundaries
* safe evolution of CRDs

For systems built around APIs—not just containers—that trade is worth it.

---

## The Principle

Orkestra is built on a simple idea:

> Reconciliation should be continuous.
> Configuration should be deliberate.

Everything else follows.
