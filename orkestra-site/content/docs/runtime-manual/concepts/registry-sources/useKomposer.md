---
title: "UseKomposer"
weight: 142
---

# The `useKomposer` field

Controls which source file is loaded from the pattern. `false` by default.

```yaml
useKomposer: false   # default — load katalog.yaml
useKomposer: true    # load komposer.yaml
```

Exactly one file is loaded. You cannot load both from a single registry
entry. This is by design.

## `useKomposer: false` (default)

Loads `katalog.yaml` from the pattern. This gives you the CRD definitions
and reconcile templates — the core of the operator — without any of the
upstream pattern's own source declarations.

This is the right choice for most consumers. You take the definitions,
override what you need inline in your own Komposer, and run.

```yaml
sources:
  registry:
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
      oci: true
      # useKomposer: false ← default, no need to declare

spec:
  crds:
    # Override workers for your environment
    postgres:
      workers: 8
```

## `useKomposer: true`

Loads `komposer.yaml` from the pattern. This means you accept the upstream
operator's full source tree — their own sources, their defaults, everything
they declared in their Komposer.

Use this when an internal team publishes a canonical Komposer that exactly
represents what all consumers should run — no overrides needed. A team
that manages the registry for a platform owns the Komposer. Consumers
pull and run it verbatim.

{{< callout type="warning" title="Understand the upstream dependency tree" >}}
A Komposer has its own `sources` block. When you load an upstream
Komposer, its sources are also resolved — and those sources may have
their own sources. This can pull more patterns than you expect.
{{< /callout >}}

    Before using `useKomposer: true`, read the upstream `komposer.yaml`
    and understand its full dependency tree.

{{< callout type="warning" title="Kind mismatch errors" >}}
If `useKomposer: false` but `katalog.yaml` contains a `Komposer` kind,
or `useKomposer: true` but `komposer.yaml` contains a `Katalog` kind,
Orkestra will error with a clear message: 

See [Error reference](./error-reference.md/#kind-mismatch).
{{< /callout >}}


This catches mismatched patterns early — a pattern that has a Komposer
where a Katalog is expected is a structural problem in the pattern, not
something Orkestra should silently accept.

