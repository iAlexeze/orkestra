# 02 — Result Maps

Every protocol client returns a `map[string]interface{}` that is injected under `.external.<name>`. The keys differ by protocol. This document lists what each protocol puts in the map and how to reference it in templates.

All result maps include two common keys:

| Key | Value |
|-----|-------|
| `error` | Empty string on success. Error message on failure. |
| `called` | `"true"` always (except skipped calls — see below). |

---

## HTTP (`protocol: http` or omitted)

```yaml
external:
  - name: api
    url: "http://my-service/status"
```

| Key | Type | Value |
|-----|------|-------|
| `status` | string | HTTP status code as string: `"200"` |
| `body` | string | Raw response body (max 4096 bytes) |
| `error` | string | Empty on success; `"HTTP 404"` or `"expected status 200, got 500"` on failure |
| `called` | string | `"true"` |

When the response body is valid JSON, its keys are also merged directly into the result. A response `{"pendingJobs": 100}` makes `.external.api.pendingJobs` available alongside `.external.api.body`.

```yaml
# accessing structured JSON response
- field: external.api.pendingJobs
  greaterThan: "50"

# accessing raw body
- field: external.api.body
  contains: "ready"

# status check
- field: external.api.status
  equals: "200"
```

### Skipped calls (HTTP)

When `when:` conditions are not met, a skipped result is injected:

```json
{
  "status": "", 
  "body": "", 
  "error": "", 
  "called": "false"
}
```

---

## Prometheus (`protocol: prometheus`)

```yaml
external:
  - name: queueDepth
    protocol: prometheus
    url: "http://prometheus.monitoring.svc:9090"
    query: "sum(rabbitmq_queue_messages{vhost=\"/\"})"
```

| Key | Type | Value |
|-----|------|-------|
| `result` | string | The scalar value from the PromQL result |
| `raw` | map | The full parsed Prometheus API response |
| `error` | string | Empty on success; error string from Prometheus or the client |
| `called` | string | `"true"` |

`result` holds the canonical scalar — for `vector` result types, this is `value[1]` of the first series. Use `promAboveThreshold` and `promBelowThreshold` notes to compare it:

```yaml
- field: "{{ promAboveThreshold .external.queueDepth 10000 }}"
  equals: "true"
```

For multi-series vectors, use `promSum` or `promMax` on `.external.queueDepth.raw` to aggregate across series:

```yaml
- field: "{{ promSum .external.queueDepth.raw }}"
  greaterThan: "500"
```

The `url:` field can point directly at the API endpoint or at a Prometheus base URL — the client appends `/api/v1/query` automatically when the path does not already contain it.

---

## Redis (`protocol: redis`)

```yaml
external:
  - name: cacheSize
    protocol: redis
    url: "redis://redis.default.svc:6379"
    query: "LLEN myqueue"
    auth:
      secretRef:
        name: redis-secret
        namespace: default
        key: password
```

| Key | Type | Value |
|-----|------|-------|
| `result` | string | String representation of the command response |
| `raw` | map | `{"value": <untyped response>}` |
| `error` | string | Empty on success |
| `called` | string | `"true"` |

`query:` is a Redis command string. Any command that returns a scalar is supported:

```
GET mykey
HGET myhash field
LLEN mylist
SCARD myset
STRLEN mykey
```

Multi-word arguments can be quoted: `HGET myhash "field with spaces"`.

The credential from `auth:` is used as the Redis password regardless of whether the URL already contains one.

---

## Postgres (`protocol: postgres`)

```yaml
external:
  - name: jobCount
    protocol: postgres
    url: "postgres://user:pass@postgres.default.svc:5432/mydb"
    query: "SELECT COUNT(*) FROM jobs WHERE status = 'pending'"
    auth:
      secretRef:
        name: pg-secret
        namespace: default
        key: connection-string
```

| Key | Type | Value |
|-----|------|-------|
| `result` | string | First column of the first row as a string |
| `raw` | map | All columns of the first row as `{"col": value}` |
| `error` | string | Empty on success |
| `called` | string | `"true"` |

`query:` is a SQL `SELECT` statement. Only the first row is read — for count queries and scalar lookups this is always sufficient.

When `auth.secretRef` is set, the credential overrides the password in the `url:` DSN. Use this when the connection string is in a Secret and you do not want to embed credentials in the Katalog.

---

## MongoDB (`protocol: mongo`)

```yaml
external:
  - name: queueDepth
    protocol: mongo
    url: "mongodb://mongo.default.svc:27017"
    query: "mydb.jobs"
    auth:
      secretRef:
        name: mongo-secret
        namespace: default
        key: uri
```

| Key | Type | Value |
|-----|------|-------|
| `result` | string | Document count as a string |
| `raw` | map | `{"count": <int64>}` |
| `error` | string | Empty on success |
| `called` | string | `"true"` |

`query:` is `"<database>.<collection>"`. The client runs `CountDocuments({})` against that collection. For filtered counts, the filter is not yet supported — the full collection count is returned.

When `auth.secretRef` is set, the credential replaces the `url:` entirely and is used as the full MongoDB URI.

---

## Kafka (`protocol: kafka`)

```yaml
external:
  - name: consumerLag
    protocol: kafka
    url: "broker1:9092,broker2:9092"
    query: "my-consumer-group/my-topic"

  - name: topicInfo
    protocol: kafka
    url: "broker1:9092"
    query: "@my-topic"
```

### Consumer group lag (`"group/topic"`)

| Key | Type | Value |
|-----|------|-------|
| `result` | string | Total lag across all partitions |
| `lag` | string | Same as `result` |
| `partitions` | string | Partition count for the topic |
| `error` | string | Empty on success |
| `called` | string | `"true"` |

### Topic metadata (`"@topic"`)

| Key | Type | Value |
|-----|------|-------|
| `result` | string | Partition count |
| `partitionCount` | string | Same as `result` |
| `error` | string | Empty on success |
| `called` | string | `"true"` |

### SASL/PLAIN authentication

Set `auth.secretRef` to a Secret containing `username:password` in a single key:

```yaml
auth:
  secretRef:
    name: kafka-credentials
    namespace: default
    key: sasl-plain
```

```yaml
# Secret
stringData:
  sasl-plain: "myuser:mysecret"
```

The client splits on the first `:` and constructs a SASL/PLAIN dialer. When no `auth:` is declared, the default dialer is used (no authentication).

### Broker URL format

`url:` accepts a comma-separated broker list or a `kafka://` prefixed URL:

```
broker1:9092,broker2:9092
kafka://broker:9092
```

---

## Skipped result (non-HTTP)

When `when:` conditions are not met for any non-HTTP protocol:

```go
{ "result": "", "raw": {}, "error": "", "called": "false" }
```

Use `external.<name>.called` to distinguish a skipped call from a successful one that returned an empty value.

---

**Next →** [03 — Auth Resolution](03-auth.md)
