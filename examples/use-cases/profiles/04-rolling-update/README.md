# Profiles 04 — Rolling Update

One CR. Three Deployments. Each uses a different rollout strategy from a profile name — no `maxSurge` or `maxUnavailable` to configure.

**What you learn:** `rollingUpdate.profile`, what each strategy expands to, and how to trigger a rollout to observe the difference.

---

## Profiles at a glance

| Profile | maxSurge | maxUnavailable | Behaviour |
|---|---|---|---|
| `safe` | 1 | 0 | Zero capacity drop — adds one pod before removing one. Default for production. |
| `fast` | 25% | 25% | Removes and adds pods in parallel. Faster rollout with brief capacity reduction. |
| `blue-green` | 100% | 0 | Full duplicate capacity before old pods are removed. Most expensive, cleanest cutover. |

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

Open [http://localhost:8081](http://localhost:8081). Select **service-rolling-profiles**, then **Service**.

---

## Step 5 — Apply the CR

```bash
kubectl apply -f ../cr.yaml
```

Verify the rollout strategies:

```bash
kubectl get deployments -o custom-columns=\
'NAME:.metadata.name,MAX-SURGE:.spec.strategy.rollingUpdate.maxSurge,MAX-UNAVAIL:.spec.strategy.rollingUpdate.maxUnavailable'
```

Expected:
```
NAME                MAX-SURGE   MAX-UNAVAIL
my-service-safe     1           0
my-service-fast     25%         25%
my-service-bg       100%        0
```

---

## Step 6 — Trigger a rollout and observe the difference

Patch the image to start a rolling update across all three:

```bash
kubectl patch services.demo.orkestra.io my-service --type=merge -p '{"spec":{"image":"nginx:1.26"}}'
```

Watch the rollout in real time:

```bash
kubectl rollout status deployment/my-service-safe     # one pod at a time
kubectl rollout status deployment/my-service-fast     # parallel
kubectl rollout status deployment/my-service-bg       # full surge first
```

During the blue-green rollout, watch the pod count double temporarily and the start-time difference:

```bash
watch kubectl get pods -l orkestra-owner=my-service
```

---

## Using a profile in your own Katalog

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    replicas: "3"
    rollingUpdate:
      profile: safe   # maxSurge: 1, maxUnavailable: 0
```

---

## E2E

Run the full lifecycle in one command — applies the CR, asserts all three rolling update profile Deployments are created, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Three rolling update profile Deployments created
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-service-safe
        namespace: default
      - kind: Deployment
        name: my-service-fast
        namespace: default
      - kind: Deployment
        name: my-service-bg
        namespace: default
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
