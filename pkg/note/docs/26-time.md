# 26 — Time Notes

Parse timestamps, measure elapsed time, and validate duration strings in Katalog templates. Primarily used in `status.fields` to surface human-readable time values and in `when:` conditions to drive time-based rotation and expiry logic.

All timestamp notes accept RFC3339, RFC3339Nano, `2006-01-02T15:04:05Z`, and `YYYY-MM-DD` formats. Safe zero values are returned for unparseable input.

## Reference

### `timeAgo`

Return a human-readable elapsed-time string from a timestamp. Outputs seconds, minutes, hours, or days depending on magnitude.

Keywords: time, ago, elapsed, human, readable, display, status, string

```yaml
status:
  fields:
    - path: lastSyncAgo
      value: "{{ timeAgo .children.cronjob.status.lastScheduleTime }}"
      # → "5s ago" / "12m ago" / "3h ago" / "2d ago"

    - path: createdAgo
      value: "{{ timeAgo .metadata.creationTimestamp }}"
```

---

### `timeSince`

Return the number of seconds elapsed since a timestamp as an integer. Use for programmatic comparisons in `when:` conditions.

Keywords: time, since, seconds, elapsed, duration, int, compare

```yaml
# Gate rotation on time elapsed (30 days = 2592000 seconds)
when:
  - field: "{{ timeSince (index .metadata.annotations \"myorg.io/last-rotated\") }}"
    operator: gte
    value: "2592000"
```

---

### `isExpired`

Return `true` when a timestamp plus a duration is in the past. The canonical way to drive rotation logic in `when:` conditions. Duration follows Go's `time.ParseDuration` format (`"30m"`, `"24h"`, `"720h"` for 30 days — Go does not support `"d"`).

Keywords: time, expired, expiry, rotation, ttl, boolean, when, condition

```yaml
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
```

---

### `timeFormat`

Reformat a timestamp string using Go's time format layout. Returns `""` for unparseable input.

Keywords: time, format, layout, display, readable, string, date

```yaml
status:
  fields:
    - path: createdDate
      value: "{{ timeFormat .metadata.creationTimestamp \"Jan 2, 2006\" }}"
      # → "Apr 13, 2026"

    - path: expiresAt
      value: "{{ timeFormat .status.expirationTime \"2006-01-02\" }}"
```

---

### `durationSeconds`

Parse a Go duration string and return the total number of seconds as an integer. Returns `0` for invalid input.

Keywords: time, duration, seconds, parse, int, convert

```yaml
status:
  fields:
    - path: resyncIntervalSeconds
      value: "{{ durationSeconds .spec.resyncInterval }}"
      # "5m" → 300  |  "1h30m" → 5400
```

---

### `durationAdd`

Add two Go duration strings and return the result as a canonical duration string. Returns `"0s"` for invalid input.

Keywords: time, duration, add, combine, string

```yaml
# Compute the total window from base + buffer
status:
  fields:
    - path: totalWindow
      value: "{{ durationAdd .spec.baseWindow .spec.buffer }}"
      # "5m" + "30s" → "5m30s"
```

---

### `durationValid`

Return `true` when the string is a valid Go duration. Use in validation rules to reject malformed duration fields. Note: Go does not support `d` (days) — `"720h"` is correct for 30 days.

Keywords: time, duration, valid, validate, boolean, check, go

```yaml
spec:
  crds:
    myApp:
      validate:
        - message: "spec.rotationPeriod must be a valid Go duration (e.g. 720h, 30m)"
          deny:
            - field: "{{ durationValid .spec.rotationPeriod }}"
              equals: "false"
```

---

## Quick reference

| Note | Accepts | Returns | Notes |
|------|---------|---------|-------|
| `timeAgo` | `timestamp string` | `string` | `"Xs ago"` / `"Xm ago"` / `"Xh ago"` / `"Xd ago"` |
| `timeSince` | `timestamp string` | `int64` | seconds since timestamp |
| `isExpired` | `timestamp string, duration string` | `bool` | `true` when `timestamp + duration` is in the past |
| `timeFormat` | `timestamp, layout string` | `string` | Go time layout |
| `durationSeconds` | `duration string` | `int64` | Go duration → seconds |
| `durationAdd` | `a, b string` | `string` | sum of two Go durations |
| `durationValid` | `duration string` | `bool` | `false` for `"5d"` — Go has no days |
