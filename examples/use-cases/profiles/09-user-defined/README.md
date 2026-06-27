# Profiles 09 — User-Defined

One CR. One Deployment. Three NetworkPolicies. Two HPAs. All profile names are declared in the Katalog's `profiles:` block — none are Orkestra built-ins.

**What you learn:** how to declare user-defined profile classes; that `ork validate` enforces references against your registry; that profile names are scoped to the Katalog and can be anything meaningful to your team.

**Contrast with 06 and HPA built-ins:** those examples use names Orkestra ships. This example owns the names — `team-conservative` and `team-allow-internal` mean exactly what this team defines, and a future reader of the Katalog finds the definition right at the top.

---

## Profiles declared in this Katalog

| Class | Name | Purpose |
|---|---|---|
| `networkPolicies` | `team-deny-all` | Block all ingress and egress |
| `networkPolicies` | `team-allow-internal` | Allow ingress from pods in the same namespace |
| `networkPolicies` | `team-allow-dns` | Allow outbound UDP/TCP 53 for DNS |
| `hpa` | `team-conservative` | 70% CPU, scale-down one pod per minute |
| `hpa` | `team-responsive` | 50% CPU, scale-down 25% per 15 seconds |

---

## Step 1 — Validate

```bash
ork validate
```

## Step 2 — Simulate

```bash
ork simulate
```

---

## Step 3 — Start the runtime

```bash
ork run
```

---

## Step 4 — Apply the CR

In a separate terminal:

```bash
kubectl apply -f ../cr.yaml
```

Verify the resources:

```bash
kubectl get networkpolicies,hpa
```

---

## Using user-defined profiles in your own Katalog

```yaml
profiles:
  networkPolicies:
    - name: team-deny-all
      policyTypes: [Ingress, Egress]

  hpa:
    - name: team-conservative
      targetCPUUtilizationPercentage: "70"
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 300

spec:
  crds:
    mycrd:
      operatorBox:
        onCreate:
          networkPolicies:
            - name: "{{ .metadata.name }}-deny-all"
              namespace: "{{ .metadata.namespace }}"
              podSelector: {}
              profile: team-deny-all
          hpa:
            - name: "{{ .metadata.name }}-hpa"
              namespace: "{{ .metadata.namespace }}"
              scaleTargetRef:
                apiVersion: apps/v1
                kind: Deployment
                name: "{{ .metadata.name }}"
              minReplicas: "2"
              maxReplicas: "6"
              behavior:
                profile: team-conservative
```

---

## E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
