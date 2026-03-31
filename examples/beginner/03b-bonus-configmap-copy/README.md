# 03 — Copy ConfigMap Across Namespaces

Orkestra can manage built‑in Kubernetes resources — not just CRDs you invent.  
Similar to the SecretDistribution example, this example builds a ConfigMap distribution operator: a CR that copies a ConfigMap from a platform namespace into every team namespace that needs it.

**What you learn:** Built‑in kind enrichment, `fromConfigMap`/`fromNamespace`, `toNamespaces`, `reconcile: true` keeping copies in sync with the source.

**Similar to:** [03 — Copy Secret Across Namespaces](../03-secret-copy/README.md)

---

## The pattern

A `ConfigMapDistribution` CR declares:

- Which ConfigMap to copy (`spec.configMapName`)
- Where it lives (`spec.sourceNamespace`)
- Where to copy it (`spec.targetNamespaces`)

Orkestra reads the source ConfigMap and creates copies in each target namespace, with owner references pointing to the `ConfigMapDistribution` CR. When the CR is deleted, all copies are garbage‑collected automatically.

`reconcile: true` means if the source ConfigMap changes, all copies are updated on the next reconcile cycle. The platform team manages one ConfigMap. Every team namespace stays in sync automatically.

> ConfigMapDistribution is able to distribute ConfigMaps across the cluster because it is a **cluster‑scoped CRD**.

---

## Steps

### 1. Create namespaces and the source ConfigMap

```bash
kubectl apply -f setup.yaml
```

Verify:

```bash
kubectl get configmap app-config -n platform
```

### 2. Install the CRD

```bash
kubectl apply -f crd.yaml
```

### 3. Start the operator

```bash
ork run --katalog katalog.yaml
```

### 4. Apply the CR

```bash
kubectl apply -f cr.yaml
```

### 5. Verify copies exist

```bash
kubectl get configmap app-config -n team-alpha
kubectl get configmap app-config -n team-beta
```

Both should exist and contain the same data as the source.

### 6. Verify owner references

```bash
kubectl get configmap app-config -n team-alpha -o yaml | grep -A8 ownerReferences
```

The owner is the `ConfigMapDistribution` CR — deletion of the CR triggers garbage collection of all copies.

### 7. Test sync (source change propagation)

Update the source ConfigMap:

```bash
kubectl patch configmap app-config -n platform \
  --type=merge -p '{"data":{"logLevel":"debug"}}'
```

Wait one resync interval (15s), then check a copy:

```bash
kubectl get configmap app-config -n team-alpha -o jsonpath='{.data.logLevel}'
# debug
```

### 8. Test cleanup

Delete the CR and watch copies disappear:

```bash
kubectl delete configmapdistribution app-config-distribution
kubectl get configmap app-config -n team-alpha   # gone
kubectl get configmap app-config -n team-beta    # gone
kubectl get configmap app-config -n platform     # still exists — source is untouched
```

---

## What you just built

A platform primitive that solves a real problem — distributing shared configuration to multiple namespaces — in a Katalog that any team member can read and understand. The operator handles sync, cleanup, and drift correction automatically.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

If you want, I can also generate the matching `katalog.yaml` entry for ConfigMapDistribution or the reconciler template for copying ConfigMaps.