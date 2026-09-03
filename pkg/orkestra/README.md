# pkg/orkestra

`orkestra` is the process lifecycle manager. It registers `domain.Komponent` services, starts them in registration order, and shuts them down in reverse order on `SIGINT` or `SIGTERM`.

```go
ork := orkestra.NewOrkestra(30*time.Second, logLevel)
ork.Register([]domain.Komponent{kubeclient, event, konductor, health})
ork.Start(ctx)
```

## Komponent contract

Any service that participates in the lifecycle implements `domain.Komponent`:

```go
type Komponent interface {
    Name() string
    Start(ctx context.Context) error
    Shutdown(ctx context.Context)
    Started() bool
}
```

Start order matches registration order. Shutdown order is reversed — the last-started service is the first to stop. The event handler is always shut down last, regardless of registration position, so that other services can still emit events during their shutdown.

## Post-start hooks

Some services must start only after the full komponent list is running (e.g. leader election, which needs the kube client up first):

```go
ork.AddPostStartHook(konductorComp, func(ctx context.Context) {
    ko.Start(ctx)
})
```

Post-start hooks run in goroutines after all `Start` calls succeed.

## Shutdown hooks

Cleanup that must happen after all komponents have stopped (TLS cert deletion, webhook configuration removal, RBAC cleanup) registers via `OnShutdown`:

```go
ork.OnShutdown(func(ctx context.Context) {
    certmanager.DeleteCertificateAndSecret(ctx, ...)
})
```

Hooks run sequentially in registration order, within the shutdown timeout. If the timeout is exceeded mid-way, remaining hooks are skipped and the process exits.

## Graceful shutdown timeout

`NewOrkestra(timeout, logLevel)` sets the maximum time allowed for all komponents and hooks to finish shutting down. If exceeded at any point, `Orkestra` stops waiting and closes the `done` channel. The main goroutine calls `ork.Wait()` which blocks on this channel.
