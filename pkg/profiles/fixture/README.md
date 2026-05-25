# pkg/profiles/fixture

Living integration fixture for Orkestra profiles.

## Why this exists

Unit tests verify expansion logic in isolation. This fixture verifies that the expanded values land correctly on real Kubernetes resources — right CPU/memory on the Deployment, right security context on the Pod spec, right probe timings on the container.

Apply a CR, run the operator, and inspect the child resources directly on the cluster.

---

## Katalogs

All katalogs use the same `ProfileProbe` CRD. One CRD is enough to probe every profile family.

| File | Profiles covered |
|---|---|
| `katalog-resource.yaml` | tiny, small, medium, large, burst, steady, compute-heavy, memory-heavy |
| `katalog-security.yaml` | baseline, restricted, hardened (container + pod security) |
| `katalog-probes.yaml` | fast, standard, patient, slow-start |
| `katalog-autoscale.yaml` | burst, steady, batch, latency-sensitive, cost-optimized |

---

## Running a probe

```bash
cd pkg/profiles/fixture

# Run the katalog for the profile family you want to verify.
# crdFile and crFiles are embedded — Orkestra applies the CRD and CR automatically:
ork run -f katalog-resource.yaml    # resource profiles
ork run -f katalog-security.yaml   # security profiles
ork run -f katalog-probes.yaml     # probe profiles
ork run -f katalog-autoscale.yaml  # autoscale profiles

# Inspect child Deployments:
kubectl get deployments -o yaml | grep -A 10 "resources:"

# Clean up:
bash cleanup.sh
```

---

## Verifying expansion

### Resource profiles

```bash
kubectl get deployment my-probe-medium -o jsonpath='{.spec.template.spec.containers[0].resources}'
```

Expected for `medium`:
```json
{"limits":{"cpu":"1","memory":"1Gi"},"requests":{"cpu":"250m","memory":"256Mi"}}
```

### Security profiles

```bash
kubectl get deployment my-probe-hardened -o jsonpath='{.spec.template.spec.containers[0].securityContext}'
```

Expected for `hardened`:
```json
{"allowPrivilegeEscalation":false,"readOnlyRootFilesystem":true,"runAsNonRoot":true,"capabilities":{"drop":["ALL"]}}
```

### Probe profiles

```bash
kubectl get deployment my-probe-fast -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}'
```

Expected for `fast`:
```json
{"initialDelaySeconds":5,"periodSeconds":10,"failureThreshold":2,"successThreshold":1,"timeoutSeconds":5}
```

---

## Adding a profile

When you add a profile name to any family, add a deployment entry to the matching katalog file:

```yaml
- name: "{{ .metadata.name }}-xlarge"
  image: "{{ .spec.image }}"
  port: "{{ .spec.port }}"
  resources:
    profile: xlarge
  reconcile: true
```

Run `ork run -f katalog-resource.yaml` and verify the Deployment's resource block matches the definition in `pkg/profiles/resource.go`.
