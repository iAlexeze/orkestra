# Profile-Based Limits

`02-profiles` removes the `spec.limits` block entirely. The squad picks a tier name and Orkestra expands the ResourceQuota at reconcile time.

```yaml
apiVersion: provisioner.orkestra.io/v1
kind: NamespaceClaim
metadata:
  name: team-beta-claim
spec:
  targetNamespace: team-beta
  team: beta
  owner: beta-operator
  ownerNamespace: default
  tier: medium
```

Four fields. No quota knowledge required from the CR author.

---

## What changes in the Katalog

The only difference from `01-explicit` is the ResourceQuota declaration:

```yaml
# 01-explicit
resourceQuotas:
  - name: "{{ .spec.team }}-quota"
    hard:
      pods: "{{ .spec.limits.maxPods }}"
      requests.cpu: "{{ .spec.limits.cpu }}"
      requests.memory: "{{ .spec.limits.memory }}"

# 02-profiles
resourceQuotas:
  - name: "{{ .spec.team }}-quota"
    profile: "{{ .spec.tier }}"
```

Orkestra expands `medium` to `pods: 20`, `requests.cpu: 4`, `requests.memory: 8Gi` before the ResourceQuota is applied. `ork validate` rejects unknown tier names at load time.

| Tier | Pods | CPU | Memory |
|------|------|-----|--------|
| `small` | 10 | 2 | 4Gi |
| `medium` | 20 | 4 | 8Gi |
| `large` | 50 | 8 | 16Gi |
| `xlarge` | 100 | 16 | 32Gi |

---

## Try it

```bash
cd 02-profiles
ork validate
ork simulate
ork run
```

After `ork run`, confirm the profile expanded correctly:

```bash
kubectl get resourcequota beta-quota -n team-beta -o jsonpath='{.spec.hard.pods}'
# 20
kubectl get resourcequota beta-quota -n team-beta -o jsonpath='{.spec.hard.requests\.cpu}'
# 4
```

---

## Run both together

From the root of the example:

```bash
cd ..
ork simulate   # runs both sub-examples
ork e2e        # provisions two kind clusters in sequence
```

---

→ Next: [User-defined profiles](04-user-defined-profiles.md) | [Back to index](index.md)
