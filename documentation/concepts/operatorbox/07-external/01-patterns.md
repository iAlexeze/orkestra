# Patterns

Six patterns covering the most common reasons teams reach for `external:`. Each shows the minimal Katalog snippet that makes it work and explains the key design choice.

---

## Health gate

Gate a Deployment on an upstream service being healthy. If the health check fails, the Deployment is not created or updated — no broken app, no partial rollout.

```yaml
onReconcile:
  external:
    - name: healthCheck
      url: "{{ .spec.serviceUrl }}/health"
      expectedStatus: 200
      continueOnError: true   # reconcile continues — phase state machine shows the failure
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
```

`continueOnError: true` means a failed health check is visible in status rather than surfacing as a reconcile error. Use `continueOnError: false` when the Deployment must never exist without a passing health check — the reconcile halts and `Ready=False` is written on the condition.

**Try it:**
```bash
ork init --pack use-cases
cd external/01-health-gate
ork run
```

---

## Dynamic config injection

Fetch a JSON config blob from a config server on every reconcile. Embed the response body into a ConfigMap. The Deployment mounts the ConfigMap — the app always sees current config without a pod restart.

```yaml
onReconcile:
  external:
    - name: appConfig
      url: "{{ .spec.serviceUrl }}/config/{{ .metadata.name }}"
      continueOnError: true   # config unavailable → keep the last written ConfigMap
      timeout: 5s

  configMaps:
    - name: "{{ .metadata.name }}-config"
      reconcile: true
      data:
        app.json: "{{ .external.appConfig.body }}"
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
ork run
```

---

## Image signing — "once per image change"

Call a signing service when the image changes. Gate the Deployment on the signed status. Use a status field to remember the last signed image — the call is skipped on every subsequent reconcile until the image changes again.

```yaml
onReconcile:
  external:
    - name: signImage
      url: "{{ .spec.serviceUrl }}/sign"
      method: POST
      body: '{"image": "{{ .spec.image }}"}'
      token: "$IMAGE_SIGNING_TOKEN"
      expectedStatus: 200
      continueOnError: false
      timeout: 15s
      when:
        # Skip the call when the image is already signed.
        - field: status.signedImage
          notEquals: "{{ .spec.image }}"

  deployments:
    - name: "{{ .metadata.name }}"
      when:
        - field: status.signedImage
          equals: "{{ .spec.image }}"

status:
  fields:
    # Written after a successful sign — prevents re-signing on the next reconcile.
    - path: signedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.status
          equals: "200"
```

The pattern — **call → write result to status → gate future calls on status** — generalises to any "call once per spec change" scenario: license checks, DNS record creation, service mesh registration, cluster provisioning.

**Try it:**
```bash
ork init --pack use-cases
cd external/03-image-signing
ork run
```

---

## Sequential chained calls

Fetch a short-lived token, then use it in the next call. The resolver is updated after every call so later calls can reference earlier results via template expressions in their `url:`, `token:`, or `body:` fields.

```yaml
onReconcile:
  external:
    - name: tokenFetch
      url: "{{ .spec.authUrl }}/token"
      method: POST
      token: "$CLIENT_SECRET"
      continueOnError: false   # no token = don't proceed
      timeout: 5s

    - name: resourceCheck
      url: "{{ .spec.serviceUrl }}/resources/{{ .metadata.name }}"
      token: "{{ .external.tokenFetch.body }}"   # uses the previous call's result
      continueOnError: true
      timeout: 5s
```

If `tokenFetch` fails (with `continueOnError: false`), the reconcile halts and `resourceCheck` never runs. This is the correct behaviour — there is nothing to authenticate with.

**Try it:**
```bash
ork init --pack use-cases
cd external/04-chained
ork run
```

---

## Conditional webhook notification

Fire a Slack or Teams webhook only when the phase transitions to a specific state. Use `when:` to gate the call so it does not fire on every reconcile.

```yaml
onReconcile:
  external:
    - name: slackAlert
      url: "$SLACK_WEBHOOK_URL"
      method: POST
      body: '{"text": "{{ .metadata.name }} is {{ .status.phase }} in {{ .metadata.namespace }}"}'
      continueOnError: true
      timeout: 3s
      when:
        - field: status.phase
          equals: "Degraded"
```

The `when:` condition checks the current status before the call runs. The call fires only while the CR is in `Degraded` — once the phase changes, the condition fails and the call is skipped. To fire exactly once on transition (not every reconcile while degraded), add a second condition:

```yaml
when:
  - field: status.phase
    equals: "Degraded"
  - field: status.alertSent
    operator: notExists

# Then write status.alertSent after a successful notify:
status:
  fields:
    - path: alertSent
      value: "true"
      when:
        - field: external.slackAlert.status
          equals: "200"
    - path: alertSent
      value: ""
      clearOnFalse: true
      when:
        - field: status.phase
          equals: "Degraded"
```

---

## Feature flags and runtime toggles

Fetch feature flag state from LaunchDarkly, Unleash, or a custom flags API. Gate which resources to create on the flag value. The `continueOnError: true` setting ensures the Deployment keeps running even if the flags service is unavailable.

```yaml
onReconcile:
  external:
    - name: featureFlags
      url: "https://flags.internal/api/{{ .metadata.name }}"
      token: "$FEATURE_FLAG_TOKEN"
      continueOnError: true
      timeout: 3s

  deployments:
    # v2 Deployment only created when the feature flag enables it.
    - name: "{{ .metadata.name }}-v2"
      image: "{{ .spec.imageV2 }}"
      when:
        - field: external.featureFlags.called
          equals: "true"
        - field: external.featureFlags.body
          contains: '"v2Enabled":true'
```

The `contains:` operator on `.body` checks for a JSON substring. This avoids any JSON parsing — the operator treats the body as opaque text.
