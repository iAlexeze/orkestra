# Time Notes

Parse timestamps, measure elapsed time, and validate duration strings in Katalog templates. Primarily used in `status.fields` to surface human-readable time values and in `when:` conditions to drive time-based rotation and expiry logic.

All timestamp notes accept RFC3339, RFC3339Nano, `2006-01-02T15:04:05Z`, and `YYYY-MM-DD` formats. Safe zero values are returned for unparseable input.

## Reference

| Note | Description |
|------|-------------|
| `timeAgo` | Return a human-readable elapsed-time string from a timestamp. |
| `timeSince` | Return the number of seconds elapsed since a timestamp as an integer. |
| `isExpired` | Return `true` when a timestamp plus a duration is in the past. |
| `timeFormat` | Reformat a timestamp string using Go's time format layout. |
| `durationSeconds` | Parse a Go duration string and return the total number of seconds as an integer. |
| `durationAdd` | Add two Go duration strings and return the result as a canonical duration string. |
| `durationValid` | Return `true` when the string is a valid Go duration. |

## Examples

```yaml
# timeAgo
status:
  fields:
    - path: lastSyncAgo
      value: "{{ timeAgo .children.cronjob.status.lastScheduleTime }}"
      # → "5s ago" / "12m ago" / "3h ago" / "2d ago"

    - path: createdAgo
      value: "{{ timeAgo .metadata.creationTimestamp }}"

# timeSince
# Gate rotation on time elapsed (30 days = 2592000 seconds)
when:
  - field: "{{ timeSince (index .metadata.annotations \"myorg.io/last-rotated\") }}"
    operator: gte
    value: "2592000"

# isExpired
# Recreate the secret when the rotation annotation says it is due
onCreate:
  secrets:
    - name: "{{ .metadata.name }}-token"
      once: true
      rotateAfter: 720h

# Gate a resource on whether a timestamp annotation is past its TTL
when:
  - field: "{{ isExpired (index .metadata.annotations \"myorg.io/generated-at\") \"720h\" }}"
    equals: "true"

# timeFormat
status:
  fields:
    - path: createdDate
      value: "{{ timeFormat .metadata.creationTimestamp \"Jan 2, 2006\" }}"
      # → "Apr 13, 2026"

    - path: expiresAt
      value: "{{ timeFormat .status.expirationTime \"2006-01-02\" }}"

# durationSeconds
status:
  fields:
    - path: resyncIntervalSeconds
      value: "{{ durationSeconds .spec.resyncInterval }}"
      # "5m" → 300  |  "1h30m" → 5400

# durationAdd
# Compute the total window from base + buffer
status:
  fields:
    - path: totalWindow
      value: "{{ durationAdd .spec.baseWindow .spec.buffer }}"
      # "5m" + "30s" → "5m30s"

# durationValid
spec:
  crds:
    myApp:
      validate:
        - message: "spec.rotationPeriod must be a valid Go duration (e.g. 720h, 30m)"
          deny:
            - field: "{{ durationValid .spec.rotationPeriod }}"
              equals: "false"
```
