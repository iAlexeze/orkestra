# 12 — CI automation with GitHub Actions

The capstone. Everything you have built in this guide — a declarative operator and a typed one — now ships automatically from a single `git push`. No manual `ork push`, no local Docker builds. A version tag triggers the pipeline; the registry gets the artifact.

Two workflows are provided. One for each producer type:

| Workflow | Pattern | Trigger |
|----------|---------|---------|
| `publish.yml` | `webapp-operator` — declarative | `git tag v*` |
| `publish-typed.yml` | `database-operator` — typed Go hooks | `git tag v*` |

See [`orkspace/orkestra-action`](https://github.com/orkspace/orkestra-action) for all available inputs and outputs.

---

## Files

```text
12-ork-action/
├── .github/
│   └── workflows/
│       ├── publish.yml          ← declarative pipeline
│       └── publish-typed.yml    ← typed operator pipeline
├── webapp-operator/             ← from 02-katalog-api
│   ├── katalog.yaml
│   ├── crd.yaml
│   └── ...
└── database-operator/           ← from 10-hooks-katalog
    ├── katalog.yaml
    ├── crd.yaml
    └── ...
```

---

> **Before you start:** Replace `myorg` with your actual GitHub username or organisation throughout this example. The workflows derive the registry path from `github.repository_owner` automatically — no extra secrets required for GHCR.

---

## Set up

**1. Create a GitHub repository and connect it**

From this directory:

```bash
git init
git remote add origin https://github.com/myorg/ork-automation.git
```

**2. Ensure versions match**

The tag you push becomes the artifact version. Both `katalog.yaml` files should have `metadata.version` set to match:

```yaml
metadata:
  name: webapp-operator
  version: v1.1.2
```

If the tag and `metadata.version` differ, the push step fails with a version mismatch error and nothing is published.

**5. Commit and push**

```bash
git add .
git commit -m "initial"
git push -u origin main
```

---

## Release

Tag and push to trigger both pipelines:

```bash
git tag v1.1.2
git push origin v1.1.2
```

Both workflows start in parallel. You can watch them at https://github.com/myorg/ork-automation/actions.

---

## What each pipeline does

**Declarative (`publish.yml`)**

```text
checkout
  → template      validate output offline
  → validate      check CRD, RBAC, operatorBox
  → simulate      assert resources created on cycle 1
  → ork push      publish webapp-operator:v1.1.2 to ghcr.io
```

**Typed (`publish-typed.yml`)**

```text
checkout
  → generate-registry      write zz_generated_typeregistry.go
  → build custom ork       full CLI binary with type registry compiled in
  → build runtime binary   production binary (-tags runtime) for Docker image
  → docker build+push      publish runtime image to ghcr.io
  → [same as declarative]
  → template               validate output offline
  → validate               check CRD + typed hooks registration
  → simulate               assert StatefulSet + Service + CronJob on cycle 1
  → ork push               publish database-operator:v1.1.2 to ghcr.io
```

The katalog is always published after the image. Consumers can always pull both atomically.

---

## Confirm

Once both pipelines are green:

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork inspect webapp-operator:v1.1.2
ork inspect database-operator:v1.1.2
```

Expected for the typed one:

```text
  Simulate:    ✓ Verified · 3 assertions
  Typed:       ✓ hooks · requires custom runtime
  Runtime:     v0.7.6
```

`Simulate: ✓ Verified` tells you the CI gate ran and passed. `Typed: ✓ hooks` tells you the hooks annotation was written. `Runtime: v0.7.6` is the Orkestra version the operator was built against.

---

## Consumers

Both patterns are now referenceable by tag:

```yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.1.2
    - oci://ghcr.io/myorg/katalogs/database-operator:v1.1.2
```

That is the exact import block used in [11-typed-komposer](../11-typed-komposer/README.md). A consumer pulls, validates, simulates, and deploys without writing a single line of Go or YAML.

---

## Cleanup

Delete the `v1.1.2` tag and packages from GitHub if you no longer need them:

```bash
git tag -d v1.1.2
git push origin --delete v1.1.2
```

Packages can be removed from `https://github.com/myorg?tab=packages`.
