# pkg/note/fixture

Living integration fixture for the kubernetes-family note functions.

## Why this exists

Notes in `kubernetes.go`, `kube_replica.go`, `kube_container.go`, `kube_job.go`,
and `kube_service.go` take `map[string]interface{}` from the dynamic client as
input. Unit tests construct these maps by hand — but real API responses differ in
subtle ways: extra metadata fields, different numeric types after JSON
round-tripping, absent optional fields that are zero in the real API.

This fixture closes that gap. It creates a real Deployment, Service, and Job in a
live cluster, then exercises every kubernetes-family note against the actual API
response. Status fields hold the results — observable via `kubectl get noteprobe
<name> -o yaml`.

## Adding a new kubernetes-family note

When you add a note to `kubernetes.go`, `kube_replica.go`, `kube_container.go`,
`kube_job.go`, or `kube_service.go`:

1. **Add a `status.fields` entry** to `katalog.yaml`:
   ```yaml
   - path: myNewNote
     value: "{{ myNewNote .children.deployment }}"
   ```

2. **Run locally** to verify it returns the expected value:
   ```bash
   make test-fixture-note
   ```
   or manually:
   ```bash
   kubectl apply -f pkg/note/fixture/crd.yaml
   ork bundle --file pkg/note/fixture/katalog.yaml | kubectl apply -f -
   helm install orkestra ./charts/orkestra --namespace default --wait
   kubectl apply -f pkg/note/fixture/cr.yaml
   kubectl get noteprobe my-probe -o yaml -w
   ```

3. **Clean up:**
   ```bash
   cd pkg/note/fixture && bash cleanup.sh
   ```

## CI

The `fixture-note` job in `.github/workflows/validate-pr.yml` runs this fixture
on every PR that touches `pkg/note/`. It spins up a kind cluster, installs
Orkestra via Helm, applies the fixture, and asserts `status.phase` is set.
