# Configuration is Deliberate

Most infrastructure tools are built around one idea:

> If the source changes, apply the change.

This works well for deployments. It breaks down for APIs.

Orkestra makes a deliberate choice: **it does not automatically re-fetch or re-apply your Katalog when a remote source changes.** That is not a missing feature. It is the point.

---

## CRDs Are Not Deployments

A Deployment is disposable. A CRD is not.

CRDs define long-lived state stored in etcd, API contracts that clients depend on, and reconciliation behavior that shapes how your cluster evolves. Changing a CRD is not shipping a new version — it is changing the rules of the system while it is running.

Auto-syncing that kind of change is dangerous:

- You can introduce breaking schema changes mid-operation
- Reconciliation logic can silently shift under a running workload
- Behavior can diverge across environments when sources update at different times

In practice, this leads to instability disguised as automation.

---

## What Orkestra Does Instead

When Orkestra starts, it:

1. Pulls all sources — files, OCI artifacts, Komposer imports
2. Merges them into a single resolved Katalog
3. Validates everything
4. Starts the runtime

And then it stops listening to sources.

No polling. No background updates. No surprise changes.

From that moment on, **the configuration is fixed. Only the cluster state evolves.**

The reconciler watches your CRs continuously and corrects drift forever. But the Katalog it reconciles against — the templates, the rules, the resource declarations — is what was resolved at startup. It does not change until you change it and restart.

---

## Why This Matters at Scale

A single Orkestra runtime can compose multiple sources:

- Public baselines from the Orkestra registry
- Internal platform Katalogs
- Security policies
- Compliance katalogs

This is not one source of truth. It is a composed system. If Orkestra auto-synced all of these:

- One bad commit to one source could corrupt the entire runtime
- Partial updates could leave the system in an inconsistent intermediate state
- Network issues could create behavioral divergence between replicas

When you manage multiple CRDs in one process, **input integrity becomes critical to system stability.** So Orkestra treats configuration as immutable during execution.

---

## Production Is the Default

Most systems treat production as a special mode. Orkestra treats every run as production.

Starting Orkestra is a deployment:

- Configuration is resolved
- State is locked in
- Reconciliation begins

Upgrading Orkestra is also a deployment:

- You update the bundle
- You restart the runtime
- You take responsibility for the change

Nothing happens implicitly. This is what makes Orkestra safe to operate — and safe to leave running.

---

## Upgrading Deliberately

Because configuration locks in at startup, upgrading a Katalog requires a restart. The restart is not a limitation. It is the handoff.

```bash
# 1. Update your Komposer or Katalog
# 2. Regenerate the bundle
ork generate bundle -f komposer.yaml -o bundle.yaml

# 3. Apply the new bundle
kubectl apply -f bundle.yaml

# 4. Restart — this is the moment the new contract takes effect
kubectl rollout restart deployment/orkestra-runtime -n orkestra-system
kubectl rollout status deployment/orkestra-runtime -n orkestra-system
```

Between step 3 and step 4, the ConfigMap has changed but the runtime has not. The old behavior continues. The restart is the explicit signal: *I am ready to serve the new contract.*

This means rollback is the same operation in reverse — reapply the previous bundle, restart. The old version is still in the registry. Nothing was deleted. The contract simply reverts.

---

## The Trade-Off

You lose:

- Automatic updates from a Git push or a registry tag bump
- Continuous sync behavior familiar from GitOps tools

You gain:

- Deterministic runtime behavior — the Katalog at startup is the Katalog forever
- Stable APIs — clients calling the API server get the same schema throughout the runtime's lifetime
- Clear deployment boundaries — every configuration change is a named, auditable event
- Safe CRD evolution — schema changes do not surprise running reconcilers mid-cycle

For systems built around APIs — not just containers — that trade is worth it.

---

## The Principle

> Reconciliation is continuous.
> Configuration is deliberate.

The reconciler watches your CRs forever and corrects drift without interruption. The Katalog it reconciles against is what you deployed, locked in at start time, unchanged until you say otherwise. These two properties together make Orkestra predictable to operate and safe to upgrade.

---

## Where this appears in practice

- **The upgrade story** — two Katalogs in the same cluster on different versions of the same Motif, each changing on its own schedule.

  **Try it**
  ```bash
  ork init --pack registry-guide
  cd registry-guide/07-upgrade
  ```

- **`ork plan`** — shows you the reconciler template diff before you apply a new bundle and restart, so you know what changes before you commit to the restart.
