# 03 — Copy Secret Across Namespaces

Orkestra can manage built-in Kubernetes resources — not just CRDs you invent.
This example builds a Secret distribution operator: a CR that copies a Secret
from a platform namespace into every team namespace that needs it.

**What you learn:** Built-in kind enrichment, `fromSecret`/`fromNamespace`,
`toNamespaces`, `reconcile: true` keeping copies in sync with source.

**Builds on:** [02 — Website with Service](../02-website-with-service/README.md)

---

## The pattern

A `SecretDistribution` CR declares:
- Which Secret to copy (`spec.secretName`)
- Where it lives (`spec.sourceNamespace`)
- Where to copy it (`spec.targetNamespaces`)

Orkestra reads the source Secret and creates copies in each target namespace,
with owner references pointing to the `SecretDistribution` CR. When the CR
is deleted, all copies are garbage-collected automatically.

`reconcile: true` means if the source Secret changes, all copies are updated
on the next reconcile cycle. The platform team manages one Secret. Every team
namespace stays in sync automatically.

> [!NOTE]
> `SecretDistribution` is able to distribute secrets across the cluster because it is a **cluster-scoped CRD**.
---

## Steps

### 1. Create namespaces and the source Secret

```bash
kubectl apply -f setup.yaml
```

Verify:

```bash
kubectl get secret database-credentials -n platform
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
kubectl get secret database-credentials -n team-alpha
kubectl get secret database-credentials -n team-beta
```

Both should exist and have the same data as the source.

### 6. Verify owner references

```bash
kubectl get secret database-credentials -n team-alpha -o yaml | grep -A8 ownerReferences
```

The owner is the `SecretDistribution` CR — deletion of the CR triggers
garbage collection of all copies.

### 7. Test sync (source change propagation)

Update the source Secret:

```bash
kubectl patch secret database-credentials -n platform \
  --type=merge -p '{"stringData":{"password":"newpassword"}}'
```

Wait one resync interval (15s), then check a copy:

```bash
kubectl get secret database-credentials -n team-alpha \
  -o jsonpath='{.data.password}' | base64 -d && echo
# newpassword
```

### 8. Test cleanup

Delete the CR and watch copies disappear:

```bash
kubectl delete secretdistribution db-creds
kubectl get secret database-credentials -n team-alpha   # gone
kubectl get secret database-credentials -n team-beta    # gone
kubectl get secret database-credentials -n platform     # still exists — source is untouched
```

---

## What you just built

A platform primitive that solves a real problem — distributing credentials to
namespaces — in a Katalog that any team member can read and understand. The
operator handles sync, cleanup, and drift correction automatically.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
