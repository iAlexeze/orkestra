# Patterns

Five patterns covering the most common reasons teams reach for `external:`. Each shows the minimal Katalog snippet that makes it work and explains the key design choice.

---

## Health gate

Gate a Deployment on an upstream service being healthy. When the health check fails the Deployment is not created or updated — no broken app, no partial rollout. The phase state machine surfaces the failure in status so it is visible in the Control Center.

```yaml
onReconcile:
  external:
    - name: healthCheck
      url: "{{ .spec.healthCheckUrl }}"   # full URL — no path appended
      expectedStatus: 200
      continueOnError: true               # reconcile continues — failure visible in status
      timeout: 5s

  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      when:
        - field: external.healthCheck.status
          equals: "200"

status:
  fields:
    - path: phase
      value: "Degraded"
      when:
        - field: external.healthCheck.called
          equals: "true"
        - field: external.healthCheck.status
          notEquals: "200"

    - path: phase
      value: "Ready"
      when:
        - field: external.healthCheck.status
          equals: "200"
        - field: "{{ allReplicasReady .children.deployment }}"
          equals: "true"

    - path: lastHealthCheck
      value: "{{ .external.healthCheck.status }}"
```

Use `spec.healthCheckUrl` (the full URL) rather than a base `serviceUrl` with `/health` appended. Each CR declares exactly which endpoint to call — production operators use internal standards like `/healthz`, `/ready`, or `/actuator/health`. The field name makes the intent explicit.

`continueOnError: true` means a failed health check surfaces in `status.phase` rather than as a reconcile error. Use `continueOnError: false` when the Deployment must never exist without a passing check — the reconcile halts and `Ready=False` is written to the condition.

**Try it:**
```bash
ork init --pack use-cases
cd external/01-health-gate
ork run --dev-server   # GET /health → 200, GET /status/503 → 503

kubectl apply -f cr-dev-healthy.yaml
kubectl apply -f cr-dev-degraded.yaml
```

---

## Dynamic config injection

Fetch a JSON config blob from a config server on every reconcile. Embed the response body into a ConfigMap. The Deployment mounts the ConfigMap — the app always sees current config without a pod restart or a redeployment.

```yaml
onReconcile:
  external:
    - name: appConfig
      url: "{{ .spec.serviceUrl }}/config/{{ .metadata.name }}"
      continueOnError: true   # config unavailable → retain the last written ConfigMap
      timeout: 5s

  configMaps:
    - name: "{{ .metadata.name }}-config"
      data:
        app.json: "{{ .external.appConfig.body }}"
      when:
        - field: external.appConfig.called
          equals: "true"
        - field: external.appConfig.error
          operator: notExists

  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      volumes:
        - name: app-config
          configMap:
            name: "{{ .metadata.name }}-config"
      volumeMounts:
        - name: app-config
          mountPath: /etc/config

status:
  fields:
    - path: configFresh
      value: "true"
      when:
        - field: external.appConfig.called
          equals: "true"
        - field: external.appConfig.error
          operator: notExists
```

The `when:` condition on the ConfigMap means the config is only overwritten when the call succeeds. A transient config service outage leaves the last-written config in place — the Deployment keeps running without interruption.

**Try it:**
```bash
ork init --pack use-cases
cd external/02-config-inject
ork run --dev-server   # GET /config/:name → static JSON blob

kubectl apply -f cr.yaml
kubectl get configmap my-app-config -o jsonpath='{.data.app\.json}' | jq .
```

---

## Image signing — "once per image change" with rejection tracking

Call a signing service when the image changes. Gate the Deployment on the signed status. Track both successful signs (`signedImage`) and definitive rejections (`rejectedImage`) in status — different status fields, different gates, different behaviors on the next reconcile.

```yaml
onReconcile:
  external:
    - name: signImage
      url: "{{ .spec.serviceUrl }}/sign"
      method: POST
      body: '{"image": "{{ .spec.image }}", "namespace": "{{ .metadata.namespace }}"}'
      token: "$IMAGE_SIGNING_TOKEN"
      expectedStatus: 200
      continueOnError: true   # rejection details must be visible in status
      timeout: 15s
      when:
        - field: status.signedImage
          notEquals: "{{ .spec.image }}"    # skip if already signed
        - field: status.rejectedImage
          notEquals: "{{ .spec.image }}"    # skip if definitively rejected for this image

  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      when:
        - field: status.signedImage
          equals: "{{ .spec.image }}"       # only deploy confirmed-safe images

status:
  fields:
    # Phase: Signing → waiting for first sign attempt
    - path: phase
      value: "Signing"
      when:
        - field: status.signedImage
          notEquals: "{{ .spec.image }}"

    # Phase: SigningRejected — reads from persistent status, survives reconciles where call is skipped
    - path: phase
      value: "SigningRejected"
      when:
        - field: status.rejectedImage
          equals: "{{ .spec.image }}"

    # Phase: SigningUnavailable — 5xx, transient, gate stays open, next reconcile retries
    - path: phase
      value: "SigningUnavailable"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          prefix: "5"

    # Phase: Ready — image signed, all replicas up
    - path: phase
      value: "Ready"
      when:
        - field: status.signedImage
          equals: "{{ .spec.image }}"
        - field: "{{ allReplicasReady .children.deployment }}"
          equals: "true"

    # Written after successful sign — closes the call gate until spec.image changes
    - path: signedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          equals: "200"

    # 4xx → definitive rejection → lock out retries for this exact image
    # Recovery: change spec.image → rejectedImage != spec.image → gate reopens
    - path: rejectedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          prefix: "4"

    # Surfaces the rejection reason — clears explicitly when signing succeeds
    - path: lastSigningError
      value: "{{ .external.signImage.error }}"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          notEquals: "200"

    - path: lastSigningError
      value: ""
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          equals: "200"
```

**Why `continueOnError: true` here:** signing failure is a policy decision with meaningful information (status code, rejection reason). That information needs to reach status fields. With `continueOnError: false`, the reconcile halts before status is written — the rejection is only visible as a raw `Ready=False` condition message, not as structured status fields.

**Why two gates on the call:** `signedImage != spec.image` alone retries the signing service on every reconcile after a rejection — expensive and pointless for a deterministic 403. Adding `rejectedImage != spec.image` closes the gate after a 4xx. A 5xx does not write `rejectedImage`, so the gate stays open and the reconcile retries naturally — the reconcile loop is the retry mechanism.

**Try it:**
```bash
ork init --pack use-cases
cd external/03-image-signing
ork run --dev-server   # POST /sign → 200 for most images, 403 for nginx:not-secure

kubectl apply -f cr-reject.yaml   # nginx:not-secure → SigningRejected, no Deployment
kubectl patch webapp my-app --type=merge -p '{"spec":{"image":"nginx:1.25"}}'
# → sign succeeds → Deployment created
```

---

## Sequential chained calls

Fetch a short-lived token, then use it in the next call. The resolver is updated after every call so later calls can reference earlier results via template expressions in their `url:`, `token:`, or `body:` fields.

```yaml
onReconcile:
  external:
    - name: tokenFetch
      url: "{{ .spec.serviceUrl }}/auth/token"
      method: POST
      body: '{"client_id": "{{ .metadata.name }}", "namespace": "{{ .metadata.namespace }}"}'
      token: "$CLIENT_SECRET"
      continueOnError: false      # no token = nothing to authenticate with, halt here
      timeout: 5s

    - name: resourceCheck
      url: "{{ .spec.serviceUrl }}/resources/{{ .metadata.name }}"
      token: "{{ .external.tokenFetch.body }}"   # result of the previous call
      continueOnError: true
      timeout: 5s

  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      when:
        - field: external.tokenFetch.status
          equals: "200"
        - field: external.resourceCheck.status
          equals: "200"
```

If `tokenFetch` fails (`continueOnError: false`), the reconcile halts and `resourceCheck` never runs. The token is not available — there is nothing to pass forward.

The `token: "{{ .external.tokenFetch.body }}"` expression on `resourceCheck` resolves at call time, after `tokenFetch` has completed and its result has been injected into the resolver. The same mechanism works for `url:` and `body:` — any field in a later call can reference any earlier call's result.

**Try it:**
```bash
ork init --pack use-cases
cd external/04-chained
ork run --dev-server   # POST /auth/token → "dev-token-abc123", GET /resources/:name → resource stub

kubectl apply -f cr.yaml
```

---

## Feature flag rollout — external call drives a resource attribute

Read a live flag from a feature flag service on every reconcile. Use the result to drive `replicas` directly — not as a gate condition, but as a resource attribute. The cluster converges to the correct replica count within one reconcile cycle of the flag changing.

```yaml
onCreate:
  external:
    - name: flags
      url: "{{ .spec.serviceUrl }}/flags/{{ .metadata.name }}/v2Enabled"
      method: GET
      continueOnError: true   # flag service down → degrade to baseline, keep running
      timeout: 5s

  # Full capacity when flag is on
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true
      when:
        - field: external.flags.body
          equals: "true"

  # Baseline when flag is off or flag service is unavailable
  # notEquals: "true" also catches empty body on service outage — safe degradation
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "1"
      reconcile: true
      when:
        - field: external.flags.body
          notEquals: "true"

status:
  fields:
    - path: v2Enabled
      value: "{{ .external.flags.body }}"
      when:
        - field: external.flags.called
          equals: "true"

    - path: activeReplicas
      value: "{{ readyReplicas .children.deployment }}"
```

Two deployment entries target the same name with different replica counts. Exactly one fires per reconcile — the `when:` conditions are mutually exclusive. This is declared under `onCreate` with `reconcile: true` on each entry: `onCreate` runs every reconcile, and `reconcile: true` tells Orkestra to update the existing Deployment (not just create it) when the active entry changes. The second entry catches flag service outages: an empty body from a failed call does not equal `"true"`, so the operator degrades to baseline safely rather than leaving the cluster at stale capacity.

This call fires on every reconcile intentionally — the flag can change at any time. Check `orkestra_external_calls_total` in metrics to see the call count grow as reconciles run.

**Try it:**
```bash
ork init --pack use-cases
cd external/05-feature-flags
ork run --dev-server   # GET /flags/:name/:flag → "true" by default

kubectl apply -f cr.yaml
# Deployment: 5 replicas

curl -X POST http://localhost:9999/flags/my-app/v2Enabled/toggle
# Wait one reconcile → Deployment: 1 replica

curl -X POST http://localhost:9999/flags/my-app/v2Enabled/toggle
# Wait one reconcile → Deployment: 5 replicas
```
