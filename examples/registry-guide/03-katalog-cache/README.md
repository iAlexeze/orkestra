# 03 — Katalog: Cache operator — push gate mechanics

This example does not deploy anything. It teaches how the simulate gate protects consumers, what a gate failure looks like, how `--force` publishes a draft anyway, and how `ork inspect` surfaces the quality signal so consumers know exactly what they are getting.

---

## Files

| File | Purpose |
|------|---------|
| `crd.yaml` | Cache CRD schema |
| `katalog.yaml` | Katalog that imports the data-store motif |
| `cr.yaml` | Sample CR — Redis 7, 5Gi storage |
| `simulate.yaml` | Gate: includes a failing assertion to demonstrate gate behavior |

---

> **Before you start:** `ORK_MOTIFS_REGISTRY` and `ORK_REGISTRY` must be set from step 01. If you are starting here directly, export them now (see [01-motifs](../01-motifs/README.md#push-to-the-registry)).
>
> [katalog.yaml](katalog.yaml) imports the data-store motif from a local path for development. Before running `ork push`, update the import to the OCI ref you pushed in step 01:
> ```yaml
> # katalog.yaml
> imports:
>   - motif: oci://ghcr.io/myorg/motifs/data-store:v1.0.0
> ```

---

## The gate — what simulate protects

When you run `ork push`, the simulate gate runs automatically before the artifact lands in the registry. If any assertion fails, the push is blocked. The gate result — passed, skipped, or failed — is written as an OCI annotation into the artifact. Consumers read it via `ork inspect` before they pull.

This means the quality signal is not a promise you make separately — it travels inside the artifact, immutable, at the version consumers inspect.

---

## The failing assertion

Open [simulate.yaml](simulate.yaml). It asserts that a ConfigMap named `my-cache-config` is created in cycle 1:

```yaml
- cycle: 1
  verb: create
  resource: configmaps
  name: my-cache-config     # planned: connection info ConfigMap — not yet in motif
```

The data-store motif does not create this ConfigMap. The assertion is for a resource that was planned but not yet implemented. Run simulate to see the gate fail:

```bash
ork simulate
```

You will see something like:

```text
  Cycle 1:
    + statefulsets/my-cache
    + services/my-cache-headless
    + services/my-cache-svc
    ~ status/my-cache

  ✗ configmaps/my-cache-config not found (expected cycle 1 create)

  3/4 assertions passed, 1 failed
  FAIL
```

The gate caught it. If you try to push now, it is blocked:

```bash
ork push .
# ✗ Simulate gate failed — push blocked
#   Run 'ork simulate' to see the failures
#   Use --force to override (recorded in the artifact)
```

---

## Force push a draft

You need to share this katalog with a teammate for feedback before the ConfigMap is implemented. Use `--force` to bypass the gate and `--no-e2e` to skip the cluster test:

```bash
ork push --force --no-e2e .
```

You will see:

```text
Pushing cache-operator:v1.0.0 (Katalog) to ghcr.io/myorg/katalogs...
  ✓ katalog.yaml          valid
  ✓ crd.yaml              valid
  ✓ cr.yaml               (...)
  ✓ simulate.yaml         (...)
  ~ Simulate skipped
  ~ E2E skipped
  ⠋ Pushing cache-operator:v1.0.0...

✓ Pushed: oci://ghcr.io/myorg/katalogs/cache-operator:v1.0.0
  Digest: sha256:...
```

The artifact was published. But the gates were not passed — that fact is now baked into the artifact.

---

## Inspect the artifact — quality signals travel with it

```bash
ork inspect cache-operator:v1.0.0
```

```text
cache-operator:v1.0.0
  Kind:        Katalog
  Simulate:    ⊘ Skipped
  E2E:         ⊘ Skipped
  ...
```

A consumer who inspects this before pulling sees immediately that neither gate was run. They can make an informed decision — pull a draft for review, or wait for a version with verified gates.

This is the contract: the simulate assertions are not just your test — they are the behavioral proof you publish to every consumer. A pattern with `✓ Verified · N assertions` is one the author proved. A pattern with `⊘ Skipped` is one you should verify yourself before importing.

---

## Fix it before shipping

Remove the failing assertion (or implement the ConfigMap in the motif), then push cleanly:

```bash
ork push .
```

Now simulate runs, passes, and the artifact carries `✓ Verified` — a quality signal consumers can trust.

---

## Next step

→ [04-katalog-platform](../04-katalog-platform/README.md) — admission policies via the platform-admission motif; e2e fails while simulate passes
