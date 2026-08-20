# Migration Pack

You have a working controller-runtime operator. This pack shows the same WebApp operator expressed five ways — from zero-Go declarative to a full constructor lift — so you can see what you are choosing between, and when to pick each.

```bash
ork init --pack from-controller-runtime
```

---

## The options

| Directory | What you own |
|-----------|-------------|
| `01-declarative` | Nothing — pure YAML, no binary |
| `02-hybrid` | The 10% that templates cannot express |
| `03-hooks-only` | All child resource specs in Go |
| `04-constructor-migration` | Reconcile logic; manager removed |
| `05-constructor-orkestra-resources` | Reconcile logic; resource ops simplified |
| `06-ork-migrate` | Start from an existing operator file |

---

## Where to start

If your operator only creates Kubernetes resources and applies rules — start with `01-declarative`. You may not need Go at all.

If you have an existing `Reconcile` method you want to keep — start with `04-constructor-migration`. Zero changes to your reconciler. Two lines in a constructor are all it costs.

If you want the constructor injected automatically — run `ork migrate` in `06-ork-migrate`. It removes `SetupWithManager` and injects the two-line constructor for you.

→ [Migration Guide](../../guides/migration/index.md) — detailed page per option
