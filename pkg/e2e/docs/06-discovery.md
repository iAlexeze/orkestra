# Discovery Mode — `ork e2e ./...`

`ork e2e ./...` recursively walks a directory, finds all `*e2e.yaml` leaf files, and runs them as a suite — the same pattern as `go test ./...`.

```bash
ork e2e ./...                               # all e2e files from cwd down
ork e2e ./examples/beginner/...             # scoped to a subtree
ork e2e ./... --wait 2s                     # 2s between each discovered test
ork e2e ./... --skip vendor,testdata        # exclude paths matching patterns
ork e2e ./... --dry-run                     # list files without running
```

## Discovery rules

- Finds all files whose name ends with `e2e.yaml` (e.g. `my-e2e.yaml`, `operator-e2e.yaml`)
- **Skips pure aggregators** — files with `imports:` but no `spec:`. They exist for explicit suite runs; discovery runs leaf tests only to avoid double-running
- Sorts by full path for deterministic order
- Skips paths matching any `--skip` pattern (matched against the full path as a component or glob)

## `--wait`

Injects `wait: <d>` on every import except the first. Use when tests leave cluster state that needs time to clear between runs:

```bash
ork e2e ./... --wait 5s
```

## `--skip`

Comma-separated patterns. Each pattern is matched against path components and the file's base name:

```bash
ork e2e ./... --skip vendor,testdata,external/07-vault
```

`--skip` is necessary in practice. Without it, `./...` will sweep in slow suites, vendor directories, and tests that require special infrastructure.

## `--dry-run`

Lists the files that would be discovered without running any tests:

```bash
$ ork e2e ./... --dry-run
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

## How it works

The CLI builds a synthetic in-memory aggregator (`kind: E2E` with the discovered paths as imports), writes it to a temp file, and passes it to the normal runner. Everything after that is identical to running a hand-written root `e2e.yaml`.
