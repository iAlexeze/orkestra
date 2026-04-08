---
title: "Best Practices Registry"
weight: 134
---

# Registry Best Practices

The registry source is the distribution mechanism for operator patterns.
Used well, it makes your operator layer composable, auditable, and safe to
upgrade. Used carelessly, it creates invisible dependencies that break
unexpectedly. These practices exist because the registry is new — the
community is still learning what works.

---

## Pin versions in production

Tracking `main` or `latest` means your operator behavior changes whenever
the upstream pattern is updated. This is rarely what you want at runtime.

```yaml
# Development — tracking is fine, you want the latest fixes
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres@latest
  oci: true

# Production — pin to a specific version
- url: ghcr.io/orkspace/patterns/postgres@v1.0.0
  oci: true
```

Upgrade deliberately: review the pattern's changelog, test in staging, then
bump the version in the Komposer.

{{< callout type="tip" >}}
OCI version tags are immutable — `postgres:v14.2.0` cannot be
overwritten once published. This means pinned versions are stable by
design. Git branches do not have this guarantee. If you pin to a Git
branch, you are tracking, not pinning.
{{< /callout >}}

---

## Prefer `oci: false` for internal Git registries

If your registry is a GitHub or GitLab repository, you do not need OCI.
Raw file HTTP fetches the five required files individually — faster than
a clone, no ORAS dependency.

```yaml
# Git-based internal registry — oci: false is the right choice
- url: https://github.com/myorg/operator-registry@v1.0.0
  auth:
    type: github
    fromEnv: GITHUB_TOKEN
```

Use `oci: true` when you have published your patterns as OCI artifacts —
which is the standard for the public OrkestraRegistry and for teams that
want immutable, content-addressable distribution.

---

## Use `useKomposer: false` unless you have a specific reason not to

The default is correct for most consumers. `useKomposer: false` loads
`katalog.yaml` — the CRD definitions and reconcile templates — without any
of the upstream's own source declarations.

```yaml
# Good — take the definitions, override what you need
- url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
  oci: true
  # useKomposer: false ← default, no need to declare

spec:
  crds:
    postgres:
      workers: 8
```

`useKomposer: true` pulls the full upstream source tree. This is for internal
canonical registries where the upstream Komposer is exactly what all consumers
should run — no overrides needed. Before using it, read the upstream
`komposer.yaml` and understand every source it declares.

{{< callout type="warning" >}}
`useKomposer: true` means the upstream Komposer's own `sources.registry`
entries are also resolved. A Komposer that pulls from three other registries
will trigger three more pulls. Your pull graph can grow faster than expected.
{{< /callout >}}

---

## Never put credentials in YAML

The `auth` block resolves credentials from environment variables. There is
no field for a literal token. This is intentional.

```yaml
# Wrong — these fields do not exist
auth:
  type: github
  token: ghp_myactualtoken

# Correct
auth:
  type: github
  fromEnv: GITHUB_TOKEN
```

In local development, set the environment variable in your shell. In
Kubernetes, inject it from a Secret:

```yaml
env:
  - name: GITHUB_TOKEN
    valueFrom:
      secretKeyRef:
        name: orkestra-registry-creds
        key: github-token
```

---

## Validate before you run

Run `ork validate` before `ork run` whenever you change a registry source.
Validation pulls patterns, checks the five required files, merges all sources,
and surfaces every structural error before reconciliation begins.

```bash
ork validate --katalog komposer.yaml
```

{{< callout type="tip" >}}
Make `ork validate` part of your CI pipeline. A pull request that changes
a registry source or version should trigger validation in CI — not just
at deploy time.
{{< /callout >}}

---

## Build complete patterns — all five files

If you are publishing a pattern, all five files are required. This is
enforced at pull time by every consumer. Structure your pattern directory
correctly before publishing:

```
my-operator/
  v1.0.0/
    crd.yaml
    katalog.yaml
    komposer.yaml
    cr.yaml
    README.md
```

**The README matters.** It is the first thing a consumer reads. A good README:

- Explains what the CRD is and what the operator does
- Documents every spec field with type, default, and description
- Shows the minimal CR to get started
- Lists recommended overrides for production
- Links to the full CRD schema if it is complex

**The `komposer.yaml` should demonstrate overrides.** It is not enough to
show a minimal import. Show how to override workers, resync, and resource
limits — these are the overrides every team will want.

**The `cr.yaml` should work with zero configuration.** A consumer should be
able to `kubectl apply -f cr.yaml` immediately after installing the CRD and
see a successful reconcile.

{{< callout type="warning" >}}
An empty `README.md` passes the non-empty check but fails the spirit of it.
Write documentation. Your pattern will be used by teams who have never
seen your CRD before.
{{< /callout >}}

---

## Structure private registries like the public one

If your organisation maintains a private registry, follow the same five-file
pattern structure as the public OrkestraRegistry. This means:

- Internal consumers get the same consistent experience
- You can run `ork validate` locally against your patterns before pushing
- If you ever want to contribute a pattern upstream, the structure is already right
- New team members have one mental model for all registry patterns

---

## Understand what you are pulling before `useKomposer: true`

When you load an upstream Komposer, you accept the full upstream source tree.
Before enabling `useKomposer: true`, check:

1. Does the upstream `komposer.yaml` declare its own `sources.registry` entries?
   Those will also be resolved, recursively.
2. Does the upstream Komposer use `spec.crds` overrides that assume a specific
   environment? Those overrides will apply to your cluster.
3. Does the upstream Komposer pin version references or track branches?

If the answers are acceptable, `useKomposer: true` is a reasonable shortcut.
If you are unsure, load `katalog.yaml` instead and compose explicitly.

---

## Keep registry entries focused

Each registry entry should pull one pattern for one purpose. Avoid
constructing a Komposer where multiple registry entries supply CRDs that
overlap or conflict:

```yaml
# Avoid — unclear which postgres definition wins
sources:
  registry:
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
      oci: true
    - url: https://github.com/myorg/internal-registry@main
      # also contains a postgres CRD entry
```

If you need to override a registry pattern, use `spec.crds` inline — not a
second registry entry for the same pattern.

```yaml
# Correct
sources:
  registry:
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
      oci: true

spec:
  crds:
    postgres:
      workers: 8   # override here, not via a second registry source
```

{{< callout type="note" >}}
Duplicate CRD names across non-inline sources are an error. If two
registry entries both provide a CRD named `postgres`, Orkestra will
refuse to start. Use `spec.crds` for overrides. It is always unambiguous.
{{< /callout >}}

---

## Document which registry your Komposer uses

In team environments, make the registry source obvious. Use comments to
explain why a specific version was chosen and when it should be reviewed:

```yaml
sources:
  registry:
    # Pinned to v14.2.0 — upgrade reviewed 2026-03. Next review: Q3 2026.
    # Changelog: https://github.com/orkestra-sh/orkestra-registry/releases
    - url: ghcr.io/orkspace/patterns/postgres@v1.0.0
      oci: true

    # Internal CRD — maintained by platform team
    # Komposer owned by: platform@myorg.com
    - url: https://github.com/myorg/internal-registry@main
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
```

This makes version bump pull requests reviewable — the comment explains
the current state and the change is visible in the diff.
