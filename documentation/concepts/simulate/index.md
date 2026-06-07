# Simulate

`ork simulate` runs the operator reconcile loop against a fake in-memory cluster. No Kubernetes, no `kubectl`, no network — results in milliseconds.

---

## Pages

| Page | What it covers |
|------|----------------|
| [How it works](01-how-it-works.md) | The fake cluster model, same reconciler, steady state detection |
| [simulate.yaml](02-simulate-kind.md) | The recommended entry point — schema, `expect:`, assert mode, validation |
| [Running simulate](03-running.md) | All invocation forms: `--cr`, `-f e2e.yaml`, `./...`, flags |
| [Hooks and constructors](04-hooks-and-constructors.md) | Custom binary, registry wiring, what the standard binary shows |
| [Aggregator mode](05-aggregator.md) | `./...` discovery and aggregator simulate/e2e files |
| [Limitations](06-limitations.md) | What simulate cannot cover and what to use instead |

---

## The key property

`ork simulate` does not approximate what the reconciler does. It *is* the reconciler — the same `GenericReconciler` that runs in production, wired to a fake in-memory Kubernetes store. Template expressions, `when:` conditions, `onCreate`/`onReconcile` order, and status propagation all execute identically.

Use simulate as the fast inner loop while writing an operator. Use `ork e2e` as the outer gate before pushing.

| | `ork simulate` | `ork e2e` |
|---|---|---|
| Requires cluster | No | Yes |
| Runs real reconciler | Yes | Yes |
| Tests webhooks | No | Yes |
| Tests external calls | No (inactive) | Yes |
| Speed | Milliseconds | Minutes |
| Best for | Template correctness | System correctness |

→ See also: [`ork simulate` CLI reference](../../reference/cli/05-simulate.md)
