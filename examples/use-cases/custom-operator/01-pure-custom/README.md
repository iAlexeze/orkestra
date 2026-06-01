# 01 — Pure Custom Operator (cert-manager)

`customOperator: true` with cert-manager installed via `setup.helm`.
`ork e2e` acts as a test harness only — no Orkestra bundle, no Orkestra install.

**What you learn:** How to use `ork e2e` to test any Kubernetes operator. cert-manager
is installed by the `setup.helm` block, and assertions verify the TLS Secret it creates.
Swap cert-manager for your own operator to get the same CI harness instantly.

---

## Steps

### 1. Run the e2e

```bash
ork e2e
```

What happens:
1. A kind cluster is created
2. cert-manager is installed via Helm (`setup.helm`)
3. The `setup.wait` block waits for cert-manager's webhook to be ready
4. `cr.yaml` is applied — a `ClusterIssuer` and a `Certificate`
5. Assertions verify cert-manager issued the TLS Secret
6. Cleanup: CR deleted, cert-manager uninstalled, cluster torn down

No bundle. No Orkestra helm install. `ork e2e` just runs the cluster lifecycle and
assertion loop.

### 2. Verify (manual)

```bash
kubectl get secret my-tls-secret -o yaml
kubectl get certificate my-tls-cert -o yaml | grep -A5 "status:"
```

Expected:
```
NAME            TYPE                DATA   AGE
my-tls-secret   kubernetes.io/tls   3      5s
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## E2E

```bash
ork e2e
```

The `e2e.yaml` uses `customOperator: true` — cert-manager is the operator under test.

| Expectation | What it checks |
|-------------|----------------|
| ClusterIssuer and Certificate created | `selfsigned-issuer` is Ready |
| TLS Secret issued by cert-manager | `my-tls-secret` Secret exists |
| Cleanup verified | Secret removed after CR delete |
