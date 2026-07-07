# pkg/proxy

Port-forwarding for Helm-deployed Orkestra installations.

`ork proxy` discovers Orkestra services by the `orkestra.orkspace.io/komponent` label and establishes port-forwards for each selected component. For the Runtime it forwards to the leader pod (via Lease); for Control Center and Gateway it forwards to any running pod.

| I want to… | Go to |
|---|---|
| Understand how services are discovered and how the Runtime leader pod is resolved | [docs/01-discovery.md](docs/01-discovery.md) |
| Understand the forward loop, reconnection, and port conflict handling | [docs/02-forward.md](docs/02-forward.md) |
