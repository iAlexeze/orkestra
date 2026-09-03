# Explicit Limits

`01-explicit` is the direct form. The person filing the claim sets the quota values themselves — pod ceiling, CPU, and memory budget — as plain fields in the CR spec.

```yaml
apiVersion: provisioner.orkestra.io/v1
kind: NamespaceClaim
metadata:
  name: team-alpha-claim
spec:
  targetNamespace: team-alpha
  team: alpha
  owner: alpha-operator
  ownerNamespace: default
  limits:
    maxPods: 20
    cpu: "4"
    memory: "8Gi"
```

The Katalog reads `spec.limits.*` directly into the ResourceQuota's `hard` map. The two motifs contribute NetworkPolicies and RBAC — the Katalog doesn't repeat that logic.

---

## Try it

```bash
cd 01-explicit
ork validate
ork simulate
```

Simulate produces seven resources in cycle 1 and reaches steady state by cycle 3. The `registry-credentials` secret is printed as a note — it requires a live cluster to copy from the `platform` namespace and is automatically skipped in simulation.

To run against a real cluster:

```bash
ork run
```

Then verify:

```bash
kubectl get ns team-alpha
kubectl get resourcequota alpha-quota -n team-alpha -o jsonpath='{.spec.hard}'
kubectl get networkpolicies -n team-alpha
kubectl get clusterrolebinding alpha-ns-admin-binding
```

---

→ Next: [Profile-based limits](03-profiles.md)
