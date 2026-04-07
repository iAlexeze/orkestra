---
title: "README"
weight: 14
---

# Example 3 — Komposer

A Komposer lets you **compose multiple katalog sources** into a single Orkestra runtime.  
This is how real platform teams work:

- The platform team ships CRDs as a Helm chart  
- Application teams maintain their own katalogs in files  
- A Komposer merges everything into one unified operator runtime  
- Inline overrides allow environment‑specific customization  

{{< callout type="note" >}}
This example demonstrates how Orkestra becomes the “operator of operators” — a single runtime managing CRDs from multiple teams and sources.
{{< /callout >}}

---

## What This Example Demonstrates

| Feature | Where |
|--------|-------|
| File sources | `sources.files` — loads Website and PlatformNamespace katalogs |
| Helm source | `sources.helm` — renders a local chart and extracts katalogs |
| Helm values override | `values/overrides.yaml` — environment‑specific overrides |
| Inline override | `spec.crds` — overrides the `database` CRD from the chart |
| Dependency graph | `ork template --graph` — visualize the full CRD dependency tree |

{{< callout type="tip" >}}
Komposers are declarative “bundles” of katalogs.  
They do not reconcile anything — they only assemble the final operator.
{{< /callout >}}

---

## Requirements

{{< callout type="note" >}}
This example assumes you have completed:
{{< /callout >}}

    - **Example 1 — Website**  
    - **Example 2 — Platform Namespace**  

You will also need:

- Orkestra CLI (`ork`)  
- Helm v3 (only required for Helm chart rendering)  

---

## Files

```
Komposer/
  komposer.yaml                    # The Komposer — declares all sources
  values/
    overrides.yaml                 # Helm values override for this environment
  helm-example/                    # Example Helm chart
    Chart.yaml
    values.yaml                    # Chart defaults
    templates/
      katalog.yaml                 # Renders an Orkestra Katalog from values
```

{{< callout type="tip" >}}
This structure mirrors how real teams organize platform catalogs and Helm‑based CRD bundles.
{{< /callout >}}

---

## Inspect Before Running

### Validate the merged result

```bash
ork validate --katalog komposer.yaml
# Success: Katalog is valid
```

### Preview all CRDs after merging and applying overrides

```bash
ork template --katalog komposer.yaml
```

Expected output:

```
Rendered CRDs:
  - website
  - platformnamespace
  - database     ← from Helm chart, overridden inline (workers: 4)
  - cache        ← from Helm chart
```

### JSON output — inspect the final merged state

```bash
ork template --katalog komposer.yaml --json | jq '.[].name'
# "website"
# "platformnamespace"
# "database"
# "cache"
```

Verify the inline override:

```bash
ork template --katalog komposer.yaml --json \
  | jq '.[] | select(.name == "database") | .workers'
# 4
```

### Visualize the dependency graph

```bash
ork template --katalog komposer.yaml --graph
```

Example output:

```
Dependency Graph:
website
platformnamespace
database
cache
```

{{< callout type="note" >}}
The graph is extremely useful when debugging multi‑team katalogs.
{{< /callout >}}

---

## Run It

### Step 1 — Apply all CRDs

```bash
kubectl apply -f ../website/website-crd.yaml
kubectl apply -f ../platform-namespace/platform-namespace-crd.yaml
```

### Step 2 — Start Orkestra with the Komposer

```bash
ork run --katalog komposer.yaml
```

Orkestra will:

- Resolve all sources  
- Merge katalogs  
- Apply overrides  
- Validate the final result  
- Start reconcilers for all CRDs  

### Step 3 — Apply CRs from all examples

```bash
kubectl apply -f ../website/website-cr.yaml
kubectl apply -f ../platform-namespace/platform-namespace-cr.yaml
```

### Step 4 — Verify

```bash
curl localhost:8080/katalog | jq '.crds[].name'
# "website"
# "platformnamespace"
# "database"
# "cache"

curl localhost:8080/katalog | jq '.total'
# 4
```

{{< callout type="tip" >}}
A single Orkestra runtime can manage dozens of CRDs from multiple teams — all composed declaratively.
{{< /callout >}}

---

## How the Merge Works

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
        └── database → overrides the Helm version (workers: 4)

Final result:
  [website, platformnamespace, database (workers: 4), cache]
```

{{< callout type="note" >}}
Inline overrides always win — they are applied *after* all sources are merged.
{{< /callout >}}

---

## Add More Sources

### Pull a remote katalog

```yaml
sources:
  files:
    - ../website/website-katalog.yaml
    - ../platform-namespace/platform-namespace-katalog.yaml
    - https://raw.githubusercontent.com/myorg/crds/main/katalog.yaml
```

### Use an environment variable for the remote URL

```yaml
sources:
  files:
    - $REMOTE_KATALOG_URL
```

{{< callout type="tip" >}}
This pattern is ideal for multi‑environment deployments (dev/staging/prod).
{{< /callout >}}

---

## Generate Runtime (Only When Needed)

If any CRD uses compiled Go types (`apiTypes.location` set), generate the runtime wiring:

```bash
ork generate runtime --katalog komposer.yaml
```

{{< callout type="note" >}}
This example uses only dynamic CRDs — no generation required.
{{< /callout >}}

---

If you want, I can now:

- Write **Example 4 — Drift‑Correcting Multi‑Resource Operator**  
- Write **Example 5 — Typed CRDs + Hooks**  
- Build the full Beginner → Intermediate → Advanced workflow index  

Just tell me where you want to go next.