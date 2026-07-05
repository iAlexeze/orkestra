# Profiles 03 — Probes

One CR. Four Deployments. Each gets different probe timing from a profile name — no `initialDelaySeconds`, `periodSeconds`, or `failureThreshold` to configure manually.

**What you learn:** `probes.liveness.profile`, `probes.readiness.profile`, `probes.startup.profile`, what each timing preset expands to, and when `slow-start` via a startup probe is the right choice.

---

## Profiles at a glance

| Profile | InitialDelay | Period | FailureThreshold | Timeout | Use for |
|---|---|---|---|---|---|
| `fast` | 5s | 10s | 2 | 5s | HTTP APIs that start in seconds |
| `standard` | 15s | 20s | 3 | 10s | Most web services |
| `patient` | 30s | 30s | 5 | 10s | Batch workers, slow-starting services |
| `slow-start` | 0s | 10s | 30 | 10s | JVM apps, databases (5-min startup window) |

`slow-start` is designed for startup probes — 30 failures × 10s period = 5-minute window before Kubernetes gives up.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Simulate

```bash
ork simulate
```

---

## Step 3 — Start the runtime

```bash
ork run
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **service-probe-profiles**, then **Service**.

---

## Step 5 — Apply the CR

```bash
kubectl apply -f ../cr.yaml
```

Verify the probe timings:

```bash
kubectl get deployment my-service-fast \
  -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}' | jq
```

Expected for `fast`:
```json
{
  "httpGet": {"path": "/healthz", "port": 8080},
  "initialDelaySeconds": 5,
  "periodSeconds": 10,
  "failureThreshold": 2,
  "timeoutSeconds": 5
}
```

```bash
kubectl get deployment my-service-slow-start \
  -o jsonpath='{.spec.template.spec.containers[0].startupProbe}' | jq
```

Expected for `slow-start`:
```json
{
  "httpGet": {"path": "/healthz", "port": 8080},
  "initialDelaySeconds": 0,
  "periodSeconds": 10,
  "failureThreshold": 30,
  "timeoutSeconds": 10
}
```

---

## Using a profile in your own Katalog

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    port: "8080"
    probes:
      liveness:
        type: http
        path: /healthz
        profile: standard
      readiness:
        type: http
        path: /readyz
        profile: fast
```

For JVM apps or databases, use a startup probe to get the 5-minute window:

```yaml
    probes:
      startup:
        type: http
        path: /healthz
        profile: slow-start   # 30 × 10s = 5 minutes
      liveness:
        type: http
        path: /healthz
        profile: standard     # takes over after startup succeeds
```

---

## E2E

Run the full lifecycle in one command — applies the CR, asserts all four probe profile Deployments are created with the correct liveness probe timing, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Four probe profile Deployments created
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-service-fast
        namespace: default
      - kind: Deployment
        name: my-service-slow-start
        namespace: default

  - name: Fast profile has short probe period
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get deployment my-service-fast -o jsonpath='{.spec.template.spec.containers[0].livenessProbe.periodSeconds}'
        outputContains: "5"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
