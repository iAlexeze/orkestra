# Rolling Update Profile

A rolling update profile is a named preset that expands into `maxSurge` and `maxUnavailable` values on a Deployment, StatefulSet, or ReplicaSet rolling update strategy at Katalog load time.

You write a name. Orkestra writes the rollout strategy.

---

## Profiles

| Profile | maxSurge | maxUnavailable | Use for |
|---------|----------|----------------|---------|
| `safe` | 1 | 0 | Zero capacity drop during rollout. Production default. |
| `fast` | 25% | 25% | Faster rollout with brief capacity reduction. |
| `blue-green` | 100% | 0 | Full duplicate capacity during rollout, then removes old pods. |

**`safe`**: Adds one new pod before removing an old one. Capacity never drops below 100%. Slower rollout.

**`fast`**: Kubernetes defaults. Removes and adds pods simultaneously. Briefly runs at 75% capacity. Balances speed and availability.

**`blue-green`**: Doubles capacity during rollout — all new pods come up before any old pods are removed. Most expensive but provides a clean cutover with no traffic disruption. Requires enough cluster capacity to run 2× the desired replica count.

For **StatefulSets**, `maxSurge` does not apply — only `maxUnavailable` is used. `safe` maps to `maxUnavailable: 0` (one at a time, ordered), `fast` to `maxUnavailable: 25%`, `blue-green` to `maxUnavailable: 0`.

---

## Usage

Set `rollingUpdate.profile` on any Deployment, StatefulSet, or ReplicaSet:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      port: "{{ .spec.port }}"
      rollingUpdate:
        profile: safe
```

StatefulSet:
```yaml
onCreate:
  statefulSets:
    - name: "{{ .metadata.name }}-db"
      image: "{{ .spec.image }}"
      rollingUpdate:
        profile: safe
```

---

## Rules

**Profile or explicit — not both.**

`rollingUpdate.profile` cannot coexist with `rollingUpdate.maxSurge` or `rollingUpdate.maxUnavailable`. Orkestra rejects the Katalog at load time if both are present.

```yaml
# Valid
rollingUpdate:
  profile: safe

# Valid
rollingUpdate:
  maxSurge: "1"
  maxUnavailable: "0"

# Invalid — rejected at load time
rollingUpdate:
  profile: safe
  maxSurge: "2"
```

**Unknown profiles fail fast.** Profile names are case-insensitive.

**No `rollingUpdate` declared** means the resource uses Kubernetes defaults (`maxSurge: 25%, maxUnavailable: 25%` for Deployments; `OnDelete` for StatefulSets managed by Orkestra).

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Production service — availability non-negotiable | `safe` |
| Internal service — speed matters more than brief dip | `fast` |
| Zero-downtime release with strict traffic cut | `blue-green` |
