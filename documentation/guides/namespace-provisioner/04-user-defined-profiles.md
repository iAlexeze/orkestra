# User-Defined Profiles

`03-user-defined-profiles` adds all six profile classes to the Katalog's `profiles:` block. No built-in names are used — every profile is org-owned, prefixed `org-`.

The key point is that `ork validate` treats these names as the authoritative registry. Reference `org-conservative` in `behavior.profile:` and the validator confirms the definition exists. The Katalog is self-contained: profiles declared and used in the same file.

---

## The profiles: block

```yaml
profiles:
  networkPolicies:
    - name: org-deny-all
      policyTypes: [Ingress, Egress]

    - name: org-allow-dns-egress
      egress:
        - ports:
            - port: 53
              protocol: UDP
      policyTypes: [Egress]

    - name: org-allow-monitoring
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]

  resourceQuotas:
    - name: org-medium
      hard:
        pods: "25"
        cpu: "4"
        memory: "8Gi"

  limitRanges:
    - name: org-container-defaults
      limits:
        - type: Container
          default: { cpu: 500m, memory: 512Mi }
          defaultRequest: { cpu: 100m, memory: 128Mi }

  hpa:
    - name: org-conservative
      targetCPUUtilizationPercentage: "70"
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 300

  pdb:
    - name: org-at-least-one
      minAvailable: "1"

  rollingUpdate:
    - name: org-safe
      maxSurge: "1"
      maxUnavailable: "0"
```

Each class exposes a named preset. `ork validate` reads this block and enforces that every `profile:` reference elsewhere in the Katalog points to a name defined here (or in an imported motif).

---

## How profiles are referenced

| Resource | Field | Profile |
|---|---|---|
| NetworkPolicy | `profile:` | `org-deny-all`, `org-allow-dns-egress`, `org-allow-monitoring` |
| ResourceQuota | `profile:` | `org-medium` (via template) |
| LimitRange | `profile:` | `org-container-defaults` |
| Deployment | `rollingUpdate.profile:` | `org-safe` |
| HPA | `behavior.profile:` | `org-conservative` |
| PDB | `behavior.profile:` | `org-at-least-one` |

The `org-medium` reference uses a template expression: `profile: "{{ printf \"org-%s\" .spec.tier }}"`. At validate time that field is skipped (contains `{{`); at reconcile time it expands to `org-medium` for a `tier: medium` CR and is resolved against the registry.

---

## Try it

```bash
cd 03-user-defined-profiles
ork validate
ork simulate
ork run
```

---

→ Next: [Profiles via motif](05-motif-profiles.md) | [Back to index](index.md)
