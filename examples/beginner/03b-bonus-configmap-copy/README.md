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

### 2. Start the operator

```bash
ork run
```

Orkestra reads `crdFile: ./crd.yaml`, applies the CRD and `cr.yaml` to the cluster, and starts the operator.

### 3. Open the Control Center

In a third terminal:

```bash
ork control
# username:password → orkestra
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator.

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

Update the source ConfigMap from loglevel 'info' to 'debug':

```bash
kubectl patch configmap app-config -n platform \
  --type=merge -p '{"data":{"logLevel":"debug"}}'
```

Wait one resync interval (15s), then check a copy:

```bash
kubectl get configmap app-config -n team-alpha -o jsonpath='{.data.logLevel}' && echo
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

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies setup fixtures, starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e -f e2e.yaml
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: ConfigMap distributed to team-alpha
    after: cr-applied
    timeout: 60s
    resources:
      - kind: ConfigMap
        name: app-config
        namespace: team-alpha

  - name: ConfigMap distributed to team-beta
    after: cr-applied
    timeout: 60s
    resources:
      - kind: ConfigMap
        name: app-config
        namespace: team-beta

  - name: ConfigMaps removed on delete
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: ConfigMap
        name: app-config
        namespace: team-alpha
        count: 0
      - kind: ConfigMap
        name: app-config
        namespace: team-beta
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```