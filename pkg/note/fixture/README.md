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

Each katalog is a self-contained probe for a specific resource family.

| File | Kind | Notes covered |
|---|---|---|
| `katalog.yaml` | `NoteProbe` | kubernetes, replica, container, service |
| `katalog-pods.yaml` | `PodProbe` | pod enrichment on Deployment (`enrich: [pods]`) |
| `katalog-statefulset.yaml` | `StatefulSetProbe` | pod enrichment on StatefulSet, `podByOrdinal` |
| `katalog-service.yaml` | `ServiceProbe` | endpoint enrichment on Service (`enrich: [endpoints]`) |
| `katalog-job.yaml` | `JobProbe` | job lifecycle + pod enrichment on Job (`enrich: [pods]`) |
| `katalog-warnings.yaml` | `WarningsProbe` | warning event enrichment on any resource (`enrich: [events]`) |
| `katalog-pvc.yaml` | `PVCProbe` | PVC lifecycle notes + enriched PV notes (`enrich: [pvc]`) |
| `katalog-ingress.yaml` | `IngressProbe` | Ingress notes — host, IP, rules, TLS |
| `katalog-hpa.yaml` | `HPAProbe` | HPA replica scaling notes |

### `katalog.yaml` — NoteProbe

Covers the general kubernetes-family notes that work on any child resource:
`resourceExists`, `allReplicasReady`, `containerImage`, `serviceClusterIP`,
`endpointsReady`, and the full replica + kubernetes note set.

Does **not** require `enrich: [pods]` or `enrich: [endpoints]`.

### `katalog-pods.yaml` — PodProbe

Covers the pod note family: `podNames`, `podIPs`, `podPhases`, `podNodes`,
`podCount`, `readyPodCount`, `podMaxRestarts`, `hasCrashingPod`.

Requires `enrich: [pods]` — only notes that depend on `_pods` enrichment live here.

### `katalog-statefulset.yaml` — StatefulSetProbe

Covers StatefulSet-specific patterns: ordered membership (`podNames`, `podIPs`),
and `podByOrdinal` for surfacing the primary member's name and IP.

### `katalog-service.yaml` — ServiceProbe

Covers enriched endpoint notes: `hasEndpoints`, `serviceEndpoints`,
`serviceEndpointCount`, `serviceFirstEndpoint`.

Requires `enrich: [endpoints]`.

### `katalog-job.yaml` — JobProbe

Covers job lifecycle notes (`jobSucceeded`, `jobFailed`, `jobActive`) and
enriched pod notes on jobs: `jobFirstExitCode`, `jobActivePodNames`,
`jobSucceededPodNames`, `jobFailedPodNames`.

Requires `enrich: [pods]`.

---

## Running a probe

```bash
# Apply CRD and start Orkestra (once per cluster):
kubectl apply -f pkg/note/fixture/crd.yaml
ork bundle --file pkg/note/fixture/katalog.yaml | kubectl apply -f -
helm upgrade --install orkestra ./charts/orkestra --namespace default --wait

# Apply the CR for whichever probe you want to run:
kubectl apply -f pkg/note/fixture/cr.yaml          # NoteProbe

# Watch status populate:
kubectl get noteprobe my-probe -o yaml -w
kubectl get podprobe my-probe -o yaml -w
kubectl get statefulsetprobe my-probe -o yaml -w

# Clean up:
cd pkg/note/fixture && bash cleanup.sh
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
every PR touching `pkg/note/`. It spins up a kind cluster, installs Orkestra via
Helm, applies the fixture, and asserts `status.phase` is set.
