# pkg/note/fixture

Living integration fixture for the Orkestra note functions.

## Why this exists

Notes take `map[string]interface{}` from the dynamic client as input. Unit tests
construct these maps by hand — but real API responses differ in subtle ways:
extra metadata fields, different numeric types after JSON round-tripping, absent
optional fields that resolve to zero in the real API.

This fixture closes that gap. Apply a CR, watch status populate, and every note's
output is visible directly on the object — no log-diving required.

---

## Katalogs

All katalogs use the same `NoteProbe` CRD. A single CRD with a flexible spec
is enough to probe every note family without registering separate CRD types per
resource family.

| File | Notes covered | Enrichment required |
|---|---|---|
| `katalog.yaml` | kubernetes, replica, container, service | none |
| `katalog-pods.yaml` | pod enrichment on Deployment | `enrich: [pods]` |
| `katalog-statefulset.yaml` | StatefulSet pod enrichment, `podByOrdinal` | `enrich: [pods]` |
| `katalog-service.yaml` | endpoint enrichment on Service | `enrich: [endpoints]` |
| `katalog-job.yaml` | job lifecycle + pod enrichment on Job | `enrich: [pods]` |
| `katalog-warnings.yaml` | warning event enrichment on any resource | `enrich: [events]` |
| `katalog-pvc.yaml` | PVC lifecycle notes + enriched PV notes | `enrich: [pvc]` |
| `katalog-ingress.yaml` | Ingress notes — host, IP, rules, TLS | none |
| `katalog-hpa.yaml` | HPA replica scaling notes | none |

### `katalog.yaml`

Covers the general kubernetes-family notes that work on any child resource:
`resourceExists`, `allReplicasReady`, `containerImage`, `serviceClusterIP`,
`endpointsReady`, and the full replica + kubernetes note set.

### `katalog-pods.yaml`

Covers the pod note family: `podNames`, `podIPs`, `podPhases`, `podNodes`,
`podCount`, `readyPodCount`, `podMaxRestarts`, `hasCrashingPod`.

### `katalog-statefulset.yaml`

Covers StatefulSet-specific patterns: ordered membership (`podNames`, `podIPs`),
and `podByOrdinal` for surfacing the primary member's name and IP.

### `katalog-service.yaml`

Covers enriched endpoint notes: `hasEndpoints`, `serviceEndpoints`,
`serviceEndpointCount`, `serviceFirstEndpoint`.

### `katalog-job.yaml`

Covers job lifecycle notes (`jobSucceeded`, `jobFailed`, `jobActive`) and
enriched pod notes on jobs: `jobFirstExitCode`, `jobActivePodNames`,
`jobSucceededPodNames`, `jobFailedPodNames`.

### `katalog-warnings.yaml`

Covers warning event notes: `hasWarnings`, `warningCount`, `firstWarningReason`,
`firstWarning`. Events recorded on pods owned by a workload are also
aggregated — container failures (ImagePullBackOff, OOMKilled) show up here.

---

## Running a probe

```bash
cd pkg/note/fixture

# Run the katalog for the probe family you want to test.
# crdFile and crFiles are embedded — Orkestra applies the CRD and CR automatically:
ork run -f katalog-<resource_type>.yaml

# Watch status populate:
kubectl get noteprobe my-probe -o yaml -w

# Clean up:
bash cleanup.sh
```

---

## Adding a note

When you add a note to any `kube_*.go` or `kubernetes.go` file, pick the katalog
that matches the note's resource family and add a `status.fields` entry:

```yaml
- path: myNewNote
  value: "{{ myNewNote .children.deployment }}"
```

Routing rule by resource family:

- Pod notes → `katalog-pods.yaml`
- StatefulSet ordinal notes → `katalog-statefulset.yaml`
- Endpoint notes → `katalog-service.yaml`
- Job lifecycle / pod notes → `katalog-job.yaml`
- Warning event notes → `katalog-warnings.yaml`
- PVC/PV notes → `katalog-pvc.yaml`
- Ingress notes → `katalog-ingress.yaml`
- HPA notes → `katalog-hpa.yaml`
- Everything else → `katalog.yaml`

---

## CI

The `fixture-note` job in `.github/workflows/validate-pr.yml` runs the fixture on
every PR touching `pkg/note/`. It spins up a kind cluster, runs `ork run -f katalog.yaml`,
and asserts `status.phase` is set.
