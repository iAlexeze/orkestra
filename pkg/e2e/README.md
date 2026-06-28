# pkg/e2e

`e2e` runs a declarative end-to-end test against a real Kubernetes cluster. Give it a spec file and it orchestrates the full lifecycle — cluster creation, CRD apply, operator install, CR apply, expectation checking, and cleanup — the same way locally and in CI.

```bash
ork e2e -f e2e.yaml
ork e2e ./...                    # discover and run all *e2e.yaml files recursively
ork e2e ./examples/beginner/...  # scoped discovery
ork e2e init                     # scaffold e2e.yaml from the current Katalog
ork e2e init --suite             # write a suite aggregator from discovered leaf files
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand the spec file format and all fields | [docs/01-spec.md](docs/01-spec.md) |
| Understand how expectations are evaluated (resources, commands, polling) | [docs/02-expectations.md](docs/02-expectations.md) |
| Understand the full run pipeline (what happens in order) | [docs/03-pipeline.md](docs/03-pipeline.md) |
| Understand cluster lifecycle (kind, reuse, context restore, shared Orkestra) | [docs/04-cluster.md](docs/04-cluster.md) |
| Compose test suites with imports and the `wait:` field | [docs/05-imports.md](docs/05-imports.md) |
| Use `./...` discovery mode, `--wait`, `--skip`, `--dry-run` | [docs/06-discovery.md](docs/06-discovery.md) |
| Test any Kubernetes workload without Orkestra (`custom.target: kubernetes`) | [docs/07-kubernetes-target.md](docs/07-kubernetes-target.md) |

---

## Adding a new `kubectl:` subcommand

> [!IMPORTANT]
> Every new `kubectl:` subcommand requires changes in four places and a fixture entry. Missing any one of them means the feature is incomplete.

**1. Type — `pkg/types/e2e.go`**

Add a struct for the new subcommand:

```go
type E2EKubectlMyCmd struct {
    // required fields...
    Equals            string `yaml:"equals,omitempty"`
    NotEquals         string `yaml:"notEquals,omitempty"`
    OutputContains    string `yaml:"outputContains,omitempty"`
    OutputNotContains string `yaml:"outputNotContains,omitempty"`
}
```

Add a slice for it on `E2EKubectl`:

```go
MyCmd []E2EKubectlMyCmd `yaml:"my-cmd,omitempty"`
```

**2. Runner — `pkg/e2e/verify.go`**

Add `checkKubectlMyCmd(ctx, e, workDir)` and call it from `checkKubectl`:

```go
for i, e := range k.MyCmd {
    if err := checkKubectlMyCmd(ctx, e, workDir); err != nil {
        return fmt.Errorf("kubectl.my-cmd[%d]: %w", i, err)
    }
}
```

Mutations (`apply`, `patch`) run before reads (`get`, `logs`, `describe`, `exec`, `port-forward`) so changes take effect before assertions check them.

**3. Validator — `pkg/e2e/validate.go`**

Add `validateKubectlMyCmd(loc, e)` and call it from `ValidateKubectl`:

```go
for j, e := range exp.Kubectl.MyCmd {
    errs = append(errs, validateKubectlMyCmd(loc("my-cmd", j), e)...)
}
```

**4. Schema doc — [`documentation/reference/schema/04-e2e/07-kubectl.md`](../../documentation/reference/schema/04-e2e/07-kubectl.md)**

Add a `## kubectl.my-cmd` section with the field table and a usage example.

**5. Fixture — [`pkg/e2e/fixture/e2e.yaml`](./fixture/e2e.yaml)**

Add a checkpoint that exercises the new subcommand end-to-end against the `E2EProbe` operator. See [fixture/README.md](fixture/README.md) for details.
