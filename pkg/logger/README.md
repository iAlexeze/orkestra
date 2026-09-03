# pkg/logger

`logger` is a thin wrapper over [zerolog](https://github.com/rs/zerolog) that adds request-scoped context keys and a single `Init` call for global level configuration.

```go
logger.Init("info")  // call once at startup; accepts debug/info/warn/error/fatal/panic

logger.Info().Msg("reconciler started")
logger.Error().Err(err).Msg("reconcile failed")
```

## Level initialisation

`Init(level string)` sets the global zerolog level. Any unrecognised level string defaults to `info`. Call it once in `main` after reading `ORK_ENV` or a `--log-level` flag.

## Context-scoped logging

Reconcile loops and HTTP handlers propagate three values through `context.Context`:

| Key | Type | Added by | Read by |
|-----|------|----------|---------|
| `request_id` | `string` | `logger.WithRequestID(ctx)` | `logger.FromContext(ctx)` |
| `crd` | `string` | `logger.WithCRD(ctx, name)` | `logger.FromContext(ctx)` |
| `resource` | `string` | `logger.WithResource(ctx, key)` | `logger.FromContext(ctx)` |

`FromContext` builds a logger that includes whichever of the three are present:

```go
ctx = logger.WithCRD(ctx, "myApp")
ctx = logger.WithRequestID(ctx)

log := logger.FromContext(ctx)
log.Info().Msg("starting reconcile")
// → {"level":"info","crd":"myApp","request_id":"<uuid>","message":"starting reconcile"}
```

## Direct logging

For code paths without a context, use the package-level functions directly:

```go
logger.Debug().Str("key", val).Msg("details")
logger.Warn().Err(err).Msg("retrying")
logger.Fatal().Err(err).Msg("cannot start")  // calls os.Exit(1)
```
