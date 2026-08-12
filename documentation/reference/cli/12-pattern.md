# ork pattern

Manage a single pattern artifact. `ork pattern` is to a pattern what `docker image` is to a container image — it operates on an artifact that already exists in the registry.

```bash
ork pattern <subcommand> [flags]
```

---

## Subcommands

| Subcommand | Description |
|------------|-------------|
| [`sign`](#ork-pattern-sign) | Sign a pushed pattern with Cosign keyless |
| [`verify`](#ork-pattern-verify) | Verify the Cosign keyless signature on a pattern |

---

## ork pattern sign

Sign an already-pushed pattern artifact using Cosign keyless signing. Uses the
ambient OIDC token — in GitHub Actions this is the workflow identity; locally it
opens a browser-based OIDC flow.

```bash
ork pattern sign <name>:<version> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--local` | `false` | Push to ttl.sh and sign for local testing. Does not touch the normal registry. |
| `--dir <path>` | `.` | Pattern directory to push when using `--local`. |
| `--ttl <duration>` | `1h` | TTL for the ttl.sh artifact when using `--local` (e.g. `1h`, `24h`). |

### Examples

```bash
# Sign a pattern that was already pushed to the registry
ork pattern sign postgres:v14

# Full OCI ref
ork pattern sign oci://ghcr.io/myorg/katalogs/postgres:v14

# Push to ttl.sh and sign — local testing, no real registry needed
ork pattern sign postgres:v14 --local --dir ./patterns/postgres/

# Local test with a longer TTL
ork pattern sign postgres:v14 --local --dir ./patterns/postgres/ --ttl 24h
```

### Output (standard)

```text
✓ Signed:  oci://ghcr.io/myorg/katalogs/postgres:v14
```

### Output (--local)

```text
✓ Pushed:  oci://ttl.sh/ork-a3f9c2/postgres:1h
✓ Signed

  Expires in   1h
  Verify:      ork pattern verify oci://ttl.sh/ork-a3f9c2/postgres:1h --no-tlog
  Inspect:     ork inspect oci://ttl.sh/ork-a3f9c2/postgres:1h
```

### How it works

Signing is separate from pushing. You push first — with whatever CI pipeline and
credentials you have — then sign as a separate step using the OIDC identity that
should be trusted. The signature is attached to the pushed digest as an OCI
referrer artifact.

```text
ork push postgres:v14 ./           ← publish the artifact
ork pattern sign postgres:v14      ← sign it (separate OIDC context if needed)
```

`ork push --sign` combines both steps in a single command — it is exactly `ork push`
followed by `ork pattern sign` with the same ref. Use it in CI when the push and
sign credentials are available in the same job.

---

## ork pattern verify

Verify the Cosign keyless signature on a pattern artifact. Prints the subject and
issuer of the signing identity.

```bash
ork pattern verify <name>:<version> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--verbose` | `false` | Show full certificate and Rekor log detail. |
| `--no-tlog` | `false` | Skip Rekor transparency log check. Use for ephemeral artifacts (`--local` or `ttl.sh`) and local testing. |

### Examples

```bash
# Verify a katalog signature
ork pattern verify postgres:v14

# Verify with full identity detail
ork pattern verify postgres:v14 --verbose

# Verify a locally signed ttl.sh artifact (skip Rekor check)
ork pattern verify oci://ttl.sh/ork-a3f9c2/postgres:1h --no-tlog
```

### Output

```text
✓ Verified (keyless)
  subject:  github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
  issuer:   https://token.actions.githubusercontent.com
```

Not signed:

```text
✗ Not verified
```

With `--verbose`:

```text
✓ Verified (keyless)
  subject:  github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
  issuer:   https://token.actions.githubusercontent.com

  [full cosign certificate output]
```

---

## Local testing workflow

Use `--local` to test the full sign → verify loop without a real registry or CI credentials.
The pattern is pushed to [ttl.sh](https://ttl.sh) — an ephemeral public OCI registry that
requires no auth and cleans up automatically.

```bash
# 1. Push to ttl.sh and sign (opens browser for OIDC)
ork pattern sign postgres:v14 --local --dir ./patterns/postgres/ --ttl 24h

# 2. Verify the signature (output shows the exact commands)
ork pattern verify oci://ttl.sh/ork-a3f9c2/postgres:24h --no-tlog

# 3. See the Signed row in inspect
ork inspect oci://ttl.sh/ork-a3f9c2/postgres:24h
```

`--no-tlog` is needed for ephemeral artifacts: the Rekor transparency log stores
signatures permanently, which makes no sense for a 1h test artifact.

---

## Cosign binary

`ork pattern sign` and `ork pattern verify` use the cosign CLI binary.

- If `cosign` is on `$PATH`, it is used directly.
- If not found, it is downloaded automatically to `~/.orkestra/tools/cosign` on first use. No manual install required.

---

## Related

- [`ork push --sign`](./09-push.md) — sign at push time (same as push + `ork pattern sign`)
- [`ork inspect`](./11-inspect.md) — shows `Signed:` status for every artifact
- [`publish:` schema](../schema/02-katalog/23-publish.md) — declare signing policy in the katalog
- [Artifact Signing](../../security/10-artifact-signing.md) — how keyless signing works
