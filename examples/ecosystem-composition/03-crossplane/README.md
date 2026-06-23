# 03 — Crossplane

An `Infra` CR. That is all your team creates.

Orkestra maps it to a Crossplane Composite Claim. `size: medium` becomes `storageGB: 50`. The `compositionSelector` is built from the team label. The connection secret gets a predictable name. Crossplane picks up the Claim and provisions — your team never touches the Composite spec.

The common approach to self-service infrastructure is quotas and RBAC. Those are reactive — they assume teams will behave correctly within the bounds. The gap is that nothing ties provisioning to ownership.

With Orkestra, there are two enforcement points:

**Apply time** (gateway admission webhook) — the CR is denied before it is stored if `spec.team` is not a registered team or `spec.region` is not an approved region. The reconciler never sees it.

**Reconcile time** (always, with or without the webhook) — the reconciler re-checks the same rules on every cycle and halts on a violation. On top of that, the Crossplane Claim is only created once `spec.approved: true` is set. A team creates the `Infra` CR to document intent — it exists, it is visible in the control center, but nothing is provisioned until someone with authority patches `approved: true`.

The gate is at the resource level, not in a policy document.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — replace `database.myorg.io/v1alpha1 PostgreSQLInstance` with your actual Crossplane Composite type, and set `author:` to your name or org. Crossplane must be installed with a provider and a composition for your type. The `simulate` step works without a cluster regardless.
>
> The same `Infra → Composite Claim` pattern applies to any CRD-based provisioning tool. Strimzi, Kafka topics, VictoriaMetrics — if it exposes a CRD, Orkestra can wrap it.

---

## What the team creates

```yaml
apiVersion: ecosystem.demo.orkestra.io/v1alpha1
kind: Infra
metadata:
  name: webapp-db
spec:
  type: postgres
  size: medium
  region: us-east-1
  team: platform
```

## What Orkestra creates

```yaml
apiVersion: database.myorg.io/v1alpha1
kind: PostgreSQLInstance
metadata:
  name: webapp-db
spec:
  parameters:
    storageGB: 50
    region: us-east-1
  compositionSelector:
    matchLabels:
      provider: aws
      team: platform
  writeConnectionSecretToRef:
    name: webapp-db-conn
```

The size mapping (`small=20GB`, `medium=50GB`, `large=100GB`) lives in the Katalog. The connection secret naming convention is enforced, not documented. If the mapping changes, you update the Katalog — not every team's runbook.

---

## Step 1 — Install Crossplane

```bash
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm upgrade --install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace
```

---

## Step 2 — Simulate

```bash
ork simulate
```

The `medium` → `storageGB: 50` mapping is exercised without a cluster or Crossplane installation.

---

## Step 3 — Apply the CRDs

```bash
kubectl apply -f crd.yaml
kubectl apply -f mock-crd.yaml
```

In real Crossplane usage, `postgresqlinstances.database.myorg.io` would be registered by a `CompositeResourceDefinition` (XRD) your platform team authors. `mock-crd.yaml` skips the XRD + Composition + Provider setup so you can run the demo without a cloud account. Replace it with your org's XRD-registered Composite type when going to production.

---

## Step 4 — Run locally

```bash
ork run
```

Apply the CR — `approved: false` so no Claim is created yet:

```bash
kubectl apply -f cr.yaml
kubectl get infras
kubectl get postgresqlinstances   # none yet
```

When ready to provision, approve it:

```bash
kubectl patch infra webapp-db --type=merge -p '{"spec":{"approved":true}}'
kubectl get postgresqlinstances   # Claim now exists, Crossplane picks it up
```

---

## Step 5 — Control center

In a second terminal:

```bash
ork control
# username:password → orkestra
```

---

## Step 6 — Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

---

## Step 7 — Inspect

```bash
ork inspect infra-operator:0.1.0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[04 — Platform Stack](../04-platform-stack/README.md) — compose all four operators under one control plane.
