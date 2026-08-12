# Conditional Reconciliation

`when:` conditions on resources tell Orkestra *what* to create and when.
This example goes one layer deeper: `operatorBox.preReconcile:` conditions tell
Orkestra *whether to reconcile at all*.

The difference is where evaluation happens. Resource conditions are evaluated
inside the reconciler. Pre-reconcile conditions are evaluated by the
**kordinator** — Orkestra's internal coordinator that sits between the work
queue and the reconcilers.

**What you learn:** `operatorBox.preReconcile:`, the kordinator gate, presence
and absence assertions in simulate, and using `--envtest` to verify gate
behaviour against a real API server.

**Builds on:** [Conditional Resources](../conditional-resources/README.md)

---

## The kordinator

Every time a CR is created or updated, the kordinator dequeues the event and
decides whether to call the reconciler. With `preReconcile.when:` declared in the
[`Katalog`](./katalog.yaml), it
evaluates the conditions first.

When conditions pass: the reconciler runs, resources are created or updated.

When conditions fail: the item is discarded. The reconciler is **never called**.
No error is recorded. The CR sits idle — healthy, not degraded — waiting for
the next event.

The gate does not clean up. If a Deployment was already created and the gate
later fires, the Deployment stays — the reconciler is simply no longer called.
To remove resources when a condition changes, use [resource-level `when:` conditions](../conditional-resources/README.md)
on the resources themselves — those are evaluated inside the reconciler on every
cycle and handle drift correction.

---

## The setup

Two CRDs. One gated, one not.

**App** — manages a Deployment. Gated by `spec.enabled`. When `spec.enabled`
is `false`, the kordinator discards the event and the reconciler never runs.

**Route** — manages a Service. No gate. Always reconciles.

```yaml
operatorBox:
  preReconcile:
    when:
      - field: "{{ .spec.enabled }}"
        equals: "true"
```

---

## Step 1 — Validate and simulate

Validate the Katalog, then run both simulate scenarios against a real API server:

```bash
ork validate
```

Gate passes — Deployment and Service expected in cycle 1:

```bash
ork simulate -f simulate-enabled.yaml
```

Gate fires — Deployment must be absent, Service still created:

```bash
ork simulate -f simulate-gated.yaml
```

---

## Step 2 — Start the Runtime

```bash
ork run
```

---

## Step 3 — Apply the gated CR first

Apply the App CR with `spec.enabled: false`:

```bash
kubectl apply -f cr-app-disabled.yaml
```

Watch the `ork run` terminal — no log line appears for the App. The kordinator
evaluated `spec.enabled: false`, discarded the event, and never called the
reconciler.

Check that no Deployment was created:

```bash
kubectl get deployments
# No resources found.
```

---

## Step 4 — Apply the Route CR

```bash
kubectl apply -f cr-route.yaml
```

The Route has no gate. The reconciler runs immediately. Watch the `ork run`
terminal:

```
{"level":"info","crd":"demo.orkestra.io/v1alpha1, Kind=Route","resource":"default/my-route","message":"reconciled ..."}
```

Check the Service:

```bash
kubectl get services | grep my-route
# my-route   ClusterIP
```

---

## Step 5 — Enable the App

Patch the App CR to `spec.enabled: true`:

```bash
kubectl patch app my-app --type=merge -p '{"spec":{"enabled":true}}'
```

The update triggers a new event. The kordinator evaluates the gate — it now
passes — and calls the reconciler for the first time. Watch the `ork run`
terminal:

```
{"level":"info","crd":"demo.orkestra.io/v1alpha1, Kind=App","resource":"default/my-app","message":"reconciled ..."}
```

```bash
kubectl get deployments | grep my-app
# my-app   1/1
```

---

## Step 6 — Disable again

```bash
kubectl patch app my-app --type=merge -p '{"spec":{"enabled":false}}'
```

No log line. No reconcile. The Deployment stays — the gate discards the event,
the reconciler is never called, nothing changes on the cluster.

```bash
kubectl get deployments | grep my-app
# my-app   1/1   (still there — gate does not clean up)
```

---

## E2E

```bash
ork e2e -f e2e.yaml
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Previous

[Conditional Resources](../conditional-resources/README.md) — resource-level `when:` conditions that gate individual resources inside the reconciler.
