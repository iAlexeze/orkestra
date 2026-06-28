# Profiles 07 — ResourceQuota

One CR. Four ResourceQuotas. Each uses a different tier profile — no pod counts or CPU values to configure.

**What you learn:** `resourceQuotas.profile`, what each tier expands to, and how to apply multiple quota presets side-by-side.

---

## Profiles at a glance

| Profile | Pods | CPU | Memory |
|---|---|---|---|
| `small` | 10 | 2 | 4Gi |
| `medium` | 20 | 4 | 8Gi |
| `large` | 50 | 8 | 16Gi |
| `xlarge` | 100 | 16 | 32Gi |

Each tier sets both `requests.*` and `limits.*` at a 1:2 ratio.

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

Verify the quotas and inspect the expanded hard limits:

```bash
kubectl get resourcequota
kubectl describe resourcequota my-service-quota-medium
```

---

## Using a profile in your own Katalog

```yaml
resourceQuotas:
  - name: "{{ .metadata.name }}-quota"
    namespace: "{{ .metadata.namespace }}"
    profile: medium   # pods: 20, cpu: 4, memory: 8Gi
    reconcile: true
```

Or drive the tier from the CR:

```yaml
profile: "{{ .spec.tier }}"
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
