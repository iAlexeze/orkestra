# 05 — When Conditions

One CRD. One Katalog. Three tiers. Different resources at each tier — without
a single line of conditional logic in Go.

This is the declarative topology pattern: the operator's behavior changes
continuously with the state of the CR.

**What you learn:** `when:` conditions, multi-condition gates, topology that
changes with CR state, the difference between conditions and admission rules.

**Builds on:** [04 — Multi-Resource with Status](../04-multi-resource/README.md)

---

## The pattern

`when:` conditions are evaluated per-resource, per-reconcile-cycle. When
conditions evaluate to false, the resource is simply not created — no error,
no warning, no event. The CR is healthy. The reconcile succeeds.

When the CR changes (e.g. `spec.tier` changes from `free` to `enterprise`),
the next reconcile creates all resources whose conditions now pass. Resources
whose conditions no longer pass are not re-created. Because `reconcile: true`
is set on all resources, this is handled automatically.

---

## Step 1 — Start with the free tier

```bash
kubectl apply -f crd.yaml
ork run --katalog katalog.yaml
kubectl apply -f cr-free.yaml
```

Check what was created:

```bash
kubectl get deployments,services,configmaps | grep my-platform
```

Expected — only the core resources, no LoadBalancer, no monitoring, no enterprise config:

```
deployment.apps/my-platform          1/1
service/my-platform-internal         ClusterIP
```

Check status:

```bash
kubectl get platform my-platform -o jsonpath='{.status.tier}'
# free
```

## Step 2 — Upgrade to enterprise

No need to delete the CR. Just patch it:

```bash
kubectl patch platform my-platform --type=merge \
  -p '{"spec":{"tier":"enterprise","replicas":4,"monitoring":true}}'
```

Wait one reconcile cycle, then check:

```bash
kubectl get deployments,services,configmaps | grep my-platform
```

Expected — all resources now exist:

```
deployment.apps/my-platform                   4/4
service/my-platform-internal                  ClusterIP
service/my-platform-lb                        LoadBalancer   ← appeared
configmap/my-platform-monitoring              2              ← appeared
configmap/my-platform-enterprise-config       3              ← appeared
```

The topology changed with the data. No operator code changed. No redeployment.

## Step 3 — Downgrade back to free

```bash
kubectl patch platform my-platform --type=merge \
  -p '{"spec":{"tier":"free","replicas":1,"monitoring":false}}'
```

Wait one reconcile cycle:

```bash
kubectl get services | grep my-platform
# my-platform-lb is gone — conditions no longer pass, not re-created
```

```bash
kubectl get configmaps | grep my-platform
# monitoring and enterprise configmaps gone
```

## Step 4 — Understand the status

The status shows the current tier and replica state at all times:

```bash
kubectl get platform my-platform -o yaml | grep -A15 "status:"
```

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
  tier: free          ← updates with the CR spec
  phase: Running
  readyReplicas: "1"  ← from the live Deployment
```

---

## The key insight

`when:` conditions answer: *given that this CR is valid, should this resource
exist right now?*

They are not admission rules — the CR is valid at every tier value. They are
not business logic — there is no Go code. They are declarations of when a
resource belongs in the world.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
