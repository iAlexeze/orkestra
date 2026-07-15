# Intermediate Pack

Core patterns before advanced composition. Builds on beginner.

```bash
ork init --pack intermediate
cd intermediate/04-multi-resource
ork run
```

| Example | What it teaches |
|---|---|
| `04-multi-resource` | Multiple resource types from one CR — Deployment, Service, ConfigMap, and Ingress together. Resource ordering, owner references across types, status aggregated from children. |
| `05-when-conditions` | Conditional resource creation. `when:` gates resources on CR spec fields — free vs enterprise tier. Resources that do not match are never created and drift is not corrected for them. |
| `06-komposer-basic` | Your first Komposer. Two Katalogs composed into one runtime: a namespace operator and a website operator. Shows the `imports.files` pattern for local composition. |
| `07-crd-file` | Using an external CRD YAML. Decouples the schema file from the operator definition — useful when the CRD is owned by another team or published separately. |
| `08-state-machine` | A multi-step pipeline operator with no Go code. Motif-based state machine: each phase writes status, the next phase gates on it. Shows how `when:` conditions on status fields replace Go state machine code. |
| `09-notes` | Notes — the template vocabulary of a Katalog. Built-in notes for live cluster queries and fallback values (`01-built-in`), then user-defined notes declared inline and imported from Motifs via `spec.imports` (`02-user-defined`). |
