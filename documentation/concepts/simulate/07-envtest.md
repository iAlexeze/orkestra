# Running with envtest

`ork simulate --envtest` runs the same simulate.yaml against a real `kube-apiserver` + `etcd` spun up locally — no cluster, no deployed operator. The reconciler, CR, and `expect:` assertions are unchanged; only the backend switches from fake in-memory clients to a real API server.

```bash
ork simulate -f simulate.yaml            # fake clients, instant
ork simulate -f simulate.yaml --envtest  # real API server, ~3–5s startup
ork e2e -f e2e.yaml                      # real cluster, operator deployed
```

---

## What it catches that regular simulate cannot

| Behaviour | simulate | simulate --envtest |
|---|---|---|
| Real CRD schema validation | No | Yes |
| Status subresource (`.status` requires a status patch) | No | Yes |
| Irregular plural resource names | No | Yes — real REST mapper |
| Real watch stream delivery | No | Yes |
| Admission webhook enforcement | No | No — use `ork e2e` |

---

## Setup

No manual install. On first run, `--envtest` downloads `kube-apiserver` and `etcd` to `~/.ork/envtest-bins` automatically. Subsequent runs use the cache — startup is ~3s.

The default Kubernetes version is `1.32`. To pin a different version:

```bash
ork simulate -f simulate.yaml --envtest --k8s-version 1.31
```

To use pre-existing binaries or a CI-managed cache, set `KUBEBUILDER_ASSETS` — when set, `--k8s-version` is ignored:

```bash
export KUBEBUILDER_ASSETS=/path/to/bins
ork simulate -f simulate.yaml --envtest
```

---

## simulate.yaml requirements

`--envtest` needs the CRD schema to install into the API server before the reconcile loop starts. Declare the CRD file(s) in your simulate.yaml:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: my-operator-envtest
spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  crd: ./crds/my-operator.yaml    # single CRD file
  cycles: 3
  expect:
    steady: true
    noErrors: true
```

For operators with multiple CRDs:

```yaml
spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  crdFiles:
    - ./crds/website.yaml
    - ./crds/database.yaml
```

Running without `crd` or `crdFiles` and passing `--envtest` is an error — the API server needs the schema.

---

→ Back to: [Limitations](06-limitations.md) | [simulate.yaml](02-simulate-kind.md)
