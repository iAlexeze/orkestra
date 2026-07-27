# 05 — Adding a Protocol

Adding a new protocol takes four steps. Use `redis.go` as the simplest reference — it is the shortest client and covers the full pattern.

---

## Step 1 — Add the protocol constant

In `pkg/types/external.go`, add a constant to the `ExternalProtocol` type:

```go
const (
    ProtocolHTTP       ExternalProtocol = "http"
    ProtocolPrometheus ExternalProtocol = "prometheus"
    ProtocolRedis      ExternalProtocol = "redis"
    ProtocolPostgres   ExternalProtocol = "postgres"
    ProtocolMongo      ExternalProtocol = "mongo"
    ProtocolKafka      ExternalProtocol = "kafka"
    ProtocolNATS       ExternalProtocol = "nats"       // ← example
    ProtocolMQTT       ExternalProtocol = "mqtt"
    ProtocolGRPC       ExternalProtocol = "grpc"
)
```

---

## Step 2 — Implement `ProtocolClient`

Create `pkg/external/<protocol>.go`. The interface is:

```go
type ProtocolClient interface {
    Fetch(
        ctx           context.Context,
        spec          orktypes.ExternalCallSpec,
        resolvedURL   string,
        resolvedQuery string,
        credential    string,
    ) (map[string]interface{}, error)
}
```

Rules:
- **Never return a Go error for network or protocol failures.** Capture them in `result["error"]` and return `nil`. The Go error path is for context cancellations and internal panics only — things the caller cannot recover from.
- **Always set `"called": "true"` in the result.** The `errorResult()` helper in `prometheus.go` does this correctly; call it for failure paths.
- **Respect `spec.Timeout`.** Parse it with `time.ParseDuration`; fall back to `defaultExternalTimeout` (10s). Apply it via `context.WithTimeout`.
- **Use `credential` as-is.** `resolveAuth()` has already resolved it from `secretRef` or `env`. Your client decides what to do with the raw string.
- **Decide the result map shape and document it.** Non-HTTP protocols conventionally use `result` (primary scalar), `raw` (structured data), `error`, and `called`. Add protocol-specific keys when useful (e.g. `lag`, `partitions` for Kafka).

### Minimal template

```go
package external

import (
    "context"
    "fmt"
    "time"

    orktypes "github.com/orkspace/orkestra/pkg/types"
)

type myProtocolClient struct{}

func (c *myProtocolClient) Fetch(
    ctx context.Context,
    spec orktypes.ExternalCallSpec,
    resolvedURL, resolvedQuery, credential string,
) (map[string]interface{}, error) {
    timeout := defaultExternalTimeout
    if spec.Timeout != "" {
        if d, err := time.ParseDuration(spec.Timeout); err == nil {
            timeout = d
        }
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // ... connect, query, handle errors ...

    result := "42"
    return map[string]interface{}{
        "result": result,
        "raw":    map[string]interface{}{"value": result},
        "error":  "",
        "called": "true",
    }, nil
}
```

---

## Step 3 — Register in `protocol.go`

Add a case to `newProtocolClient`:

```go
func newProtocolClient(protocol orktypes.ExternalProtocol) ProtocolClient {
    switch protocol {
    case "", orktypes.ProtocolHTTP:
        return &httpProtocolClient{}
    case orktypes.ProtocolPrometheus:
        return &prometheusClient{}
    // ...
    case orktypes.ProtocolMyProtocol:
        return &myProtocolClient{}
    default:
        return &httpProtocolClient{}
    }
}
```

---

## Step 4 — Wire validation

In `validate.go`, add a case to the `switch call.Protocol` block:

```go
case orktypes.ProtocolMyProtocol:
    if call.Query == "" {
        return fmt.Errorf("%s: query is required for protocol: myprotocol (e.g. \"...\")", location)
    }
    if call.Body != "" || call.Method != "" || len(call.Headers) > 0 || call.ExpectedStatus != 0 {
        return fmt.Errorf("%s: body, method, headers, and expectedStatus are HTTP-only fields", location)
    }
```

Also add the constant to `validateProtocol`:

```go
func validateProtocol(location string, p orktypes.ExternalProtocol) error {
    switch p {
    case "", orktypes.ProtocolHTTP, orktypes.ProtocolPrometheus,
        orktypes.ProtocolRedis, orktypes.ProtocolPostgres, orktypes.ProtocolMongo,
        orktypes.ProtocolKafka, orktypes.ProtocolMyProtocol:
        return nil
    }
    return fmt.Errorf("...")
}
```

---

## Step 5 — Add a fixture

Create `pkg/external/fixtures/<protocol>/` following the pattern in `fixtures/redis/` or `fixtures/prometheus/`:

- `katalog.yaml` — a katalog that calls the protocol and surfaces the result in `status.fields`
- `README.md` — setup instructions, a Docker Compose snippet for the dependency, what to inspect

Run `ork validate -f katalog.yaml` before committing.

---

## Checklist

- [ ] Protocol constant in `pkg/types/external.go`
- [ ] `Fetch()` implementation in `pkg/external/<protocol>.go`
- [ ] `newProtocolClient` case in `protocol.go`
- [ ] `validateProtocol` and `switch call.Protocol` cases in `validate.go`
- [ ] Result map documented in `docs/02-result-maps.md`
- [ ] Fixture in `pkg/external/fixtures/<protocol>/`
- [ ] `ork validate` passes on the fixture katalog

---

**← Back** [04 — The Result Cache](04-cache.md) · [README](README.md)
