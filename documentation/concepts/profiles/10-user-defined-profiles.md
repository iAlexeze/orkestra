# User-Defined Profiles

Built-in Orkestra profiles cover the common cases. User-defined profiles let your team define the cases specific to you — your network topology, your compliance requirements, your capacity tiers — and name them so that intent travels through every Katalog and Motif that imports them.

---

## Declaring profiles

Profiles are declared at the root of a Katalog or Motif alongside `spec:` and `security:`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: platform-operator

profiles:
  networkPolicies:
    - name: allow-monitoring
      description: Allow ingress from the platform monitoring namespace
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]

  resourceQuotas:
    - name: team-medium
      description: Standard allocation for a medium-sized team namespace
      hard:
        pods: "30"
        cpu: "6"
        memory: "12Gi"
        requests.cpu: "3"
        requests.memory: "6Gi"

spec:
  crds:
    namespace:
      ...
      operatorBox:
        onCreate:
          networkPolicies:
            - name: "{{ .metadata.name }}-monitoring"
              profile: allow-monitoring
          resourceQuotas:
            - name: "{{ .metadata.name }}-quota"
              profile: team-medium
```

---

## Supported profile classes

| Class | YAML key | Expands into |
|-------|----------|--------------|
| NetworkPolicy | `profiles.networkPolicies` | ingress/egress rules, policyTypes |
| ResourceQuota | `profiles.resourceQuotas` | hard limits map |
| LimitRange | `profiles.limitRanges` | limit items |
| HPA | `profiles.hpa` | minReplicas, maxReplicas, CPU target, behavior |
| PDB | `profiles.pdb` | minAvailable or maxUnavailable |
| Rolling Update | `profiles.rollingUpdate` | maxSurge, maxUnavailable |

---

## Template expressions in profile fields

Profile field values support template expressions. They are resolved at reconcile time against the live CR:

```yaml
profiles:
  resourceQuotas:
    - name: cr-sized
      hard:
        pods: "{{ .spec.maxPods }}"
        cpu: "{{ .spec.cpuLimit }}"
        memory: "{{ .spec.memLimit }}"

  hpa:
    - name: cr-scaled
      minReplicas: "{{ .spec.minReplicas | default \"2\" }}"
      maxReplicas: "{{ .spec.maxReplicas }}"
      targetCPUUtilizationPercentage: "70"
```

At `ork validate` time, fields containing `{{` are skipped — they cannot be validated statically. At reconcile time, the expression is expanded before the profile is applied.

---

## Validation

`ork validate` enforces three rules on the `profiles:` block:

1. **Non-empty name** — every profile entry must declare a `name`.
2. **Unique within class** — two `networkPolicies` entries with the same name is an error. Two entries with the same name in different classes (`resourceQuotas` and `hpa`) is fine — class is the scope boundary.
3. **Shadowing built-ins** — allowed but warned. If you declare a `networkPolicies` profile named `deny-all`, your version is used instead of Orkestra's built-in. A warning is printed at validate time so the shadowing is explicit.

---

## Resolution order

When a resource declares `profile: some-name`, Orkestra resolves it in this order:

1. User-defined profiles in the katalog `profiles:` block
2. User-defined profiles merged from imported Motifs
3. Built-in Orkestra profiles

The first match wins. Built-ins are only consulted when the name is not found in any user registry.

---

## Profiles in Motifs

A Motif can declare its own `profiles:` block. When a Katalog imports the Motif, its profiles are merged into the Katalog's registry:

```yaml
# tenant-isolation.motif.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: tenant-isolation
  version: v0.2.0

profiles:
  networkPolicies:
    - name: allow-monitoring
      ingress:
        - from:
            - namespaceSelector:
                team: platform
      policyTypes: [Ingress]
    - name: allow-internal
      ingress:
        - from:
            - namespaceSelector:
                scope: internal
      policyTypes: [Ingress]

resources:
  networkPolicies:
    - name: "{{ .metadata.name }}-deny-all"
      profile: deny-all
    - name: "{{ .metadata.name }}-monitoring"
      profile: allow-monitoring
```

### Conflict detection

If the same profile name appears in the same class in both the Katalog and an imported Motif — or in two imported Motifs — it is a **hard error** at load time:

```
profile conflict: networkPolicies "allow-monitoring" defined in both motif "tenant-isolation" and the katalog
```

The same name in different classes is not a conflict — `resourceQuotas.medium` and `hpa.medium` are independent.

---

## ork validate output

When a Motif declares profiles, `ork validate` shows them alongside resources:

```
● tenant-isolation
  Reusable network isolation motif
  version  : v0.2.0
  inputs   : 2
  resources: networkPolicies(1)
  profiles : networkPolicies(2)
```

---

→ Back: [09 — NetworkPolicy Profile](09-networkpolicy-profile.md) | [Profiles index](index.md)
