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

### 1. Simulate (no cluster needed)

```bash
ork simulate
```

Because the Secret copy requires reading the source Secret from a live cluster, Orkestra detects this automatically and skips it during simulation:

```text
note: secrets/{{ .spec.secretName }}: cross-namespace copy skipped in simulate — requires a live cluster

  Cycle 1:
    ~ status/db-creds

  ~ Max cycles reached (1) in 245ms
```

No errors, no cluster, no guessing. Use `ork e2e` to exercise the copy against a real cluster.

---

### 2. Start the runtime

```bash
ork run       # add --dev if you don't have a cluster; Orkestra will create a kind cluster
```

Orkestra applies the CRD, `setup.yaml` (which creates the source Secret in the `platform` namespace), and `cr.yaml`, then starts the operator.

### 3. In a second terminal — verify and control

Confirm the source Secret and CR are live:

```bash
kubectl get secretdistribution db-creds
kubectl get secret database-credentials -n platform
```

Then open the Control Center:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) to see the live operator and the resources created.

### 4. Verify copies exist

```bash
kubectl get secret database-credentials -n team-alpha
kubectl get secret database-credentials -n team-beta
```

Both should exist and have the same data as the source.

### 5. Verify owner references

```bash
kubectl get secret database-credentials -n team-alpha -o yaml | grep -A8 ownerReferences
```

The owner is the `SecretDistribution` CR — deletion of the CR triggers
garbage collection of all copies.

### 6. Test sync (source change propagation)

#### Confirm the current password

```bash
kubectl get secret database-credentials -n team-alpha \
  -o jsonpath='{.data.password}' | base64 -d && echo
# supersecret
```

#### Update the source Secret:

```bash
kubectl patch secret database-credentials -n platform \
  --type=merge -p '{"stringData":{"password":"newpassword"}}'
```

#### Verify the new password

Wait one resync interval (15s), then check a copy:

```bash
kubectl get secret database-credentials -n team-alpha \
  -o jsonpath='{.data.password}' | base64 -d && echo
# newpassword
```

### 7. Test cleanup

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

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies setup fixtures, starts the operator, applies the CR, asserts every expectation, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Secret distributed to team-alpha
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Secret
        name: database-credentials
        namespace: team-alpha

  - name: Secret distributed to team-beta
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Secret
        name: database-credentials
        namespace: team-beta

  - name: Secrets removed on delete
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Secret
        name: database-credentials
        namespace: team-alpha
        count: 0
      - kind: Secret
        name: database-credentials
        namespace: team-beta
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
