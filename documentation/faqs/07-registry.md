# Registry

## How do I publish a Katalog to the registry?

Create a pattern directory with a `katalog.yaml` and push it:

```bash
ork push postgres:v14 ./patterns/postgres/
```

The minimal directory is just `katalog.yaml`. In practice, include a `simulate.yaml` and `e2e.yaml` — they gate the push automatically and bake a quality signal into the artifact that consumers can see before importing.

```text
postgres/
  katalog.yaml    # required
  crd.yaml        # recommended — CRD manifest for consumers who don't have it
  cr.yaml         # recommended — sample CR
  simulate.yaml   # runs before e2e, blocks push if assertions fail
  e2e.yaml        # runs against a real kind cluster, blocks push if expectations fail
  README.md       # shown in registry UI
```

→ [Katalog patterns in the registry](../orkestra-registry/02-katalogs.md)

---

## How do I publish a Motif?

Same command — Motifs push to `ORK_MOTIFS_REGISTRY` instead of `ORK_REGISTRY`:

```bash
ork push security-baseline:v1 ./motifs/security-baseline/
```

If a pattern directory contains both a `katalog.yaml` and a `motif.yaml`, one push publishes both — the Katalog to the Katalog registry and the Motif to the Motif registry.

→ [Motif patterns in the registry](../orkestra-registry/01-motifs.md)

---

## How do simulate and e2e gate publication?

When `simulate.yaml` and `e2e.yaml` are present in a pattern directory, `ork push` runs them automatically — simulate first, then e2e — and blocks publication if either fails.

```text
ork push postgres:v14 ./patterns/postgres/
  → simulate runs  (< 1s, no cluster)
  → e2e runs       (kind cluster, 2–5 min)
  → artifact pushed with quality annotations
```

The gates are ordered deliberately. Simulate is instant — it catches template errors, wrong resource names, and wrong cycle order without spinning up a cluster. E2E runs after because there is no point provisioning a cluster when the reconciler is already broken.

The result is baked into the OCI artifact:

```text
io.orkestra.simulate.status   passed | no-assertion | skipped
io.orkestra.e2e.status        passed | skipped | forced
```

`ork patterns` shows an `E2E` column. `ork inspect` shows both simulate and e2e status. Consumers can see what quality guarantees a pattern carries before they import it.

To skip individual gates:

```bash
ork push postgres:v14 ./patterns/postgres/ --no-simulate   # skip simulate only
ork push postgres:v14 ./patterns/postgres/ --no-e2e        # skip e2e only
ork push postgres:v14 ./patterns/postgres/ --force         # skip both
```

Skipping is recorded in the annotations — consumers can see it.

→ [How simulate gates publication](../orkestra-registry/05-simulate.md#how-it-gates-publication)  
→ [How e2e gates publication](../orkestra-registry/04-e2e.md#how-it-gates-publication)

---

## Can I use a private registry?

Yes — point Orkestra at any OCI-compatible registry:

```bash
export ORK_REGISTRY=oci://ghcr.io/myorg/katalogs
export ORK_MOTIFS_REGISTRY=oci://ghcr.io/myorg/motifs
```

Push and pull work identically. Credentials come from `~/.docker/config.json` — run `docker login` before pushing.

---

---

## How do I sign a pattern?

Push first, then sign:

```bash
ork push postgres:v14 ./patterns/postgres/
ork pattern sign postgres:v14
```

Or in one step:

```bash
ork push postgres:v14 ./patterns/postgres/ --sign
```

Signing uses Cosign keyless — no keys, no secrets. The ambient OIDC token in CI
is the credential. In GitHub Actions this is the workflow identity; locally it
opens a browser-based OIDC flow.

After signing, `ork inspect` shows the signer on the `Signed:` row without any
extra flags.

→ [Artifact Signing](../security/10-artifact-signing.md)

---

## How do I test signing without a real registry?

Use `--sign-local` or `ork pattern sign --local`. This pushes to
[ttl.sh](https://ttl.sh) — a free ephemeral OCI registry — signs it, and prints
the exact verify and inspect commands:

```bash
# From push
ork push postgres:v14 ./ --sign-local --ttl 24h

# Or directly
ork pattern sign postgres:v14 --local --dir ./patterns/postgres/ --ttl 24h
```

```text
✓ Pushed  oci://ttl.sh/ork-a3f9c2/postgres:24h
✓ Signed

  Expires in   24h
  Verify:      ork pattern verify oci://ttl.sh/ork-a3f9c2/postgres:24h --no-tlog
  Inspect:     ork inspect oci://ttl.sh/ork-a3f9c2/postgres:24h
```

Use `--no-tlog` when verifying ephemeral artifacts — the Rekor transparency log
is not useful for a short-lived test artifact.

---

## How do I require consumers to verify the signature?

Add a `publish:` block to `katalog.yaml`:

```yaml
publish:
  signing:
    verify: true
    expectedIdentities:
      - github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
```

With `verify: true`, consumers who run `ork pull --verify` are blocked unless a
valid signature from one of the listed identities is present.

→ [`publish:` schema](../reference/schema/02-katalog/23-publish.md)

---

## How do I mark a pattern as deprecated?

Set `lifecycle.maturity: deprecated` and fill in `lifecycle.deprecation`:

```yaml
lifecycle:
  maturity: deprecated
  deprecation:
    migratedTo: ghcr.io/myorg/katalogs/webapp@v2.0.0
    message: "Migrate to v2.0.0 before 2027-01-01."
    timeline:
      from: "2026-01-01"
      to: "2027-01-01"
```

Then push normally — `ork push` creates a new artifact with the deprecation metadata baked in. The previous version is not modified.

→ [Lifecycle guide](guides/registry/10-lifecycle.md)

---

## What does maturity: alpha mean for consumers?

`ork validate` prints a non-fatal warning when a Katalog or an imported pattern carries `maturity: alpha` or `maturity: beta`. The warning does not block validation — it is informational.

To acknowledge the warning on a Komposer, list the pattern in `lifecycle.accept.patterns`:

```yaml
lifecycle:
  accept:
    patterns:
      - name: cache-operator
        author: myorg
```

---

## How do I suppress lifecycle warnings in ork validate?

Warnings for deprecated or pre-stable imports are suppressed by declaring acceptance on the Komposer — not by a flag, and not on the Katalog itself.

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator
        author: myorg
```

Each `patterns` entry covers all lifecycle concerns for that import — deprecated, alpha, beta.

→ [Lifecycle — Accept](guides/registry/10-lifecycle.md#accept--komposer-level)

---

## Further reading

- **[Orkestra Registry](../orkestra-registry/index.md)** — full publishing and pulling reference
- **[simulate gates](../orkestra-registry/05-simulate.md)** — how simulate quality signals work
- **[e2e gates](../orkestra-registry/04-e2e.md)** — how e2e quality signals work
- **[Artifact Signing](../security/10-artifact-signing.md)** — keyless signing, CI setup, local testing
- **[Lifecycle](guides/registry/10-lifecycle.md)** — maturity, deprecation, compatibility, and acceptance
