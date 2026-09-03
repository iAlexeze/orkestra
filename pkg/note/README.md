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

## User-defined notes

Teams can define their own notes in a Katalog's `notes:` block or in a Motif imported via `spec.imports`. User-defined notes call built-ins and each other, and are available everywhere built-in notes are.

```yaml
notes:
  functions:
    - name: fullImage
      expression: "{{ .spec.image }}:{{ .spec.tag | default \"latest\" }}"
    - name: serviceHost
      expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

See [documentation/concepts/notes/](../../documentation/concepts/notes/index.md) for the full guide.

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
| Parse resource quantities | [11 — Quantity Notes](docs/11-quantity.md) |
| Inspect replica / rollout state | [12 — Replica Notes](docs/12-replica.md) |
| Gate on Job and CronJob lifecycle | [13 — Job Notes](docs/13-job.md) |
| Surface Service networking details | [14 — Service Notes](docs/14-service.md) |
| Read raw object fields | [15 — Field Notes](docs/15-fields.md) |
| Work with network addresses | [16 — Net Notes](docs/16-net.md) |
| Inspect enriched pod state | [17 — Pod Notes](docs/17-pods.md) |
| Surface Kubernetes warnings | [18 — Warning Notes](docs/18-warnings.md) |
| Read HPA scaling state | [19 — HPA Notes](docs/19-hpa.md) |
| Inspect Ingress routing | [20 — Ingress Notes](docs/20-ingress.md) |
| Read PVC / StorageClass details | [21 — PVC Notes](docs/21-pvc.md) |
| Inspect node topology | [22 — Node Notes](docs/22-node.md) |
| Read StatefulSet revision state | [23 — StatefulSet Notes](docs/23-statefulset.md) |
| Navigate ReplicaSet ownership | [24 — ReplicaSet Notes](docs/24-replicaset.md) |
| Work with timestamps and durations | [26 — Time Notes](docs/26-time.md) |
| Gate on Kubernetes label/annotation format | [31 — Kubernetes Validation Notes](docs/31-kube-validation.md) |
| Check emails, git URLs, images, JSON, ports | [32 — Validation Notes](docs/32-validation.md) |

## Adding a new note

**1. Implement**

- Identify the domain (`cron`, `strings`, `math`, `types`, `conditional`, `kubernetes`, `replica`, `container`, `job`, `service`, …)
- Add the function to the appropriate `*.go` file
- Register it in that file's `xxxNotes()` function — it is automatically included via `note.Map()`

> [!IMPORTANT]
> Document the note in the corresponding `docs/` file under `pkg/note/docs/` using this [template](./docs/_template.md).
> The [note catalog](./catalog_generated.go) is **generated automatically** from these markdown files by `make generate-notes` — it must **never be edited directly**. 
> A note that is not documented in `docs/` will not appear in `ork notes`, even if it is registered and working.

**Contract:** handle empty/nil input with a safe zero value, not a panic. Return `(value, error)` for functions that can meaningfully fail; return just `value` for infallible ones.

**2. Unit test**

Write a test in the corresponding `*_test.go` file. Cover the nil/empty path, the happy path, and any edge cases. For most note families (strings, math, conditional, types, cron, random, collections) this is sufficient — the note is done.

**3. Kubernetes-family notes also require a katalog entry**

Notes in `kubernetes.go`, `kube_replica.go`, `kube_container.go`, `kube_job.go`, and `kube_service.go` take live Kubernetes objects as input (`map[string]interface{}` from the dynamic client). Unit tests use hand-crafted maps — they cannot cover missing fields, null values, or schema variations that only appear with real API responses. The note katalog closes that gap.

Add a `status.fields` entry to [`fixture/katalog.yaml`](fixture/katalog.yaml):

```yaml
- path: myNewNote
  value: "{{ myNewNote .children.deployment }}"
```

Then verify it against a live cluster:

```bash
kubectl apply -f pkg/note/fixture/crd.yaml
ork run -f pkg/note/fixture/katalog.yaml   # keep running in one terminal

kubectl apply -f pkg/note/fixture/cr.yaml
kubectl get noteprobe my-probe -o yaml -w  # watch until phase: Ready
```

Confirm the field appears with the expected value. Clean up when done:

```bash
cd pkg/note/fixture && bash cleanup.sh
```

---

The [`fixture/`](fixture/) katalog is the living e2e test for the kubernetes note family. It runs in CI on every change to `kube_*.go` or `kubernetes.go`.
