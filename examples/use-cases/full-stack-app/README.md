# Advanced Usecases — Orkestra Declarative Patterns

These examples show what Orkestra can express declaratively that previously required custom Go code. Each example names the Go pattern being replaced, shows the declarative equivalent, and tells you where to watch it happen in the Control Center.

This directory uses a `komposer.yaml` that imports all six example katalogs and starts them together as one operator. Each example also has its own `katalog.yaml` — to run them individually, enter the subfolder and running `ork validate` and `ork run` there.

---

## Step 1 — Apply the CRDs

```bash
kubectl apply -f 01-multi-region/crd.yaml
kubectl apply -f 02-external-gate/crd.yaml
kubectl apply -f 03-cross-crd/crd.yaml
kubectl apply -f 04-once-secret/crd.yaml
kubectl apply -f 05-anyof/crd.yaml
kubectl apply -f 06-full-stack/crd.yaml
```

---

## Step 2 — Validate

```bash
ork validate
```

Expected:
```
✓ multi-region-app    kind: MultiRegionApp   group: advanced.orkestra.io / version: v1alpha1
✓ gated-app           kind: GatedApp         group: advanced.orkestra.io / version: v1alpha1
✓ managed-database    kind: ManagedDatabase  group: advanced.orkestra.io / version: v1alpha1
✓ database-backed-app kind: DatabaseBackedApp group: advanced.orkestra.io / version: v1alpha1
✓ secure-app          kind: SecureApp        group: advanced.orkestra.io / version: v1alpha1
✓ flex-app            kind: FlexApp          group: advanced.orkestra.io / version: v1alpha1
✓ full-stack-app      kind: FullStackApp     group: advanced.orkestra.io / version: v1alpha1
```

---

## Step 3 — Start the operator

`--dev-server` starts a mock HTTP server on `:9999`. Example 02 (external health gate) uses it for `/health` and `/flags/:name` calls — no real service needed:

```bash
ork run --dev-server
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **advanced-usecases** to watch all CRDs reconcile in real time.

---

## Example 01 — Multi-Region Deployment (forEach)

**Go pattern replaced:** A constructor that ranged over `spec.regions` and called `client.Create()` for each Deployment. Typically 40–60 lines of Go including error handling and drift detection.

**Katalog (no Go required):**

```yaml
deployments:
  - name: "{{ .metadata.name }}-{{ .item }}"
    image: "{{ .spec.image }}"
    replicas: "{{ .spec.defaultReplicas }}"
    forEach:
      field: spec.regions
      as: item
```

**Apply:**
```bash
kubectl apply -f 01-multi-region/cr.yaml
```

**Watch it:**
1. Control Center → advanced-usecases → multi-region-app → View Resources
2. Click the CR — Children section shows `my-app-us-east-1`, `my-app-eu-west-1`, `my-app-ap-southeast-1`
3. All three Deployments are owned by the CR and managed for drift

---

## Example 02 — External Health Gate (external:)

**Go pattern replaced:** A hook that called an external health endpoint before creating resources. Required an HTTP client, timeout handling, error interpretation — typically 80 lines of Go.

**Katalog:**

```yaml
external:
  - name: healthCheck
    url: "{{ .spec.serviceUrl }}/health"
    expectedStatus: 200
    continueOnError: false
    timeout: 5s
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    when:
      - field: external.healthCheck.status
        equals: "200"
```

**Apply:**
```bash
kubectl apply -f 02-external-gate/cr.yaml
```

**Watch it:**
1. Control Center → gated-app → View Resources
2. Phase shows `Waiting` until health check fires, then `Ready` when it passes
3. Deployment appears only after health check returns 200
4. Events tab shows `external call healthCheck: 200`
5. The feature flags call (`/flags/:name`) is non-blocking — logged if it fails

---

## Example 03 — Cross-CRD Dependency (cross:)

**Go pattern replaced:** A hook that queried the API server for another CRD's CR status using `client.Get()`. Required importing the other CRD's types, handling not-found, re-queuing manually. 50–100 lines.

**Katalog:**

```yaml
cross:
  - crd: managed-database
    selector:
      name: "{{ .metadata.name }}-db"
      namespace: "{{ .metadata.namespace }}"
    as: db
onReconcile:
  deployments:
    - name: "{{ .metadata.name }}"
      when:
        - field: "{{ phase .cross.db }}"
          equals: "Ready"
  configMaps:
    - name: "{{ .metadata.name }}-config"
      data:
        DB_HOST: "{{ .cross.db.status.endpoint }}"
```

**Apply:**
```bash
kubectl apply -f 03-cross-crd/database-cr.yaml    # apply Database first
kubectl apply -f 03-cross-crd/application-cr.yaml # Application observes it
```

**Watch it:**
1. Control Center → application → View Resources → click `my-app`
2. Status shows the `phase` field — watch it change as Database reconciles
3. The Deployment appears only when the Database CR reaches `Ready`
4. The ConfigMap contains the database endpoint — auto-updated on every reconcile

The cross observation reads from the in-memory informer cache. Zero API calls.
The Application reconciler and Database reconciler never communicate directly.

---

## Example 04 — Idempotent Secret Generation (once:)

**Go pattern replaced:** A hook that called `crypto/rand`, stored the result in a Secret, and checked existence before generating. 60–80 lines of Go.

**Katalog:**

```yaml
onCreate:
  secrets:
    - name: "{{ .metadata.name }}-credentials"
      once: true
      data:
        password: "{{ randomAlphanumeric 32 }}"
        apiKey:   "{{ randomHex 16 }}"
        jwt:      "{{ randomBase64 32 }}"
```

**Apply:**
```bash
kubectl apply -f 04-once-secret/cr.yaml
```

**Watch it:**
1. Control Center → secure-app → View Resources → click `my-secure-app`
2. Children section shows the Secret
3. Delete the Secret manually — next reconcile recreates it with a new password
```bash
kubectl delete secret my-secure-app-credentials
```
4. Normal reconciles: password never changes (`once:` guard skips generation)

---

## Example 05 — OR Conditions (anyOf:)

**Go pattern replaced:** Multiple if-conditions checking whether a resource should be created based on any one of several states.

**Katalog:**

```yaml
jobs:
  - name: "{{ .metadata.name }}-cleanup"
    anyOf:
      - field: status.phase
        equals: "Failed"
      - field: status.phase
        equals: "Succeeded"
  - name: "{{ .metadata.name }}-notify"
    when:
      - field: spec.notify
        equals: "true"
    anyOf:
      - field: status.phase
        equals: "Running"
      - field: status.phase
        equals: "Failed"
      - field: status.phase
        equals: "Succeeded"
```

**Apply:**
```bash
kubectl apply -f 05-anyof/cr.yaml
```

**Watch it:**
1. Control Center → flex-app → View Resources → click your CR
2. While `Running`: no Jobs

Then enable notifications to trigger the notify Job:
```bash
kubectl patch flexapp my-flex-app --type=merge -p '{"spec":{"notify":"true"}}'
```

3. After `spec.notify: true` and phase transitions: notify Job appears, then cleanup Job appears

```bash
kubectl get job
```
Expectation:
```
NAME                  COMPLETIONS   DURATION   AGE
my-flex-app-cleanup   1/1           4s         17s
my-flex-app-notify    1/1           6s         27s
```

---

## Example 06 — Full Stack (all patterns combined)

One CR, all patterns at once: forEach + external + cross + once + anyOf.

**What gets created:**
- 3 regional Deployments (forEach over `spec.regions`)
- 1 Secret with generated credentials (once:)
- 1 ConfigMap with database endpoint (cross: from ManagedDatabase)
- Only after health check passes (external:)
- Cleanup Job appears when phase becomes Succeeded or Failed (anyOf:)

**Apply:**

> Apply the ManagedDatabase from Example 03 first — the full-stack app depends on it.

```bash
kubectl apply -f 06-full-stack/cr.yaml
```

**Watch it:**
1. Control Center → full-stack-app → View Resources → click `my-app`
2. Phase progresses: `Pending` → `WaitingForDatabase` → `HealthCheckFailed` or `Ready`
3. Children: 3 Deployments + Secret + ConfigMap — all owned and drift-managed
4. Events: external call, cross observation, resource creation — full lifecycle visible

---

## Patterns reference

| Pattern | Katalog field | What it replaces |
|---|---|---|
| Iterate over list | `forEach: {field: ..., as: ...}` | `for _, item := range obj.Spec.Items` |
| External health gate | `external: [{name, url, expectedStatus}]` | `http.Get + if status != 200` |
| Cross-CRD read | `cross: [{crd, selector, as}]` | `client.Get + if notFound` |
| Random secret | `once: true + randomAlphanumeric` | `crypto/rand + secretExists check` |
| OR conditions | `anyOf: [{field, equals}]` | `if a \|\| b` |
| AND conditions | `when: [{field, equals}]` | `if a && b` |
| Ordered phases | `status.fields` with `when:` | State machine in constructor |
| Child status | `{{ readyReplicas .children.deployment }}` | `client.Get` for child resource |

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
