# pkg/signing

Provider-agnostic signing and verification for Orkestra OCI artifacts.

The package boundary is intentionally thin — it exposes the operations Orkestra
needs (sign after push, verify at pull) without coupling callers to a specific
signing implementation. Today the only provider is Cosign; future providers
(Notary v2, an internal CA) slot in as siblings without touching call sites.

## Packages

| Package | Description |
|---------|-------------|
| [`cosign/`](cosign/) | Cosign keyless signing and verification — binary-based, auto-download |

## Usage

```go
import orksigning "github.com/orkspace/orkestra/pkg/signing/cosign"

// Sign after push
err := orksigning.Sign(ctx, ref, orksigning.SignOptions{
    SkipConfirmation: true,  // always true in CI / ork push
})

// Verify at pull
err := orksigning.Verify(ctx, ref, orksigning.VerifyOptions{
    ExpectedIdentities: katalog.Publish.ExpectedIdentities(),
})
```

## Provider docs

- [Cosign provider](cosign/README.md)
