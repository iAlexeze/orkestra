# Example 3 — Komposer

Composition. A Komposer is an orkestra resource that pulls CRD definitions from multiple sources —
Katalog files and a Helm chart — then applies inline overrides.

This is how teams work in practice: the platform team ships CRDs as a Helm
chart, the application team keeps their CRDs in their own files, and a single
Komposer composes everything into one Orkestra runtime.

---

## What this example demonstrates

| Feature | Where |
|---|---|
| File sources | `sources.files` — loads website and platform-namespace Katalogs |
| Helm source | `sources.helm` — renders a local chart and extracts CRD definitions |
| Helm values override | `values/overrides.yaml` — overrides chart defaults per environment |
| Inline override | `spec.crds` — overrides the `database` CRD from the chart with more workers |
| Dependency graph | `ork template --graph` — visualise the full CRD dependency tree |

---

## Requirements

- Completed [Example 1 — Website](../website/README.md)
- Completed [Example 2 — Platform Namespace](../platform-namespace/README.md)
- Orkestra CLI installed (`ork`)
- Helm v3 installed (only needed if rendering Helm charts)

---

## Files

```
Komposer/
  komposer.yaml                        The Komposer — declares all sources
  values/
    overrides.yaml                 Helm values override for this environment
  helm-example/                    Example Helm chart
    Chart.yaml
    values.yaml                    Chart defaults
    templates/
      katalog.yaml                 Renders an Orkestra Katalog from values
```

---

## Inspect before running

**Validate the merged result:**
```bash
ork validate --katalog komposer.yaml
# Success: Katalog is valid
```

**Preview all CRDs after merging and applying overrides:**
```bash
ork template --katalog komposer.yaml
# Success: Katalog is valid
#
# Rendered CRDs:
#   - website
#   - platformnamespace
#   - database     ← from Helm chart, overridden inline (workers: 4)
#   - cache        ← from Helm chart
```

**JSON output — see the full post-validation state:**
```bash
ork template --katalog komposer.yaml --json | jq '.[].name'
# "website"
# "platformnamespace"
# "database"
# "cache"

# Verify the inline override applied (workers should be 4, not 2)
ork template --katalog komposer.yaml --json | jq '.[] | select(.name == "database") | .workers'
# 4
```

**Dependency graph:**
```bash
ork template --katalog komposer.yaml --graph
# Dependency Graph:
# website
# platformnamespace
# database
# cache
```

---

## Run it

**Step 1 — Apply all CRDs**

```bash
kubectl apply -f ../website/website-crd.yaml
kubectl apply -f ../platform-namespace/platform-namespace-crd.yaml
```

**Step 2 — Start Orkestra with the Komposer**

```bash
ork run --katalog komposer.yaml
```

Orkestra resolves all sources, merges them, validates, and starts reconcilers
for all four CRDs in one runtime.

**Step 3 — Apply CRs from all examples**

```bash
kubectl apply -f ../website/website-cr.yaml
kubectl apply -f ../platform-namespace/platform-namespace-cr.yaml
```

**Step 4 — Verify**

```bash
curl localhost:8080/katalog | jq '.crds[].name'
# "website"
# "platformnamespace"
# "database"
# "cache"

curl localhost:8080/katalog | jq '.total'
# 4
```

---

## How the merge works

```
komposer.yaml
  │
  ├── sources.files[0] → website-katalog.yaml
  │     └── spec.crds: [website]
  │
  ├── sources.files[1] → platform-namespace-katalog.yaml
  │     └── spec.crds: [platformnamespace]
  │
  ├── sources.helm[0] → helm-example/ (rendered with values/overrides.yaml)
  │     └── spec.crds: [database, cache]
  │
  └── spec.crds (inline — merged last, wins on conflict)
        └── database → overrides the helm source database (workers: 4)

Final result: [website, platformnamespace, database (workers:4), cache]
```

---

## Add more sources

To pull CRD definitions from a remote Katalog:

```yaml
sources:
  files:
    - ../website/website-katalog.yaml
    - ../platform-namespace/platform-namespace-katalog.yaml
    - https://raw.githubusercontent.com/myorg/crds/main/katalog.yaml
```

To use an environment variable for the remote URL:

```yaml
sources:
  files:
    - $REMOTE_KATALOG_URL   # export REMOTE_KATALOG_URL=https://...
```

---

## Generate runtime before running

If any CRD in the merged result uses compiled Go types (`apiTypes.location` set),
generate the runtime wiring first:

```bash
ork generate runtime --katalog komposer.yaml
```

For this example all CRDs are dynamic (no `apiTypes.location`) so generation
is not required.