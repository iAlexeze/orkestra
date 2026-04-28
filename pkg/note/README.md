# pkg/note

Notes are pure transformation functions available in every Orkestra template expression — `status.fields`, `normalize.spec`, `onCreate`, `onReconcile`, mutation rules, validation rules, and conversion paths.

The name is intentional. In music, notes are the atomic units from which everything is composed — precise, combinable, universally understood. In Orkestra, notes serve the same role.

## What a note is

- **Pure** — same input always produces the same output
- **Safe** — handles empty/nil input without panicking
- **Stateless** — no I/O, no external calls, no shared state

Notes are not hooks. Hooks are for external API calls and side effects. Notes are for data transformation.

## Where notes work

Every `{{ }}` expression in any Katalog field:

```yaml
normalize:
  spec:
    schedule: "{{ cronFromMap .spec.schedule }}"

status:
  fields:
    - path: phase
      value: "{{ boolTernary .spec.suspend \"Suspended\" \"Active\" }}"

onCreate:
  secrets:
    - name: "{{ .metadata.name }}-creds"
      once: true
      data:
        password: "{{ randomAlphanumeric 32 }}"
```

## Developer documentation

Complete documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Work with strings | [01 — String Notes](docs/01-strings.md) |
| Do arithmetic on spec fields | [02 — Math Notes](docs/02-math.md) |
| Express conditional values | [03 — Conditional Notes](docs/03-conditional.md) |
| Inspect or convert field types | [04 — Type Notes](docs/04-types.md) |
| Work with cron expressions | [05 — Cron Notes](docs/05-cron.md) |
| Generate secrets | [06 — Random Notes](docs/06-random.md) |
| Work with lists and maps | [07 — Collection Notes](docs/07-collections.md) |
| Access fields safely with defaults | [08 — Safe Access Notes](docs/08-safe-access.md) |
| Navigate child/cross-CRD objects | [09 — Kubernetes Notes](docs/09-kubernetes.md) |
| Read container images and env vars | [10 — Container Notes](docs/10-container.md) |

## Adding a new note

**1. Implement**

- Identify the domain (`cron`, `strings`, `math`, `types`, `conditional`, `kubernetes`, `replica`, `container`, `job`, `service`, …)
- Add the function to the appropriate `*.go` file
- Register it in that file's `xxxNotes()` function — it is automatically included via `note.Map()`
- Document it in the corresponding `docs/` file

**Contract:** handle empty/nil input with a safe zero value, not a panic. Return `(value, error)` for functions that can meaningfully fail; return just `value` for infallible ones.

**2. Unit test**

Write a test in the corresponding `*_test.go` file. Cover the nil/empty path, the happy path, and any edge cases. For most note families (strings, math, conditional, types, cron, random, collections) this is sufficient — the note is done.

**3. Kubernetes-family notes also require a katalog entry**

Notes in `kubernetes.go`, `kube_replica.go`, `kube_container.go`, `kube_job.go`, and `kube_service.go` take live Kubernetes objects as input (`map[string]interface{}` from the dynamic client). Unit tests use hand-crafted maps — they cannot cover missing fields, null values, or schema variations that only appear with real API responses. The note katalog closes that gap.

Add a `status.fields` entry to [`example/katalog.yaml`](example/katalog.yaml):

```yaml
- path: myNewNote
  value: "{{ myNewNote .children.deployment }}"
```

Then verify it against a live cluster:

```bash
kubectl apply -f pkg/note/example/crd.yaml
ork run -k pkg/note/example/katalog.yaml   # keep running in one terminal

kubectl apply -f pkg/note/example/cr.yaml
kubectl get noteprobe my-probe -o yaml -w  # watch until phase: Ready
```

Confirm the field appears with the expected value. Clean up when done:

```bash
cd pkg/note/example && bash cleanup.sh
```

---

The [`example/`](example/) katalog is the living e2e test for the kubernetes note family. It runs in CI on every change to `kube_*.go` or `kubernetes.go`.
