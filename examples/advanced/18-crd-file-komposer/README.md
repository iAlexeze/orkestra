# 18 — CRD File + API Type Override in Komposer

Combine both override types in a single Komposer: replace the vendor's API group
with yours (`apiTypes`) and declare the path to your CRD file (`crdFile`). The
result is a fully self-contained Komposer — hand it off and `ork run` handles
the rest.

**What you learn:** `apiTypes` + `crdFile` together in a Komposer override,
the self-contained operator pattern, and how dev-mode auto-apply interacts
with Komposer imports.

**Builds on:** [17 — API Type Override](../17-apitype-override/README.md) and
[07 — CRD File](../../intermediate/07-crd-file/README.md)

---

## The pattern

```
vendor-katalog.yaml            →  logic (reconciler, status, onCreate)
komposer.yaml (override)       →  apiTypes: platform.acme.io    ← your group
                                  crdFile:  ./crd-mine.yaml     ← your CRD
```

The vendor Katalog provides all the operator logic. The Komposer declares:
1. **Which CRD to watch** — your group, not the vendor's
2. **Where the CRD file lives** — applied automatically by `ork run` in dev mode

No manual `kubectl apply -f crd-mine.yaml` step. No modifications to the vendor
file.

---

## Steps

### 1. Validate

```bash
ork validate --file komposer.yaml
```

Expected:
```
✓ managed-service
    kind: ManagedService
    group: platform.acme.io / version: v1alpha1 / plural: managedservices
    crdFile: ./crd-mine.yaml
    mode: dynamic / workers: 2 / resync: 30s
```

### 2. Start the runtime

```bash
ork run --file komposer.yaml
```

On startup you will see:
```
INF crdFile applied crd=managed-service path=.../crd-mine.yaml
```

`crd-mine.yaml` is installed automatically. The operator begins watching
`platform.acme.io/ManagedService` immediately.

### 3. Apply a CR

```bash
kubectl apply -f cr.yaml
```

`cr.yaml` uses `apiVersion: platform.acme.io/v1alpha1` — your API surface,
not the vendor's.

### 4. Verify

```bash
kubectl get managedservices
kubectl get deployments | grep my-service
kubectl get services   | grep my-service
```

Expected (after a moment):
```
NAME         IMAGE                PHASE
my-service   nginx:stable-alpine  Running
```

---

## How it differs from 17-apitype-override

| Aspect | 17 — apitype-override | 18 — crd-file-komposer |
|--------|-----------------------|------------------------|
| CRD install | Manual: `kubectl apply -f crd-mine.yaml` | Automatic: `ork run` applies it |
| `apiTypes` override | Yes | Yes |
| `crdFile` override | No | Yes |
| Startup step count | 5 (install → validate → run → CR → verify) | 4 (validate → run → CR → verify) |

Use **18** when you want the Komposer to be the single artifact to hand off.
Use **17** when your CI pipeline handles CRD installation separately.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
