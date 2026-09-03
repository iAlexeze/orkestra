# 04 — Security

The `security` block in the Katalog YAML controls two opt-in features: deletion protection and RBAC auto-apply. Both use the same default-on-when-declared semantics.

## Default-on semantics

Orkestra distinguishes three states for any security flag:

| YAML state | Go value | Meaning |
|------------|----------|---------|
| `security` block absent | `nil` | Feature not opt-in — disabled |
| Block present, flag absent | `nil` inside block | Opted in — enabled by default |
| `enabled: true` | `*true` | Explicitly enabled |
| `enabled: false` | `*false` | Explicitly disabled |

This uses `*bool` fields so that `nil` ("not declared") is distinguishable from `false` ("explicitly off").

## Deletion protection

```go
k.IsDeletionProtectionEnabled()  // bool
```

Decision:
- No `security` block → `false`
- Block present, no `deletionProtection` key → `true` (default-on)
- `deletionProtection.enabled: true` → `true`
- `deletionProtection.enabled: false` → `false`

Also gates TLS certificate generation — `k.NeedsCertificates()` returns `true` when deletion protection is on and the user has not provided their own TLS cert.

`CleanupOnShutdown` is explicitly false by default: Deletion protection webhook survive operator restarts to maintain the security.

## Example

```yaml
security:
  deletionProtection:
    enabled: false   # explicitly disabled
    cleanupOnShutdown: true
```

→ Next: [05-builtins.md](05-builtins.md)
