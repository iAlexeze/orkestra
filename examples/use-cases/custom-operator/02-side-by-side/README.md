# 02 — Side-by-Side (Migration Parity)

The same `AppCert` CRD tested two ways using the same underlying infrastructure
(cert-manager). Identical assertions in both files. When both pass, the two
implementations are proven equivalent.

**What you learn:** How to use `customOperator: true` for migration parity testing.
`e2e-orkestra.yaml` uses Orkestra's katalog to compose cert-manager via
`customResources`. `e2e-custom.yaml` uses cert-manager directly. Same inputs,
same assertions, same output — two implementations proven equivalent by e2e.

---

## The two approaches

**Orkestra side (`e2e-orkestra.yaml`):**
- Orkestra katalog wraps cert-manager — an `AppCert` CR triggers creation of a
  namespaced `Issuer` + `Certificate` as child custom resources
- Orkestra manages the lifecycle; cert-manager does the actual certificate work
- Result: `my-app-tls` Secret created

**Custom side (`e2e-custom.yaml`):**
- `customOperator: true` — cert-manager is the operator, no Orkestra
- The test applies the `AppCert` CR (tracked as the CR under test) then manually
  creates the cert-manager resources via a command assertion
- Same result: `my-app-tls` Secret created

Both sides use `setup.helm` to install cert-manager. Both assert the same Secret.

---

## Steps

### 1. Run the Orkestra side

```bash
ork e2e -f e2e-orkestra.yaml
```

### 2. Run the custom side

```bash
ork e2e -f e2e-custom.yaml
```

Both should produce 3/3 passing. The assertions are identical — the difference is
what manages the cert-manager resources.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## E2E

```bash
ork e2e -f e2e-orkestra.yaml    # Orkestra composes cert-manager
ork e2e -f e2e-custom.yaml      # cert-manager directly, ork e2e as test harness
```

| Expectation | What it checks |
|-------------|----------------|
| AppCert CR created | `my-app` AppCert exists in default namespace |
| TLS Secret issued | `my-app-tls` Secret created by cert-manager |
| Cleanup verified | Secret removed after CR delete |
