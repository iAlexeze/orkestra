# Profiles 06 — NetworkPolicy

One CR. Five NetworkPolicies. Each uses a different built-in profile — no ingress or egress rules to write.

**What you learn:** `networkPolicies.profile`, what each preset expands to, and how to layer policies by applying multiple profiles from a single CR.

---

## Profiles at a glance

| Profile | policyTypes | Rules |
|---|---|---|
| `deny-all` | Ingress, Egress | Empty ingress and egress — blocks all traffic |
| `deny-all-ingress` | Ingress | Empty ingress — blocks all inbound, egress unrestricted |
| `deny-all-egress` | Egress | Empty egress — blocks all outbound, ingress unrestricted |
| `allow-same-namespace` | Ingress | Ingress from any pod in the same namespace |
| `allow-dns-egress` | Egress | Egress to UDP/TCP 53 — DNS resolution only |

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

Verify the policies:

```bash
kubectl get networkpolicies
```

Expected:
```text
NAME                         POD-SELECTOR   AGE
my-service-deny-all          <none>         5s
my-service-deny-ingress      <none>         5s
my-service-deny-egress       <none>         5s
my-service-allow-same-ns     <none>         5s
my-service-allow-dns         <none>         5s
```

Inspect the expanded rules for any policy:

```bash
kubectl get networkpolicy my-service-allow-dns -o jsonpath='{.spec}' | jq
```

---

## Using a profile in your own Katalog

```yaml
networkPolicies:
  - name: "{{ .metadata.name }}-deny-all"
    namespace: "{{ .metadata.namespace }}"
    podSelector: {}
    profile: deny-all
    reconcile: true
  - name: "{{ .metadata.name }}-allow-dns"
    namespace: "{{ .metadata.namespace }}"
    podSelector: {}
    profile: allow-dns-egress
    reconcile: true
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
