# filter fixture

Living fixture for `preReconcile.enqueueGate`. Verifies that the enqueue gate
drops objects at the informer layer — before they enter the work queue —
when `spec.active` is false.

The key distinction from `preReconcile.reconcileGate`: the kordinator never sees
the object, so CRD health stays **healthy** (not `gated`). Queue depth stays 0.
The reconciler is never called.

```bash
ork validate -f pkg/runtime/informer/fixture/filter/katalog.yaml
ork simulate -f pkg/runtime/informer/fixture/filter/simulate.yaml
ork e2e      -f pkg/runtime/informer/fixture/filter/e2e.yaml
```
