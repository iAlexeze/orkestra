# ork delete

Delete Orkestra infrastructure resources.

```bash
ork delete <subcommand> [flags]
```

---

## Subcommands

### `ork delete cluster`

Delete a local kind cluster created by [`ork create cluster`](./create.md).

```bash
ork delete cluster [flags]
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--name <name>` | `ork-playground` | Name of the kind cluster to delete |

#### Examples

Delete the default playground cluster:

```bash
ork delete cluster
```

Delete a named cluster:

```bash
ork delete cluster --name ork-e2e
```

#### Behavior

- Calls `kind delete cluster --name <name>`.
- Safe to call when the cluster does not exist — kind exits cleanly.
- Does not affect any other `kubectl` contexts.

---

## Notes

- `ork delete cluster` only removes kind clusters. It does not delete remote clusters or other Kubernetes resources.
- To create a cluster: [`ork create cluster`](./create.md).
