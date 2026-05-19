# Semver Notes

Parse and compare semantic version strings. Uses [Masterminds/semver v3](https://github.com/Masterminds/semver) — supports `v`-prefix, pre-release tags, and build metadata.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `semverMajor` | `string` | `string` — major component, `""` on invalid |
| `semverMinor` | `string` | `string` — minor component, `""` on invalid |
| `semverPatch` | `string` | `string` — patch component, `""` on invalid |
| `semverValid` | `string` | `bool` |
| `semverCompare` | `a, b string` | `int` (-1 when a < b, 0 equal, 1 when a > b) |
| `semverBump` | `version, component string` | `string` — bumped version |
| `semverConstraint` | `version, constraint string` | `bool` |

## Examples

```yaml
# Gate upgrade on version constraint
- path: upgradeReady
  value: "{{ semverConstraint .spec.version \">=1.0.0,<2.0.0\" }}"

# Expose version components in status
- path: majorVersion
  value: "{{ semverMajor .spec.version }}"

- path: minorVersion
  value: "{{ semverMinor .spec.version }}"

# Validate image tag is a semver
- field: spec.version
  value: "{{ semverValid .spec.version }}"
  message: "spec.version must be a valid semver string (e.g. 1.2.3)"
  action: deny

# Auto-bump patch version
version: "{{ semverBump .spec.currentVersion \"patch\" }}"
```

## Constraint syntax

The `semverConstraint` note accepts Masterminds semver constraint expressions:

| Expression | Meaning |
|-----------|---------|
| `>=1.0.0` | 1.0.0 or later |
| `<2.0.0` | before 2.0.0 |
| `>=1.0.0,<2.0.0` | range (AND) |
| `^1.2.0` | compatible with 1.2.0 (>=1.2.0, <2.0.0) |
| `~1.2.0` | patch-level compatible (>=1.2.0, <1.3.0) |
| `1.x` | any 1.x version |
