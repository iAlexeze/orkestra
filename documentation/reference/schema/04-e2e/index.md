# E2E

!!! note "E2E is not a Kubernetes CRD"
    `kubectl apply` will not work. Orkestra kinds are consumed by the `ork` CLI and runtime — not by the Kubernetes API server. [Your CRD is enough](/blog/your-crd-is-enough/).

An `E2E` is a declarative end-to-end test for a Katalog. It tells Orkestra exactly what to apply, which cluster to use, and what the expected state is after each step.

```text
Motif     — smallest reusable unit
    ↓
Katalog   — operator declaration
    ↓
Komposer  — platform declaration
    ↓
E2E       — end-to-end test for a Katalog
```

`ork validate` checks the structure. `ork e2e` runs it.

---

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E

metadata:
  name: hello-website-e2e
  description: Deploy a website, verify it comes up, verify cleanup on delete.

spec:
  katalog: ./katalog.yaml
  crd: ./crd.yaml
  cr: ./cr.yaml

  cluster:
    provider: kind
    name: ork-e2e
    reuse: false

  setup:                          # optional
    apply:
      - ./setup.yaml
    wait:
      - kind: Secret
        name: my-secret
        namespace: default
        timeout: 15s

  expect:
    - name: Deployment ready
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          name: hello-website
          namespace: default
          ready: true

    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: Deployment
          name: hello-website
          namespace: default
          count: 0

imports:                          # optional — run other E2E files after this one
  - ./auth-e2e.yaml
  - path: ./infra-e2e.yaml
    freshCluster: true
```

---

## Lifecycle

```text
cluster ready
    ↓
CRD applied
    ↓
setup.apply  — prerequisite manifests applied
    ↓
setup.helm   — prerequisite charts installed
    ↓
setup.wait   — block until prerequisites exist/ready
    ↓
Katalog loaded — operator starts watching
    ↓
CR applied  ← "cr-applied" checkpoints run here
    ↓
CR deleted  ← "cr-deleted" checkpoints run here
    ↓
Cluster torn down (if reuse: false)
```

---

## Validation

```bash
ork validate -f e2e.yaml
```

Catches: missing required fields, unknown `after` values, unknown `provider` values, unreachable file paths for `katalog`, `crd`, `cr`, and `setup.apply` entries.

---

## Try it

```bash
ork init --pack beginner
cd beginner/01-hello-website
ork e2e
```

This scaffolds the beginner example pack, moves into the first example, and runs the full E2E test — creates a kind cluster, applies the operator, applies the CR, checks the Deployment is ready, deletes the CR, and verifies cleanup. The whole cycle takes about 90 seconds.

To keep the cluster for inspection after the test:

```bash
ork e2e --keep-cluster
```

### Run a full pack

Every example pack ships with a root `e2e.yaml`. Run the full suite in one command:

```bash
ork e2e -f examples/beginner/e2e.yaml --use-current
ork e2e -f examples/intermediate/e2e.yaml --use-current
```

Orkestra installs once before the suite, each test updates the bundle in place, Orkestra uninstalls once at the end.

### Discover and run without a root file

`./...` finds every leaf test automatically — no root `e2e.yaml` needed:

```bash
ork e2e ./examples/beginner/...          # all beginner tests
ork e2e ./... --skip external            # everything except external gate tests
ork e2e ./... --dry-run                  # list what would run, no cluster created
```

See [06-discovery.md](06-discovery.md).

---

## Where to go

| Page | Covers |
|------|--------|
| [01-spec.md](01-spec.md) | `katalog`, `crd`, `cr`, `cluster`, `customOperator` |
| [02-setup.md](02-setup.md) | `setup.apply`, `setup.helm`, `setup.wait` |
| [03-expect.md](03-expect.md) | `expect` — resources, commands, after, timeout |
| [04-imports.md](04-imports.md) | `imports` — test suites, `wait:`, cluster strategy, pure aggregators |
| [05-custom-operator.md](05-custom-operator.md) | `customOperator: true` — test any operator without Orkestra |
| [06-discovery.md](06-discovery.md) | `./...` discovery, `--wait`, `--skip`, `--dry-run` |

---

## See also

- [Katalog schema](../02-katalog/01-top-level.md) — the Katalog under test
- [E2E in the Orkestra Registry](../../../orkestra-registry/04-e2e.md) — how E2E gates pattern publication
- [ork e2e CLI reference](../../cli/08-e2e.md)
- [Writing your first E2E](../../../getting-started/06-writing-your-first-e2e.md)
