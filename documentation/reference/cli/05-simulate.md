# ork simulate

Run the operator reconcile loop against a fake in-memory cluster — no real Kubernetes cluster required. Useful for verifying that a declarative Katalog creates the right resources in the right order.

```bash
ork simulate
```

`--file` defaults to `katalog.yaml` (then `komposer.yaml`) and `--cr` defaults to `cr.yaml` — both are optional when your files use the standard names.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `katalog.yaml` | Path to katalog or komposer file |
| `--cr` | | `cr.yaml` | Path to the CR YAML file to reconcile |
| `--crd` | | *(all)* | CRD name to simulate. Defaults to all CRDs in the Katalog. |
| `--cycles` | | `10` | Maximum number of reconcile cycles per CRD |

## Examples

```bash
# Standard layout — no flags needed
ork simulate

# Non-standard filenames
ork simulate -f my-operator.yaml --cr my-cr.yaml

# Simulate a specific CRD only
ork simulate --crd website

# Run up to 5 reconcile cycles
ork simulate --cycles 5
```

## Output

```text
Simulating website/my-site

  Cycle 1:
    + deployments/my-site
    + services/my-site
    ~ status/my-site
  Cycle 2:
    ~ status/my-site
  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 189ms
```

`+` green — resource created. `~` yellow — resource updated. `-` red — resource deleted. Consecutive identical cycles are collapsed. `status/...` appears every cycle — that is expected.

`--cycles N` runs exactly N cycles. Steady state is noted but does not stop the run.

## How it works

- Spins up a fake in-memory Kubernetes cluster using `k8s.io/client-go/kubernetes/fake`
- Runs the same `GenericReconciler` used in production
- Records all `create`, `update`, `delete`, `patch` operations per cycle
- Advances Deployment status to `Available` after cycle 1 to unblock state machines
- Detects steady state when two consecutive cycles produce identical operations

`ork simulate` does not require a real cluster and does not read from or write to `~/.kube/config`.
