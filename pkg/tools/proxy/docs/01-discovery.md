# Service discovery

`ork proxy` discovers Orkestra components by label — not by hardcoded service name. This means it works regardless of Helm release name or custom chart values.

## The label

Every Orkestra service and deployment carries:

```
orkestra.orkspace.io/komponent: runtime | control-center | gateway
```

`FindService` lists Services with this label in the target namespace and returns the first match. If no service is found the component is not deployed — `ork proxy` prints `⊗ not deployed` and continues without that component.

## Runtime: Lease-based pod targeting

The Runtime's `/katalog` API reflects live reconciler state, which only the leader holds. Forwarding to a follower would return stale or empty data. `ork proxy` therefore forwards Runtime traffic directly to the leader pod, not to the Service.

`ResolveRuntimePod` reads the `coordination.k8s.io/v1` Lease named `orkestra-konductor` in the target namespace and returns `spec.holderIdentity`. This is the pod name of the current leader.

The resolution uses the Go Kubernetes client directly — not a kubectl subprocess. `pkg/tools/proxy` has no dependency on `pkg/registry/e2e`.

## Control Center and Gateway: Service pod targeting

Control Center and Gateway are stateless — any running replica serves the same content. `ResolvePod` gets the service's `.spec.selector`, lists matching pods, and returns the first one in `Running` phase.

The forward targets the pod directly (the Kubernetes portforward API operates on pods, not services). The port used is the service port, which equals the container port for all Orkestra components.
