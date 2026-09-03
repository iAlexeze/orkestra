# pkg/signing/cosign

Cosign keyless signing and verification for Orkestra OCI artifacts. Uses the
cosign CLI binary — checking PATH first, downloading and caching at
`~/.orkestra/tools/cosign` if not found. No install step required.

## How it works

```
ork pattern sign postgres:v1  (or ork push --sign)

  1. Cosign requests a short-lived signing certificate from Fulcio
     using the ambient OIDC token (GitHub Actions workflow, GitLab CI job, etc.)
  2. The OCI image digest is signed with the ephemeral key
  3. The signature is pushed back to the registry as a referrer artifact
```

Verification (`ork pattern verify`, `ork inspect`) fetches the referrer, validates
the certificate chain back to the Fulcio root CA, and optionally checks the
subject claim against `publish.signing.expectedIdentities`.

## Identity subjects by CI platform

The `expectedIdentities` field in `publish.signing` uses OIDC subject claims.

| Platform | Subject format | Issuer |
|----------|---------------|--------|
| GitHub Actions | `github.com/<org>/<repo>/.github/workflows/<file>.yaml@refs/heads/<branch>` | `https://token.actions.githubusercontent.com` |
| GitLab CI | `gitlab.com/<namespace>/<project>//<job>@refs/heads/<branch>` | `https://gitlab.com` |
| Other | Platform-specific | sigstore public issuer |

The issuer is inferred automatically from the subject prefix — no extra config needed.

## Cosign binary

The package resolves the binary in this order:

1. `cosign` on `$PATH`
2. Cached binary at `~/.orkestra/tools/cosign`
3. Download from GitHub releases (`sigstore/cosign`) and cache it

The first sign or verify call that needs cosign fetches it automatically.

## See also

- [pkg/signing/](../README.md) — provider-agnostic signing boundary
