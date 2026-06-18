# ork create

Create Orkestra infrastructure resources.

```bash
ork create <subcommand> [flags]
```

---

## Subcommands

### `ork create cluster`

Create a local [kind](https://kind.sigs.k8s.io/) cluster for Orkestra development or testing. Downloads `kind` automatically if it is not found in `PATH` and switches `kubectl` to the new cluster's context.

```bash
ork create cluster [flags]
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--name <name>` | `ork-playground` | Name of the kind cluster to create |
| `--provider <provider>` | `kind` | Cluster provider — only `kind` is supported |

#### Examples

Create the default playground cluster:

```bash
ork create cluster
```

Create a named cluster:

```bash
ork create cluster --name ork-e2e
```

#### Behavior

- Creates the kind cluster and waits for it to be ready.
- Switches the active `kubectl` context to the new cluster (`kind-<name>`).
- Subsequent `ork run`, `ork e2e`, and `kubectl` commands operate against this cluster.

---

## Notes

- `ork create cluster` is intended for local development and CI environments, not production.
- Only the `kind` provider is supported. Other providers (minikube, k3s) are not yet available.
