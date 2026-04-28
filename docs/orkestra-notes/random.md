# Random Notes

Cryptographically random generation for secrets.

> **Required:** always use with `once: true` on secret templates. Without it, a new value is generated on every reconcile cycle — passwords change every 30 seconds, breaking every application that uses them.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `randomAlphanumeric` | `n int` | `string` of exactly n characters (a-z, A-Z, 0-9) |
| `randomHex` | `n int` | `string` of 2n hex characters (n random bytes) |
| `randomBase64` | `n int` | URL-safe base64 from n random bytes |

## Example

```yaml
secrets:
  - name: "{{ .metadata.name }}-credentials"
    once: true                              # required — evaluated once, on creation only
    data:
      password: "{{ randomAlphanumeric 32 }}"
      apiKey:   "{{ randomHex 16 }}"
      jwtSecret: "{{ randomBase64 32 }}"
```

## Why `once: true` is required

Notes are pure by contract — same input, same output. Random notes are the exception: they produce different output on every call. `once: true` is the semantic safeguard: Orkestra evaluates the template only when the Secret does not yet exist. On subsequent reconciles, the existing Secret is left untouched.

Using random notes without `once: true` is not blocked at the language level, but it will cause credentials to rotate on every reconcile cycle.
Use `rotateAfer` for automatic secret rotation.
