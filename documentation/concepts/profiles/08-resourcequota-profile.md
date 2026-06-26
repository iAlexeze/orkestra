# ResourceQuota Profile

A ResourceQuota profile is a named preset that expands into a complete Kubernetes `ResourceQuota` hard limits block at reconcile time.

You write a profile name. Orkestra writes the pod counts, CPU, and memory limits for the namespace.

---

## Profiles

| Profile | Pods | CPU | Memory | CPU Request | Memory Request | Use case |
|---------|------|-----|--------|-------------|----------------|----------|
| `small` | 10 | 2 | 4Gi | 1 | 2Gi | Development namespaces, small teams |
| `medium` | 20 | 4 | 8Gi | 2 | 4Gi | Standard team namespaces |
| `large` | 50 | 8 | 16Gi | 4 | 8Gi | Production namespaces, larger services |
| `xlarge` | 100 | 16 | 32Gi | 8 | 16Gi | High-scale production or platform namespaces |

Each profile sets `pods`, `cpu`, `memory`, `requests.cpu`, `requests.memory`, `limits.cpu`, and `limits.memory`.

---

## Usage

Set `profile` on any `resourceQuotas` entry:

```yaml
onCreate:
  resourceQuotas:
    - name: "{{ .metadata.name }}-quota"
      profile: medium
      reconcile: true
```

Or dynamically from the CR spec:

```yaml
resourceQuotas:
  - name: "{{ .metadata.name }}-quota"
    profile: '{{ .spec.size | default "small" }}'
    reconcile: true
```

This pattern is common in namespace provisioning operators where the CR carries a `size` field that controls how much of the cluster the namespace receives.

---

## Rules

**Profile or explicit — not both.**

`profile` and `hard` cannot coexist on the same ResourceQuota entry. Orkestra ignores the profile if `hard` is also set and logs a warning.

```yaml
# Valid
resourceQuotas:
  - name: ns-quota
    profile: large

# Valid
resourceQuotas:
  - name: ns-quota
    hard:
      pods: "30"
      cpu: "6"
      memory: 12Gi

# Invalid — profile is ignored, hard wins
resourceQuotas:
  - name: ns-quota
    profile: large
    hard:
      pods: "30"
```

**Unknown profiles log a warning and skip the resource.** A typo does not cause a reconcile failure — Orkestra skips that ResourceQuota and logs the unknown profile name.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Developer sandbox, local testing | `small` |
| Team namespace, feature environment | `medium` |
| Production service namespace | `large` |
| Platform or high-traffic production | `xlarge` |
| Fine-grained control needed | Omit profile, use `hard` directly |
