# Contributing to Orkestra

Thank you for your interest in contributing to Orkestra.

Orkestra is a declarative Kubernetes operator runtime. Operators are described as YAML — the runtime, gateway, and control center turn those descriptions into running reconciliation loops, webhook servers, and an observable UI.

---

## Start here

Full contribution guides live in [`documentation/contributing/`](documentation/contributing/):

| Guide | What it covers |
|-------|---------------|
| [index](documentation/contributing/index.md) | Overview and where to go |
| [Codebase map](documentation/contributing/codebase-map.md) | Which packages belong to which binary; how to navigate the code |
| [Registry](documentation/contributing/contributing-registry.md) | Add or improve resource types in `pkg/resources` |
| [Control Center](documentation/contributing/contributing-controlcenter.md) | Improve the web UI — metrics, CR status, multi-instance views |
| [Providers](documentation/contributing/contributing-providers.md) | Add or extend cloud and database providers |
| [Rollback](documentation/contributing/contributing-rollback.md) | Complete the rollback implementation |
| [Examples](documentation/contributing/contributing-examples.md) | Add example operator packs |
| [Publishing a new pack](documentation/contributing/publishing-a-new-pack.md) | Exact checklist for adding a new examples pack without breaking CI |

---

## Quick start

```bash
# Clone
git clone https://github.com/orkspace/orkestra.git
cd orkestra

# Build
make ork

# Test
go test ./...
```

---

## Pull requests

1. Fork and create a branch from `main`.
2. Read the relevant guide above.
3. Run `go test ./...` and confirm it passes.
4. Open a PR with a short description of what and why. Link any related issues.

---

## Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(registry): add DaemonSet handler
fix(reconciler): correct window-based rollback trigger
docs(contributing): add provider contribution guide
```

---

## Code style

- `gofmt` is enforced by CI.
- Exported functions must have a doc comment.
- Wrap errors with context: `fmt.Errorf("creating deployment: %w", err)`.
- No comments that describe *what* the code does — only *why* when the reason is non-obvious.

---

## Code of Conduct

This project follows the [Orkestra Code of Conduct](CODE_OF_CONDUCT.md). Report unacceptable behaviour to [ialexeze@gmail.com](mailto:ialexeze@gmail.com).

---

## Questions?

Open a [GitHub Discussion](https://github.com/orkspace/orkestra/discussions).
