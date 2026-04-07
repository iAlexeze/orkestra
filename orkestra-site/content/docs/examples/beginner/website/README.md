---
title: "README"
weight: 16
---

# Example 1 — Website

A simple, fully declarative Orkestra operator.  
No Go code.  
Just three YAML files.

For every `Website` CR you apply, Orkestra automatically:

- Creates a `Deployment` using the image, replicas, and port from the CR  
- Creates a `Service` of the declared type (`ClusterIP`, `NodePort`, or `LoadBalancer`)  
- Sets owner references so resources are deleted automatically  
- Corrects drift — if someone edits the Deployment manually, Orkestra restores it  

{{< callout type="note" >}}
This is the simplest real‑world operator you can build with Orkestra.  
It demonstrates templating, drift correction, and reconciliation — all declaratively.
{{< /callout >}}

---

## Requirements

{{< callout type="note" >}}
This example assumes you have completed **[Getting Started](../../../getting-started/index.md)** and already know how to:

- Install Orkestra  
- Write a basic Katalog  
- Write a basic Komposer  
- Run Orkestra locally  
- Reconcile a simple CR
{{< /callout >}}

You will need:

- A Kubernetes cluster (local or remote)  
- `kubectl` configured  
- Orkestra CLI installed (`ork`)  

---

## Files

```
website/
  website-crd.yaml      # CRD definition — applied once per cluster
  website-cr.yaml       # Three sample Website CRs
  website-katalog.yaml  # Orkestra Katalog — defines how Websites are managed
```

{{< callout type="tip" >}}
Keep each example in its own folder.  
It makes it easier to experiment, iterate, and debug.
{{< /callout >}}

---

## Run It

### Step 1 — Apply the CRD

```bash
kubectl apply -f website-crd.yaml
```

Verify:

```bash
kubectl get crds websites.demo.orkestra.io
```

{{< callout type="note" >}}
CRDs must be applied **before** Orkestra starts.  
If the CRD is missing, Orkestra will wait until it appears.
{{< /callout >}}

---

### Step 2 — Start Orkestra

```bash
ork run --katalog website-katalog.yaml
```

Expected output:

```
katalogs merged  total=1 enabled=1
all informer caches synced
starting 2 workers for demo.orkestra.io/v1alpha1, Kind=Website
website workers started and ready
```

{{< callout type="tip" >}}
If you see “waiting for informer caches,” ensure the CRD is applied and available.
{{< /callout >}}

---

### Step 3 — Apply Sample CRs

In a second terminal:

```bash
kubectl apply -f website-cr.yaml
```

---

### Step 4 — Verify

```bash
kubectl get websites
# NAME           AGE
# landing-page   5s
# my-api         5s
# my-blog        5s
```

Check generated Deployments:

```bash
kubectl get deployments
# NAME           READY   UP-TO-DATE   AVAILABLE
# landing-page   1/1     1            1
# my-api         3/3     3            3
# my-blog        2/2     2            2
```

Check Services:

```bash
kubectl get services
# NAME                TYPE           CLUSTER-IP
# landing-page-svc    ClusterIP      10.96.x.x
# my-api-svc          LoadBalancer   10.96.x.x
# my-blog-svc         ClusterIP      10.96.x.x
```

{{< callout type="note" >}}
All resources include owner references.  
Deleting the CR deletes the Deployment and Service automatically.
{{< /callout >}}

---

### Step 5 — Check Health

```bash
curl localhost:8080/katalog/website/health | jq
```

Example output:

```json
{
  "name": "website",
  "healthy": true,
  "totalReconciles": 3,
  "errorRate": 0
}
```

{{< callout type="tip" >}}
The health endpoint is invaluable during development.  
It shows reconcile counts, errors, and worker status.
{{< /callout >}}

---

## Test Drift Correction

Manually change a Deployment:

```bash
kubectl scale deployment my-blog --replicas=5
```

On the next reconcile cycle, Orkestra restores it to the value in the CR:

```
spec.replicas: 2
```

{{< callout type="caution" >}}
Drift correction only applies to resources with `reconcile: true` in the Katalog.
{{< /callout >}}

---

## Test Cascade Deletion

Delete a Website CR:

```bash
kubectl delete website landing-page
```

Verify cleanup:

```bash
kubectl get deployments
# landing-page is gone
```

{{< callout type="note" >}}
Owner references ensure Kubernetes garbage collection removes all generated resources.
{{< /callout >}}

---

## Validate the Katalog

```bash
ork validate --katalog website-katalog.yaml
# Success: Katalog is valid
```

Preview what Orkestra will generate:

```bash
ork template --katalog website-katalog.yaml --json
```

{{< callout type="tip" >}}
`ork template` is perfect for debugging templates without applying anything to the cluster.
{{< /callout >}}

---

## What’s Next

- **Example 2 — Platform Namespace**  
  Provision Secrets, ConfigMaps, ServiceAccounts — the full platform bootstrap pattern.

- **Example 3 — Komposer**  
  Compose multiple katalogs from files, Helm charts, and registries.
