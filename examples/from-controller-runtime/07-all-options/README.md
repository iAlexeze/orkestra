# 07 — All Options

This example assumes you have completed the earlier migration steps:

- [00 — Baseline](../00-controller-runtime-baseline/README.md) — the controller-runtime starting point
- [01 — Declarative](../01-declarative/README.md) — zero Go
- [02 — Hybrid](../02-hybrid/README.md) — declarative + Go service hook
- [03 — Hooks only](../03-hooks-only/README.md) — all resources in Go
- [04 — Constructor migration](../04-constructor-migration/README.md) — Go owns the full reconcile loop
- [05 — Constructor Orkestra resources](../05-constructor-orkestra-resources/README.md) — constructor with Orkestra helpers


All five WebApp operator patterns running in a single Orkestra runtime. Each was published independently via `ork push` — the Komposer pulls them from OCI and unifies them into one binary.

This is the practical answer to: "Which option should I choose?" All five are running. Pick the one that fits your team.

> **Before you start:** This assumes you have completed options 01–05 and published each to your registry with `ork push`. Update the OCI references in [komposer.yaml](komposer.yaml) to match your registry org.

| Option | CRD Kind | Pattern |
|--------|----------|---------|
| 01 | `DeclarativeApp` | Pure YAML, zero Go |
| 02 | `HybridApp` | Declarative + Go service hook |
| 03 | `HooksApp` | Go hook, no declared templates |
| 04 | `ConstructorApp` | Go reconcile loop (direct migration) |
| 05 | `OrkApp` | Go reconcile loop (Orkestra resources) |

`hybridApp` and `hooksApp` wait for `declarativeApp` to start before activating — dependency ordering across independently published katalogs.

---

## Overriding apiTypes

Options 01–05 were each published with the same GVK: `kind: WebApp`, group `migration.demo.orkestra.io`. That was the right choice for their individual context — small, self-contained examples.

Here, the Komposer overrides the `apiTypes` on each imported katalog:

```yaml
crds:
  declarativeApp:
    apiTypes:
      group: allopt.demo.orkestra.io
      kind: DeclarativeApp
      plural: declarativeapps
```

The reconcile logic, templates, hooks, and constructor inside each katalog are untouched. Only the GVK changes. Five independent patterns, each originally published as `WebApp`, now run as five distinct CRD kinds in the same cluster — without modifying or republishing any of them.

This is the override model: take a published pattern, declare it yours, give it the kind and group that fit your domain.

---

## Step 1 — Pull the katalogs

```bash
ork pull -f komposer.yaml
```

Fetches all five katalogs from their OCI registries into the local cache. Re-run whenever you update a version in [komposer.yaml](komposer.yaml).

---

## Step 2 — Generate the registry and entrypoint

```bash
make registry
```

Generates one registry from the Komposer — not one per katalog. Re-run whenever the Komposer changes.

---

## Step 3 — Validate with the stock binary (expected failure)

```bash
ork validate -f komposer.yaml
```

You will see:

```
CRD "constructorApp": no constructor registered
```

Expected. The stock `ork` binary cannot run typed extensions. Options 02–05 require your custom binary.

---

## Step 4 — Build your custom binary

```bash
make clean
make build
```

This replaces the default `ork` binary in `~/.orkestra/bin/ork`, which is already on your PATH from the initial install. `ork` now knows about all five operator patterns.

---

## Step 5 — Validate with your custom binary

```bash
ork validate -f komposer.yaml
```

You should see all five CRDs valid:

```
● declarativeApp   kind: DeclarativeApp  mode: dynamic
● hybridApp        kind: HybridApp       mode: typed
● hooksApp         kind: HooksApp        mode: typed
● constructorApp   kind: ConstructorApp  mode: typed
● orkApp           kind: OrkApp          mode: typed

5 CRDs valid
```

---

## Step 6 — Run the runtime

```bash
ork run
```

Start the Control Center:

```bash
ork control
# username:password → orkestra
# → http://localhost:8081
```

Initially all five CRDs show as degraded — no CRDs applied yet.

---

## Step 7 — Apply CRDs and watch dependencies resolve

Apply all five CRDs — `declarativeApp` activates first:

```bash
kubectl apply -f crds/
```

Control Center: `declarativeApp` activates. `hybridApp` and `hooksApp` are now unblocked. All five activate.

---

## Step 8 — Apply CRs and compare behavior

```bash
kubectl apply -f crs/
```

Control Center shows five independent reconcile loops — each CRD with its own queue depth, worker utilization, and health state.

Compare what each option created:

```bash
kubectl get deployments
kubectl get services
kubectl get declarativeapps
kubectl get hybridapps
kubectl get hooksapps
kubectl get constructorapps
kubectl get orkapps
```

All five produce the same result: a Deployment and a Service. The difference is the degree of Go involvement in getting there.

---

## What this demonstrates

The five options are not competing approaches — they represent a spectrum of control. The Komposer unifies them into one runtime regardless of which mix your project uses:

| | 01 Declarative | 02 Hybrid | 03 Hooks only | 04 Constructor | 05 Ork resources |
|---|---|---|---|---|---|
| Go required | No | Yes (Service) | Yes (all) | Yes (full loop) | Yes (full loop) |
| Drift correction | `reconcile: true` | partial | hook-owned | you implement | `orkdeploy.Update` |
| Owner references | automatic | partial | hook-owned | you implement | automatic |
| Code to write | 0 lines | ~30 lines | ~60 lines | ~120 lines | ~60 lines |

Choose based on what your team needs to control. Orkestra delivers the infrastructure regardless of which path you take.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
