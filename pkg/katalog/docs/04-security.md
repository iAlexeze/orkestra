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

## RBAC

```go
k.IsRBACEnabled()          // bool — same decision table as above
k.RBACCleanupOnShutdown()  // bool — default false; RBAC survives restarts
```

RBAC auto-apply generates and applies `ClusterRole` and `ClusterRoleBinding` resources based on the resources each CRD's reconciler touches. It is enabled by default when the `security` block is present.

`RBACCleanupOnShutdown` is explicitly false by default: RBAC rules survive operator restarts so that CRs can still be read by other tools while the operator is down.

## Example

```yaml
security:
  # deletionProtection not declared → enabled by default
  rbac:
    enabled: false    # explicitly opt out of RBAC generation
```

```yaml
security:
  deletionProtection:
    enabled: false   # explicitly disabled
  rbac:
    cleanupOnShutdown: true
```

→ Next: [05-builtins.md](05-builtins.md)
