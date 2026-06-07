# Discovery Mode and Dry Run

## `ork e2e ./...`

Recursively walks a directory, finds all `*e2e.yaml` leaf files, and runs them as a suite — the same pattern as `go test ./...`. No root `e2e.yaml` file needed.

```bash
ork e2e ./...                                   # all *e2e.yaml files from cwd down
ork e2e ./examples/beginner/...                 # scoped to a subtree
ork e2e ./... --wait 2s                         # 2s between each discovered test
ork e2e ./... --skip vendor,testdata            # exclude matching paths
```

### Discovery rules

- Finds all files whose name ends with `e2e.yaml` — e.g. `operator-e2e.yaml`, `e2e.yaml`
- **Skips pure aggregators** — files that have `imports:` but no `spec:`. Those exist for explicit suite runs. Discovery runs leaf tests only to avoid running the same test twice.
- Sorted by full path for deterministic order

### `--wait`

Sleeps for the given duration before each discovered test (except the first). Use when tests leave cluster state that needs time to clear — webhook deregistration, namespace termination, cert provisioning:

```bash
ork e2e ./... --wait 5s
```

### `--skip`

Comma-separated path patterns. Matched against the full file path — either as a path component or as a glob against the file name. Skip slow suites, infrastructure-heavy tests, or vendor directories:

```bash
ork e2e ./... --skip cr-e2e.yaml,vendor,testdata,external/07-vault
```

`--skip` is necessary in practice. Without it, `./...` will sweep in tests that require special infrastructure or that are intentionally slow.

---

## `--dry-run`

Shows what would run without executing any tests or touching any cluster.

**Single file:**

```bash
ork e2e -f e2e.yaml --dry-run
```

Runs `ork validate` on the file and prints the structured validation output — field summary, setup, expectations. Identical to `ork validate -f e2e.yaml`.

**Discovery mode:**

```bash
ork e2e ./... --dry-run
```

Lists the files that would be discovered:

```text
→ Would run 12 e2e file(s) under .

  examples/beginner/01-hello-website/e2e.yaml
  examples/beginner/02-with-serviceaccount/e2e.yaml
  examples/beginner/03-secret-copy/e2e.yaml
  examples/beginner/03b-configmap-copy/e2e.yaml
  examples/intermediate/04-multi-resource/e2e.yaml
  examples/intermediate/05-when-conditions/e2e.yaml
  examples/intermediate/06-komposer-basic/e2e.yaml
  examples/intermediate/07-crd-file/e2e.yaml
  examples/intermediate/08-state-machine/e2e.yaml
  examples/use-cases/enrich/01-pod-health/e2e.yaml
  ... 2 more
```

No cluster is created. No tests are run.

---

→ Back: [05-custom-operator.md](05-custom-operator.md) | [Schema index](index.md)
