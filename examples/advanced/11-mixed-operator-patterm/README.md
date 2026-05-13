# 11 — Mixed Operator Pattern (Dynamic + Hooks + Constructor)

This example assumes you have completed the earlier tutorials:

- [01‑hello‑website](../../beginner/01-hello-website/README.md) — beginner, pure YAML operator
- [09‑hooks](../09-hooks/README.md) — typed hooks (Go)
- [10‑constructor](../10-constructor/README.md) — typed constructor (Go)

If not, please review them first. This chapter builds directly on those concepts.

In this example, we combine all three operator styles into a single Orkestra runtime:

- A dynamic operator (`Website`) from the beginner example
- A typed hooks operator (`Database`) from 09‑hooks
- A typed constructor operator (`Pipeline`) from 10‑constructor

All three live inside one Go module:

```go
github.com/orkspace/orkestra-mixed-operator-pattern
```

Each operator has its own katalog, but they share:

- one go.mod
- one generated runtime registry
- one generated cmd/orkestra/main.go
- one final binary

This is the recommended structure for multi‑CRD operators.

---

## Step 1 — Files already in place

This pack includes three operators:

- `01-hello-website/` — dynamic operatorBox
- `09-hooks/` — typed hooks
- `10-constructor/` — typed constructor

And one Komposer: [komposer.yaml](komposer.yaml)

The Komposer imports all three katalogs and defines their dependencies:

```yaml
spec:
  crds:
    database:
      workers: 5

    website:
      dependsOn:
        database:
          condition: healthy

    pipeline:
      dependsOn:
        website:
          condition: started
```

This means:

- Website waits for Database to be healthy
- Pipeline waits for Website to be started
- Database gets more more workers _`(from 3 to 5)`_

This is the mixed operator pattern: operators from different sources coordinating declaratively.

---

## Step 2 — Generate the registry and entrypoint

Instead of generating a registry per katalog, generate one registry from the Komposer:

```bash
make registry
```

Which runs:

```bash
ork generate registry -f komposer.yaml
```

This produces:

- `pkg/runtime/zz_generated_runtime_registry.go`
- `cmd/orkestra/main.go`

Both are regenerated whenever the Komposer changes.

---

## Step 3 — Validate with the stock binary (expected failure)

The standard `ork` CLI does not know about your typed CRDs:

```bash
ork validate -f komposer.yaml
```

You will see:

```
CRD "pipeline": no constructor registered
```

This is expected. The stock binary cannot run the new `typed` extensions.

---

## Step 4 — Build your custom Ork binary

```bash
make clean
make build
cp ~/.orkestra/bin/ork ./ork
```

Now `./ork` knows about:

- Database typed hooks
- Pipeline typed constructor
- Website dynamic operatorBox

---

## Step 5 — Validate with your custom binary

```bash
./ork validate -f komposer.yaml
```

You should now see:

- Database → typed
- Pipeline → typed
- Website → dynamic

All valid.

---

## Step 6 — Run the mixed operator runtime

Run Orkestra against the Komposer:

```bash
./ork run -f komposer.yaml --dev
```

This:

- creates a local Kind cluster _(omit `--dev` if you already have a running cluster)_
- deploys Orkestra
- loads all three CRDs _(initially missing)_
- starts the runtime
- activates operators as CRDs appear

Start the Control Center:

```bash
./ork control start
# → http://localhost:8081
```

Initially you will see:

- Database → degraded (CRD missing)
- Website → dependency issue (waiting for Database healthy)
- Pipeline → dependency issue (waiting for Website started)

This is correct. No CRDs have been applied yet.

---

## Step 7 — Apply CRDs and watch dependencies resolve

### Apply the Website CRD

```bash
kubectl apply -f 01-hello-website/crd.yaml
```

Logs:

```
activating CRD Website
informer started for Website
CRD Website activated
```

Control Center:

- Website is now started
- Website still has a dependency issue (Database must be healthy)

### Apply the Pipeline CRD

```bash
kubectl apply -f 10-constructor/crd.yaml
```

Logs show retry loops until the CRD appears, then:

```
activating CRD Pipeline
informer started for Pipeline
CRD Pipeline activated
```

Control Center:

- Pipeline dependency (website started) is satisfied
- Pipeline becomes Pending (ready to work)

### Give Pipeline work

```bash
kubectl apply -f 10-constructor/cr.yaml
```

The constructor creates three Jobs:

- build
- test
- notify

Control Center:

- Pipeline transitions to Healthy
- All Jobs appear under Resources

### Apply the Database CRD

```bash
kubectl apply -f 09-hooks/crd.yaml
```

Logs:

```
activating CRD Database
informer started for Database
CRD Database activated
```

Control Center:

- Database is started
- Website still has dependency issue (needs Database healthy)

### Give Database work

```bash
kubectl apply -f 09-hooks/cr.yaml
```

The typed hooks create:

- StatefulSet
- Service
- CronJob

Database becomes Healthy.

Control Center:

- Website dependency is now satisfied
- Website transitions to Pending

### Give Website work

```bash
kubectl apply -f 01-hello-website/cr.yaml
```

The dynamic operatorBox creates the Deployment.

Website becomes Healthy.

---

## What this demonstrates

Three operators from three different sources:

- dynamic operatorBox
- typed hooks
- typed constructor

All run inside one Orkestra runtime, one binary, one process.

Dependencies declared in the Komposer are enforced at runtime:

- Pipeline waits for Website to start
- Website waits for Database to be healthy

Operators activate as soon as their CRDs appear.  
Operators become healthy only after reconciling at least once.  
Partial failure is isolated: Pipeline ran successfully even while Database and Website were degraded.

This is the mixed operator pattern in practice.

---

## Cleanup

```bash
kind delete cluster --name orkestra-playground
```

---

## Next steps

- Explore cross‑operator IPC
- Add autoscaling to one of the CRDs
- Publish the katalogs to an OCI registry
- Compose them from registry sources instead of local files
