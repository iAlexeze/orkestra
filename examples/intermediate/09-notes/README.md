# 09 — Notes

Notes are Orkestra's template vocabulary — pure functions you call inside any `{{ }}` expression. They let you compute values from live CR state without writing Go code.

```bash
ork init --pack intermediate
cd intermediate/09-notes
```

---

## Examples

| Example | What it teaches |
|---------|-----------------|
| [01 — Built-in Notes](01-built-in/README.md) | What notes are. The three most common patterns: reading live cluster state with `allReplicasReady`, fallback values with `default`, and name expressions. |
| [02 — User-Defined Notes](02-user-defined/README.md) | Declaring an inline note and writing its result to a status field. |
| [03 — Notes via Motifs](03-motifs/README.md) | Packaging notes into a Motif for team-wide distribution and OCI publishing via `spec.imports`. |
| [04 — Komposer Override](04-komposer/README.md) | Overriding a Motif note at the Komposer level with an inline `notes:` block. Komposers cannot use `spec.imports` — overrides are always inline. |

**Full concept guide:** [https://orkestra.sh/docs/concepts/notes](https://orkestra.sh/docs/concepts/notes)

---

## Komposer

`komposer.yaml` composes all three katalogs into a single runtime — Website CRDs and Workload CRDs managed together:

```bash
ork validate -f komposer.yaml
ork run -f komposer.yaml
```

## Running a single example

```bash
cd 01-built-in
ork validate
ork run
kubectl apply -f cr.yaml
```

## Simulate (no cluster needed)

Run a single example:

```bash
cd 01-built-in && ork simulate
```

Run the full notes suite:

```bash
ork simulate -f simulate.yaml
```

## E2E

```bash
ork e2e -f e2e.yaml
```
