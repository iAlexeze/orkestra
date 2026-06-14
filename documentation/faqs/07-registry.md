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

## Next

- **[Orkestra Registry](../orkestra-registry/index.md)** — full publishing and pulling reference
- **[simulate gates](../orkestra-registry/05-simulate.md)** — how simulate quality signals work
- **[e2e gates](../orkestra-registry/04-e2e.md)** — how e2e quality signals work
