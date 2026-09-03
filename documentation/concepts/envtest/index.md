# Declarative Integration Testing

`ork simulate --envtest` runs the same reconciler loop as `ork simulate`, but against a real `kube-apiserver` + `etcd` spun up locally — no cluster, no deployed operator, no install step.

The three testing tiers:

```text
ork simulate                 # unit — fake clients, milliseconds
ork simulate --envtest       # integration — real API server, ~3–5s
ork e2e                      # system — real cluster, real pods
```

Same simulate.yaml. Same `expect:` block. The flag decides which backend runs.

---

## Why a middle tier

`ork simulate` runs the reconciler against fake in-memory clients. It is fast and catches template and logic errors, but it bypasses the API server entirely — CRD schema violations, status subresource enforcement, and irregular plural resource names go unnoticed.

`ork e2e` catches everything, but it requires a running cluster, image pulls, and a deployed operator. It is the right gate before pushing, not the right tool for iterating.

`--envtest` fills the gap. It gives you a real API server in ~3 seconds — schema enforcement, real watch streams, real REST mapper — without a cluster or a deployed operator.

| | `ork simulate` | `ork simulate --envtest` | `ork e2e` |
|---|---|---|---|
| Requires cluster | No | No | Yes |
| Real CRD schema enforcement | No | Yes | Yes |
| Real watch streams | No | Yes | Yes |
| Admission webhooks | No | No | Yes |
| Real pod scheduling | No | No | Yes |
| Speed | Milliseconds | ~3–5s | Minutes |
| Best for | Template correctness | API-server correctness | System correctness |

---

## Setup

No install step. On first run, `--envtest` downloads `kube-apiserver` and `etcd` to `~/.ork/envtest-bins` automatically. Subsequent runs use the cache.

The default Kubernetes version is `1.32`. To pin a different version:

```bash
ork simulate -f simulate.yaml --envtest --k8s-version 1.31
```

To use a CI-managed cache or pre-existing binaries, set `KUBEBUILDER_ASSETS` — `--k8s-version` is ignored when this is set:

```bash
export KUBEBUILDER_ASSETS=/path/to/bins
ork simulate -f simulate.yaml --envtest
```

---

## What it requires in simulate.yaml

`--envtest` installs your CRD into the API server before starting the loop. Declare the CRD file in simulate.yaml:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: my-operator
spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  crd: ./crds/my-operator.yaml
  cycles: 3
  expect:
    steady: true
    noErrors: true
```

For multiple CRDs, use `crdFiles` and `crFiles`:

```yaml
spec:
  crdFiles:
    - ./crds/website.yaml
    - ./crds/database.yaml
```

```yaml
spec:
  crFiles:
    - ./crs/website.yaml
    - ./crs/database.yaml
```

Running `--envtest` without `crd` or `crdFiles` is an error.

---

## Where to go next

- [`ork simulate`](../simulate/index.md) — the unit testing tier
- [`ork e2e`](../e2e/index.md) — the system testing tier
- [simulate.yaml envtest fields](../simulate/07-envtest.md) — crd/crdFiles fields, crFiles
- [`ork simulate` CLI reference](../../reference/cli/05-simulate.md)
