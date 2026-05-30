# Profiles 05 — PodDisruptionBudget

One CR. Three Deployments, three PDBs. Each PDB enforces a different disruption limit from a profile name — no `minAvailable` or `maxUnavailable` to calculate.

**What you learn:** `pdb.behavior.profile`, what each disruption budget expands to, and how to observe enforcement during a node drain.

---

## Profiles at a glance

| Profile | Setting | Value | Use for |
|---|---|---|---|
| `zero-downtime` | minAvailable | 100% | Stateful services, databases — no voluntary disruption allowed |
| `rolling` | maxUnavailable | 1 | Safe default — exactly one pod at a time |
| `relaxed` | maxUnavailable | 25% | Stateless services — brief capacity reduction acceptable |

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Start the operator

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **service-pdb-profiles**, then **Service**.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f ../cr.yaml
```

Verify the PDB budgets:

```bash
kubectl get pdb
```

Expected:
```
NAME                      MIN AVAILABLE   MAX UNAVAILABLE   ALLOWED DISRUPTIONS
my-service-zero-pdb       100%            N/A               0
my-service-rolling-pdb    N/A             1                 1
my-service-relaxed-pdb    N/A             25%               1
```

---

## Using a profile in your own Katalog

```yaml
pdb:
  - name: "{{ .metadata.name }}-pdb"
    selector:
      app: "{{ .metadata.name }}"
    behavior:
      profile: rolling   # maxUnavailable: 1
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
