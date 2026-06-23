# 07 — All Options

All five WebApp migration patterns running in a single Orkestra runtime. Each katalog was built independently in its own example — here a Komposer unifies them.

| Option | CRD Kind | Pattern |
|--------|----------|---------|
| 01 | `DeclarativeApp` | Pure YAML, zero Go |
| 02 | `HybridApp` | Declarative + Go service hook |
| 03 | `HooksApp` | Go hook, no declared templates |
| 04 | `ConstructorApp` | Go reconcile loop (direct migration) |
| 05 | `OrkApp` | Go reconcile loop (Orkestra resources) |

`hybridApp` and `hooksApp` wait for `declarativeApp` to start before activating.

---

## Local run

### Step 1 — Generate registry and build

```bash
make registry KOMPOSER=komposer-local.yaml
make build
```

Generates a single type registry from all five local katalogs and compiles one binary.

### Step 3 — Validate

```bash
ork validate -f komposer-local.yaml
```

### Step 4 — Run

```bash
ork run -f komposer-local.yaml
```

In a second terminal, apply CRDs and CRs:

```bash
kubectl apply -f crds/
kubectl apply -f crs/
```

Open the Control Center:

```bash
ork control
```

`declarativeApp` activates first. `hybridApp` and `hooksApp` unblock once it is started. All five reconcile loops run independently — each with its own queue depth, worker utilization, and health state.

Compare what each option created:

```bash
kubectl get deployments
kubectl get services
kubectl get declarativeapps,hybridapps,hooksapps,constructorapps,orkapps
```

All five produce the same result: a Deployment and a Service. The difference is the degree of Go involvement.

### Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Distribution (OCI)

Once you are happy with each option locally, publish them so anyone can pull the Komposer without cloning the repo.

Using hybrid as the example — repeat for each option you want to distribute:

```bash
cd ../02-hybrid
git add .
git commit -m "feat: webapp hybrid operator"
git tag v1.0.0
git push origin main --tags
ork push -f katalog.yaml
```

How people will run it:#

Update [komposer.yaml](komposer.yaml) with your registry org and tags, then:

```bash
ork pull -f komposer.yaml
make registry
make build
ork run
```

The OCI imports replace the local file imports — same runtime, distributed katalogs. Make sure your GitHub Package registry is public so the OCI references are accessible to consumers.
