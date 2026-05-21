# Time Notes

Timestamp inspection and duration arithmetic.

> Timestamp notes (`timeAgo`, `timeSince`, `isExpired`) depend on wall-clock time — they are pure with respect to their inputs but produce different output at different times. Safe to use in status fields; do not use in conversion paths.

## Timestamp notes

| Note | Signature | Returns |
|------|-----------|---------|
| `timeAgo` | `string` | `string` — "5m ago", "2h ago", "3d ago" |
| `timeSince` | `string` | `int64` — seconds elapsed since timestamp |
| `timeFormat` | `timestamp, layout string` | `string` — reformatted timestamp |
| `isExpired` | `timestamp, duration string` | `bool` — true when timestamp + duration is in the past |

Timestamps are parsed from RFC3339 / RFC3339Nano / `"2006-01-02T15:04:05Z"` / `"2006-01-02"`. Empty or invalid input returns a safe zero value (`""`, `0`, or `false`).

`timeFormat` uses Go's time layout convention: `"Jan 2, 2006"`, `"15:04"`, `"2006-01-02"`.

## Duration notes

| Note | Signature | Returns |
|------|-----------|---------|
| `durationSeconds` | `string` | `int64` — total seconds, `0` on invalid |
| `durationAdd` | `a, b string` | `string` — sum as Go duration string, `"0s"` on invalid |
| `durationValid` | `string` | `bool` |

Duration strings follow Go's `time.ParseDuration` format: `"300ms"`, `"5m"`, `"1h30m"`, `"24h"`. Note: Go does not support `"d"` for days — use `"168h"` for 7 days.

## Examples

```yaml
# Age of the CR in human-readable form
- path: age
  value: "{{ timeAgo .metadata.creationTimestamp }}"

# Formatted creation date
- path: createdAt
  value: "{{ timeFormat .metadata.creationTimestamp \"Jan 2, 2006\" }}"

# Trigger secret rotation after 30 days
- path: rotationNeeded
  value: "{{ isExpired (index .metadata.annotations \"orkestra.io/generated-at\") \"720h\" }}"

# Convert a spec duration field to seconds for a child resource
timeoutSeconds: "{{ durationSeconds .spec.timeout }}"

# Combine two durations
totalTimeout: "{{ durationAdd .spec.connectTimeout .spec.readTimeout }}"

# Validate a duration field
- field: spec.timeout
  operator: custom
  value: "{{ durationValid .spec.timeout }}"
  message: "spec.timeout must be a valid Go duration (e.g. 30s, 5m, 1h)"
  action: deny
```
