# Example 1 — Website

The simplest Orkestra operator. No Go code. Three YAML files.

For every `Website` CR you apply, Orkestra automatically:
- Creates a `Deployment` with the image, replicas, and port from the CR spec
- Creates a `Service` of the declared type (ClusterIP, NodePort, or LoadBalancer)
- Sets owner references on both — deleted automatically when the CR is deleted
- Corrects drift — if someone manually changes the Deployment, Orkestra reconciles it back

---

## Requirements

- Kubernetes cluster (local or remote)
- `kubectl` configured and pointing at your cluster
- Orkestra CLI installed (`ork`)

---

## Files

```
website/
  website-crd.yaml      CRD definition — apply once per cluster
  website-cr.yaml       Three sample Website CRs
  website-katalog.yaml  Orkestra Katalog — tells Orkestra how to manage Websites
```

---

## Run it

**Step 1 — Apply the CRD**

```bash
kubectl apply -f website-crd.yaml
```

Verify:
```bash
kubectl get crds websites.demo.orkestra.io
```

**Step 2 — Start Orkestra**

```bash
ork run --katalog website-katalog.yaml
```

You should see:
```
katalogs merged  total=1 enabled=1
all informer caches synced
starting 2 workers for demo.orkestra.io/v1alpha1, Kind=Website
website workers started and ready
```

**Step 3 — Apply sample CRs**

In a second terminal:
```bash
kubectl apply -f website-cr.yaml
```

**Step 4 — Verify**

```bash
kubectl get websites
# NAME           AGE
# landing-page   5s
# my-api         5s
# my-blog        5s

kubectl get deployments
# NAME           READY   UP-TO-DATE   AVAILABLE
# landing-page   1/1     1            1
# my-api         3/3     3            3
# my-blog        2/2     2            2

kubectl get services
# NAME                TYPE           CLUSTER-IP
# landing-page-svc    ClusterIP      10.96.x.x
# my-api-svc          LoadBalancer   10.96.x.x
# my-blog-svc         ClusterIP      10.96.x.x
```

**Step 5 — Check health**

```bash
curl localhost:8080/katalog/website/health | jq
# {
#   "name": "website",
#   "healthy": true,
#   "totalReconciles": 3,
#   "errorRate": 0
# }
```

---

## Test drift correction

Change the replicas on a Deployment manually:
```bash
kubectl scale deployment my-blog --replicas=5
```

Watch Orkestra correct it back to 2 (the value in the CR spec) on the next
reconcile cycle.

---

## Test cascade deletion

Delete a Website CR and watch Orkestra clean up the Deployment and Service:
```bash
kubectl delete website landing-page

kubectl get deployments
# landing-page is gone — owner reference triggered garbage collection
```

---

## Validate the Katalog

```bash
ork validate --katalog website-katalog.yaml
# Success: Katalog is valid
```

Preview what Orkestra will manage:
```bash
ork template --katalog website-katalog.yaml --json
```

---

## What's next

- [Example 2 — Platform Namespace](../platform-namespace/README.md)
  Secrets, ConfigMaps, ServiceAccounts — the full platform provisioning pattern
- [Example 3 — Meta Katalog](../meta-katalog/README.md)
  Composing multiple Katalogs from files and Helm charts