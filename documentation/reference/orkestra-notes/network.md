# Network Notes

IP address and CIDR block helpers. Useful for network policy operators and ingress controllers that need to validate or classify IP addresses at reconcile time.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `cidrContains` | `cidr, ip string` | `bool` — `false` on invalid CIDR or IP |
| `ipValid` | `string` | `bool` — accepts IPv4 and IPv6 |
| `ipIsPrivate` | `string` | `bool` — RFC 1918 (IPv4) and RFC 4193 (IPv6) ranges |

Private ranges checked by `ipIsPrivate`:
- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`
- `fc00::/7` (IPv6 ULA)

## Examples

```yaml
# Validation: restrict to private addresses only
- field: spec.targetIP
  value: "{{ ipIsPrivate .spec.targetIP }}"
  message: "spec.targetIP must be a private IP address"
  action: deny

# Validate IP format before using it
- field: spec.allowedIP
  value: "{{ ipValid .spec.allowedIP }}"
  message: "spec.allowedIP must be a valid IP address"
  action: deny

# Gate a network policy resource on CIDR membership
when:
  - field: "{{ cidrContains .spec.internalCIDR .spec.targetIP }}"
    equals: "true"

# Status: expose whether the target is internal
- path: isInternalTarget
  value: "{{ cidrContains \"10.0.0.0/8\" .spec.targetIP }}"
```
