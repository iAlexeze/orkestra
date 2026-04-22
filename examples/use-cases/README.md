# Advanced Usecases — Orkestra Declarative Patterns

These examples show what Orkestra can express declaratively that previously
required custom Go code. Each example names the Go pattern being replaced,
shows the declarative equivalent, and tells you where to watch it happen
in the Control Center.

---

## Before you start

Apply the CRDs and start Orkestra. Then open the Control Center:

### CRDs
```bash
kubectl apply -f 01-multi-region/crd.yaml
kubectl apply -f 02-external-gate/crd.yaml
kubectl apply -f 03-cross-crd/crd.yaml
kubectl apply -f 04-once-secret/crd.yaml
kubectl apply -f 05-anyof/crd.yaml
kubectl apply -f 06-full-stack/crd.yaml
```

### Orkestra
Follow the steps in [here](../kubebuilder-conversion/README.md#steps) to deploy Orkestra with webhook support for admission control.

```bash
kubectl port-forward svc/orkestra -n orkestra-system 9090:8090 &  # port-forward to view control center

# Accessible here: http://localhost:8090
```

You will see the `advanced use case` Katalog
Click into **advanced-usecases** to watch reconciliation happen in real time.

> [!NOTE]
> All CRDs are currently in **`started`** state. This is expected as no reconciles have occured yet.
---

## Example 01 — Multi-Region Deployment (forEach)

**Go pattern replaced:** A constructor that ranged over `spec.regions` and
called `client.Create()` for each Deployment. Typically 40–60 lines of Go
including error handling and drift detection.

**What you would have written in Go:**
```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var app v1alpha1.MultiRegionApp
    r.Get(ctx, req.NamespacedName, &app)

    for _, region := range app.Spec.Regions {
        deployment := buildDeployment(app, region)
        if err := r.createOrUpdate(ctx, deployment); err != nil {
            return ctrl.Result{}, err
        }
    }
    return ctrl.Result{}, nil
}
```

**Katalog (no Go required):**

```yaml
spec:
  crds:
    multi-region-app:
      operatorBox:
        default: true
        onReconcile:
          deployments:
            - name: "{{ .metadata.name }}-{{ .item }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicasPerRegion }}"
              forEach:
                field: spec.regions
                as: item
```

**Apply the CR:**
```bash
kubectl apply -f 01-multi-region/cr.yaml
```

**Watch it in the Control Center:**
1. Open the Control Center → advanced-usecases → multi-region-app
2. Click **View Resources** — you will see `1` resource the CR
3. Click on the resource to see the 3 deployments created, current phase, and events

The Deployments are named `my-app-us-east-1`, `my-app-eu-west-1`,
`my-app-ap-southeast-1`. All owned by the CR, all managed for drift.

---

## Example 02 — External Health Gate (external:)

**Go pattern replaced:** A hook that called an external health endpoint before
creating resources, returning an error if the service was not ready. Required
an HTTP client, timeout handling, error interpretation — typically 80 lines
of Go in a hook.

**What you would have written in Go:**
```go
func OnReconcile(ctx context.Context, obj *v1alpha1.GatedApp) error {
    resp, err := http.Get(obj.Spec.ServiceURL + "/health")
    if err != nil || resp.StatusCode != 200 {
        return fmt.Errorf("health check failed: %v", err)
    }
    // Now safe to create resources
    return createDeployment(ctx, obj)
}
```

**Katalog:**

```yaml
spec:
  crds:
    gated-app:
      operatorBox:
        default: true
        onReconcile:
          external:
            - name: health-check
              url: "{{ .spec.serviceUrl }}/health"
              expectedStatus: 200
              continueOnError: false
              timeout: 5s
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              when:
                - field: external.health-check.status
                  equals: "200"
```

**Apply:**
```bash
kubectl apply -f 02-external-gate/cr.yaml
```

> [!TIP]
> Apply with an unavialable endpoint first to see the gate work before updating to a reachable URL and reapply.

**Watch it:**
1. Control Center → gated-app → View Resources
2. If the health endpoint returns 200: Deployment appears immediately
3. If the health endpoint is down: Deployment does not appear, reconcile retries and you will see the reconcile error and message
4. Click the resource → Events tab shows "external call health-check: 200"
5. The feature flags check is expected to fail but is non-blocking because of `continueOnError: true`. It is logged

Try bringing the health endpoint down and watch the Deployment disappear on
the next reconcile cycle (within `resync` seconds).

---

## Example 03 — Cross-CRD Dependency (cross:)

**Go pattern replaced:** A hook that queried the API server for another CRD's
CR status, typically using `client.Get()`. Required importing the other CRD's
types, handling not-found, and re-queuing manually. 50–100 lines of Go.

**What you would have written in Go:**
```go
func OnReconcile(ctx context.Context, obj *v1alpha1.Application) error {
    var db v1alpha1.Database
    err := r.client.Get(ctx, types.NamespacedName{
        Name: obj.Name, Namespace: obj.Namespace,
    }, &db)
    if err != nil || db.Status.Phase != "Ready" {
        return fmt.Errorf("database not ready")
    }
    return createDeployment(ctx, obj, db.Status.Endpoint)
}
```

**Katalog (zero API calls — reads from informer cache):**

```yaml
spec:
  crds:
    database:              # CRD 1 — no changes needed
      operatorBox:
        default: true
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}-db"
              image: postgres:15

    application:           # CRD 2 — observes database
      dependsOn:
        database: healthy  # workers don't start until database CRD is healthy
      operatorBox:
        default: true
        cross:
          - crd: database
            selector:
              name: "{{ .metadata.name }}"   # look up same-named Database CR
            as: db
        onReconcile:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              when:
                - field: cross.db.status.phase
                  equals: "Ready"             # gate on Database CR's phase
          configmaps:
            - name: "{{ .metadata.name }}-config"
              data:
                DB_HOST: "{{ .cross.db.status.endpoint }}"  # copy endpoint
```

**Apply:**
```bash
kubectl apply -f 03-cross-crd/database-cr.yaml   # apply Database first
kubectl apply -f 03-cross-crd/application-cr.yaml # Application observes it
```

**Watch it:**
1. Control Center → application → View Resources → click `my-app`
2. Status shows `cross.db.status.phase` — watch it change as Database reconciles
3. The Deployment appears only when the Database CR reaches `Ready`
4. The ConfigMap contains the database endpoint — auto-updated on every reconcile

The cross observation reads from the in-memory informer cache. Zero API calls.
The Application reconciler and Database reconciler never communicate directly.

---

## Example 04 — Idempotent Secret Generation (once:)

**Go pattern replaced:** A hook that called `crypto/rand`, stored the result
in a Secret, and checked existence before generating. The idempotency logic
was always custom per-operator. 60–80 lines of Go.

**What you would have written in Go:**
```go
func OnReconcile(ctx context.Context, obj *v1alpha1.SecureApp) error {
    var secret corev1.Secret
    err := r.client.Get(ctx, types.NamespacedName{
        Name: obj.Name + "-creds", Namespace: obj.Namespace,
    }, &secret)
    if errors.IsNotFound(err) {
        password := generateRandomString(32)
        secret = buildSecret(obj, password)
        return r.client.Create(ctx, &secret)
    }
    return err // already exists — no-op
}
```

**Katalog:**

```yaml
spec:
  crds:
    secure-app:
      operatorBox:
        default: true
        onCreate:
          secrets:
            - name: "{{ .metadata.name }}-credentials"
              once: true                               # ← only evaluated once
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
3. Delete and re-apply the Secret manually — next reconcile recreates it with a NEW password
4. Delete and re-apply the CR — new CR gets a new password
5. Normal reconcile cycles: password never changes (once: guard skips generation)
6. Other resources like Deployments can consume the Secret

The `randomAlphanumeric` note is only evaluated when the Secret does not exist.
Every subsequent reconcile reads `secretExists() → true` and skips the note entirely.

---

## Example 05 — OR Conditions (anyOf:)

**Go pattern replaced:** Multiple if-conditions in a hook function that checked
whether a resource should be created based on any one of several conditions.

**What you would have written in Go:**
```go
func shouldCreateResource(obj *v1alpha1.FlexApp) bool {
    return obj.Status.Phase == "Failed" || obj.Status.Phase == "Succeeded"
}

# And more...
```

**Katalog:**

```yaml
spec:
  crds:
    flex-app:
      operatorBox:
        default: true
        onReconcile:
          jobs:
            # Cleanup job — runs when the app has finished (either outcome)
            - name: "{{ .metadata.name }}-cleanup"
              image: alpine:3.19
              command: ["/bin/sh", "-c", "echo cleanup"]
              anyOf:                              # OR — any one must pass
                - field: status.phase
                  equals: "Failed"
                - field: status.phase
                  equals: "Succeeded"

            # Notification job — similar conditions, but also requires notify: true
            # Starts when phase=Running serving as trigger
            - name: "{{ .metadata.name }}-notify"
              image: alpine:3.19
              command: ["/bin/sh", "-c", "echo notify"]
              when:                              # AND — all must pass
                - field: spec.notify
                  equals: "true"
              anyOf:                             # AND the above, OR these
                - field: status.phase
                  equals: "Running"
                - field: status.phase
                  equals: "Failed"
                - field: status.phase
                  equals: "Succeeded"
```
### Apply, Trigger, Observe

#### 1. Apply the CR
```bash
kubectl apply -f 05-anyof/cr.yaml
```
The FlexApp remains in `Running` until its `status.phase` transitions to `Succeeded`.

---

#### 2. Trigger the notify Job  
Edit the CR and enable notifications:

```yaml
spec:
  notify: true

# Then re-apply
kubectl apply -f 05-anyof/cr.yaml
```

Once `spec.notify` becomes `true`, the **notify Job** appears because:

- `when:` requires `spec.notify == "true"`  
- `anyOf:` is satisfied by the current phase (`Running`, `Failed`, or `Succeeded`)

---

#### 3. Observe the cascade  
As soon as the notify Job completes and reports `Succeeded`, its phase satisfies the cleanup job’s `anyOf:` block:

- `Failed`  
- `Succeeded`

This causes the CR phase to transition to `Succeeded` and the **cleanup Job** to appear immediately afterward.

---

#### 4. Watch it
1. Control Center → `flex-app` → select your CR  
2. **Children** section:  
   - Initially: no Jobs while phase = `Running` - just Deployment  
   - After setting `spec.notify: true`: CR phase changes to `Succeeded` and the notify Job appears  
   - After notify Job succeeds: cleanup Job appears

---

## Example 06 — Full Stack (combining all patterns)

This is the showcase example. One Katalog, one CR apply, everything declarative.

**What gets created:**
- 3 regional Deployments (forEach over spec.regions)
- 1 Secret with generated credentials (once:)
- 1 ConfigMap with database endpoint (cross: from database CRD)
- Only after health check passes (external:)
- Cleanup Job appears when phase becomes Succeeded or Failed (anyOf:)

```yaml
# katalog.yaml — see the file for full declaration
```

**Apply:**
> [!NOTE]
> Make sure you applied [database-cr](./03-cross-crd/database-cr.yaml) in [example 03](README.md/#example-03--cross-crd-dependency-cross) since the full stack app depends on the database

```bash
kubectl apply -f 06-full-stack/cr.yaml
```

**Watch the full lifecycle in the Control Center:**

1. **Open:** `http://localhost:8090/controlcenter`
2. **Select:** advanced-usecases → full-stack-app
3. **Click View Resources** (top right) — shows all CR instances
4. **Click** `my-app` — opens the CR detail page
5. **Status section:** shows `phase`, `regionsDeployed`, `databaseEndpoint`
6. **Children section:** expand to see all 3 Deployments + Secret + ConfigMap
7. **Events section:** shows the reconcile history — external call, cross observation, resource creation
8. **Watch auto-refresh:** every 10 seconds the page updates as reconciliation progresses

The entire lifecycle — from CR apply to all resources created — happens without
writing a single line of Go.

---

## Patterns reference

| Pattern | Katalog field | What it replaces |
|---|---|---|
| Iterate over list | `forEach: {field: ..., as: ...}` | `for _, item := range obj.Spec.Items` |
| External health gate | `external: [{name, url, expectedStatus}]` | `http.Get + if status != 200` |
| Cross-CRD read | `cross: [{kind, selector, as}]` | `client.Get + if notFound` |
| Random secret | `once: true + randomAlphanumeric` | `crypto/rand + secretExists check` |
| OR conditions | `anyOf: [{field, equals}]` | `if a || b` |
| AND conditions | `when: [{field, equals}]` | `if a && b` |
| Provider call | `providers: {aws: [{s3: ...}]}` | AWS SDK v2 in a hook |
| Ordered phases | `status.fields` with `when:` | State machine in constructor |
| Child status | `{{ .children.deployment.status.readyReplicas }}` | `client.Get` for child resource |