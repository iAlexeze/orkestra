# Artifact signing

Orkestra patterns carry cryptographic proof of their origin. A pushed pattern
can be signed with a keyless Cosign signature — anyone with the OCI ref can
verify who pushed it, from which workflow, without managing keys.

---

## Correctness vs identity

`ork push` runs the simulate and E2E gates automatically and bakes the results
into the artifact — correctness is already handled by the time the artifact
reaches the registry. Anyone who runs `ork inspect` can see what passed and what
didn't.

What correctness annotations cannot tell you is **who published the artifact**:
which organisation, which pipeline, which workflow identity was behind the push.
A team consuming a pattern from the registry has no cryptographic way to confirm
the artifact came from a trusted source and not from a compromised or unrelated account.

Artifact signing adds that identity layer.

---

## How keyless signing works

Orkestra uses **Cosign keyless signing** — no private keys, no key rotation,
no secrets in CI. The signing identity is the OIDC token that CI already
issues to every job.

```text
ork push postgres:v14 ./           1. Artifact pushed to OCI registry

ork pattern sign postgres:v14

  2. Cosign requests a short-lived certificate from Fulcio
     using the ambient OIDC token (GitHub Actions workflow identity,
     GitLab CI job token, etc.)
  3. The certificate binds the OIDC subject to the OCI digest
  4. The signature is stored as an OCI referrer artifact on the registry
  5. Optionally: the signature entry is recorded in the Rekor transparency log
```

The signing certificate expires in minutes. The proof lives in the OCI registry
and the Rekor log — permanently verifiable, no secrets involved.

---

## OIDC identity subjects

The `publish.signing.expectedIdentities` field uses OIDC **subject claims** —
not public keys. The subject identifies the CI context that signed the artifact.

| Platform | Subject format | Issuer |
|----------|---------------|--------|
| GitHub Actions | `github.com/<org>/<repo>/.github/workflows/<file>.yaml@refs/heads/<branch>` | `https://token.actions.githubusercontent.com` |
| GitLab CI | `gitlab.com/<namespace>/<project>//<job>@refs/heads/<branch>` | `https://gitlab.com` |

The issuer is inferred automatically from the subject prefix — no extra config needed.

---

## Declaring a signing policy

The `publish:` block in `katalog.yaml` declares the signing policy for a pattern.
Consumers see it; `ork pull --verify` enforces it.

```yaml
publish:
  signing:
    verify: true
    expectedIdentities:
      - github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
```

`verify: true` — `ork pull` refuses the artifact unless the signature is present
and the subject matches one of `expectedIdentities`.

`expectedIdentities` empty — any valid keyless signature is accepted.

`publish.signing` absent — no verification is required at pull time. `ork inspect`
still shows `Signed: ✗ not signed`.

→ Full field reference: [`publish:` schema](../reference/schema/02-katalog/23-publish.md)

---

## Signing commands

### Sign after push

```bash
ork push postgres:v14 ./
ork pattern sign postgres:v14
```

Signing is separate from pushing — the two steps can run under different OIDC
contexts. Sign from whichever CI job holds the identity you want on the certificate.

```bash
# Or combine in one command (CI convenience)
ork push postgres:v14 ./ --sign
```

`ork push --sign` is exactly `ork push` followed by `ork pattern sign`. Same code,
one command.

### Verify

```bash
ork pattern verify postgres:v14
```

```text
✓ Verified (keyless)
  subject:  github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
  issuer:   https://token.actions.githubusercontent.com
```

### Inspect (always shown)

`ork inspect` always attempts verification and shows the result inline — no flag needed:

```text
postgres:v14
  Simulate:    ✓ passed · 3 assertions · 45s · tested 2 days ago
  E2E:         ✓ passed · 5 assertions · 45s · tested 2 days ago
  Signed:      ✓ verified (keyless) · github.com/myorg/postgres/.github/workflows/release.yaml@refs/heads/main
```

Add `--verbose` to expand the issuer and Rekor log entry.

---

## Enforcing at pull time

When `publish.signing.verify: true` is set, `ork pull --verify` reads the policy
from the pulled katalog and refuses if the signature is absent or the identity
does not match:

```bash
ork pull postgres:v14 --verify
```

This is the consumer-side enforcement — the pattern author declares the policy,
the consumer enforces it at import time.

---

## Local testing with ttl.sh

You do not need a real OCI registry or CI credentials to test signing. Push to
[ttl.sh](https://ttl.sh) — an ephemeral public registry — sign with your local
browser-based OIDC flow, and verify immediately.

```bash
# Push to ttl.sh and sign (browser opens for OIDC)
ork pattern sign postgres:v14 --local --dir ./patterns/postgres/ --ttl 24h

# Output shows exactly what to run next
#   Expires in   24h
#   Verify:      ork pattern verify oci://ttl.sh/ork-a3f9c2/postgres:24h --no-tlog
#   Inspect:     ork inspect oci://ttl.sh/ork-a3f9c2/postgres:24h

# Or from push:
ork push postgres:v14 ./ --sign-local --ttl 24h
```

`--no-tlog` skips the Rekor transparency log check. For a 24h ephemeral artifact,
a permanent log entry makes no sense.

---

## Cosign binary

Signing and verification use the cosign CLI binary, not a Go library — keeping the
dependency surface minimal. The binary is resolved in this order:

1. `cosign` on `$PATH`
2. Cached binary at `~/.orkestra/tools/cosign`
3. Downloaded from GitHub releases (`sigstore/cosign`) and cached — no manual install

---

## CI example: GitHub Actions

```yaml
- name: Push pattern
  run: ork push postgres:v14 ./patterns/postgres/

- name: Sign pattern
  run: ork pattern sign postgres:v14
  # Uses ACTIONS_ID_TOKEN_REQUEST_URL automatically via cosign
  # Subject: github.com/<org>/<repo>/.github/workflows/release.yaml@refs/heads/main
```

No secrets, no key management. The OIDC token is the credential.

For full push-and-sign in one step:

```yaml
- name: Push and sign
  run: ork push postgres:v14 ./patterns/postgres/ --sign
```

---

## See also

- [`ork pattern sign` / `ork pattern verify`](../reference/cli/12-pattern.md)
- [`ork push --sign`](../reference/cli/09-push.md)
- [`publish:` schema](../reference/schema/02-katalog/23-publish.md)
- [sigstore/cosign](https://github.com/sigstore/cosign)
- [Fulcio CA](https://github.com/sigstore/fulcio)
