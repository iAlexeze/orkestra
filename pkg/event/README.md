# pkg/event

`event` wraps the Kubernetes event recorder so that reconcilers and control-plane components can emit `corev1.Event` objects without depending directly on `client-go/tools/record`.

```go
e := event.NewEvent(kube)
e.Start(ctx)

e.Eventf(obj, corev1.EventTypeNormal, "Reconciled", "created %s", name)
```

## Event recording

`Eventf` records events asynchronously — it spawns a goroutine per call and tracks in-flight work with a `sync.WaitGroup`. This prevents event recording from blocking the reconcile loop on API server latency.

Events are discarded silently once `Shutdown()` is called, so no goroutine leaks after the broadcaster is shut down.

## Shutdown

`Shutdown()` sets a stopped flag, waits for all in-flight events to flush (with the given context deadline as a timeout), then calls `broadcaster.Shutdown()` to release the underlying connection. A context timeout produces a warning log but is otherwise clean.

## NoopRecorder

`NoopRecorder` implements the `Recorder` interface and discards all events. Used by `ork simulate` and tests that do not need Kubernetes event side-effects:

```go
type Recorder interface {
    Eventf(obj runtime.Object, eventType, reason, messageFmt string, args ...interface{})
}
```

Swap `*event.Event` for `*event.NoopRecorder` anywhere a `Recorder` is accepted.
