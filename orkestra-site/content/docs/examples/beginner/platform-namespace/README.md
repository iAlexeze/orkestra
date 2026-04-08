---
title: "README"
weight: 18
---

# Example 2 — Platform Namespace

A real‑world platform engineering pattern — fully declarative, no Go code.

Every platform team provisions namespaces for application teams. The workflow is always the same:

- Create a namespace  
- Copy shared secrets  
- Create a ConfigMap with environment metadata  
- Add a ServiceAccount for workloads  
- Keep everything in sync  
- Clean up when the namespace is no longer needed  

Most companies solve this with scripts, custom controllers, or manual steps.

With Orkestra, it becomes a **single Katalog**.

For every `PlatformNamespace` CR you apply, Orkestra automatically:

- Creates a ConfigMap with `ENVIRONMENT`, `LOG_LEVEL`, `TEAM`, and `NAMESPACE`  
- Copies the registry pull secret from the `platform` namespace  
- Creates a ServiceAccount for workload identity  
- Keeps everything in sync — updates propagate on reconcile  
- Cleans up automatically when the CR is deleted  

{{< callout type="note" >}}
This example demonstrates how Orkestra can replace entire provisioning pipelines with a single declarative operator.
{{< /callout >}}

---

## Requirements

{{< callout type="note" >}}
This example assumes you have completed **Getting Started** and understand how to:

- Apply CRDs  
- Run Orkestra with a Katalog  
- Reconcile CRs  
- Inspect generated resources
{{< /callout >}}

You will need:

- A Kubernetes cluster  
- A `platform` namespace containing your registry pull secret  
- `kubectl`  
- Orkestra CLI (`ork`)  

---

## Prerequisite: Create the Registry Secret

Before running this example, create the shared pull secret:

```bash
kubectl create namespace platform

kubectl create secret docker-registry registry-pull-secret \
  --namespace platform \
  --docker-server=YOUR_REGISTRY \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD
```

{{< callout type="caution" >}}
If you do **not** have a registry secret, remove the `secrets` block from `platform-namespace-katalog.yaml` before running this example.
{{< /callout >}}

---

## Files

```
platform-namespace/
  platform-namespace-crd.yaml      # CRD definition
  platform-namespace-cr.yaml       # Three sample PlatformNamespace CRs
  platform-namespace-katalog.yaml  # Orkestra Katalog
```

{{< callout type="tip" >}}
Keeping each example in its own directory makes it easier to iterate and debug.
{{< /callout >}}

---

## Run It

### Step 1 — Apply the CRD

```bash
kubectl apply -f platform-namespace-crd.yaml
```

---

### Step 2 — Start Orkestra

```bash
ork run --katalog platform-namespace-katalog.yaml
```

{{< callout type="note" >}}
Orkestra will wait for informer caches to sync and then start workers for the `PlatformNamespace` CRD.
{{< /callout >}}

---

### Step 3 — Apply Sample CRs

```bash
kubectl apply -f platform-namespace-cr.yaml
```

---

### Step 4 — Verify

List the CRs:

```bash
kubectl get platformnamespaces
# NAME                   TEAM       ENVIRONMENT   NAMESPACE
# payments-production    payments   production    payments-prod
# payments-staging       payments   staging       payments-staging
# platform-development   platform   development   platform-dev
```

Check the generated ConfigMap:

```bash
kubectl get configmap payments-production-config -n payments-prod -oyaml
# data:
#   ENVIRONMENT: production
#   LOG_LEVEL: warn
#   NAMESPACE: payments-prod
#   TEAM: payments
```

Verify the pull secret was copied:

```bash
kubectl get secret registry-pull-secret -n payments-prod
# NAME                    TYPE                             DATA
# registry-pull-secret    kubernetes.io/dockerconfigjson   1
```

Check the ServiceAccount:

```bash
kubectl get serviceaccount payments-sa -n payments-prod
# NAME          SECRETS   AGE
# payments-sa   0         10s
```

{{< callout type="tip" >}}
All generated resources include owner references — deleting the CR cleans up the namespace automatically.
{{< /callout >}}

---

## Test Drift Correction

Update the log level:

```bash
kubectl patch platformnamespace payments-staging \
  --type merge \
  -p '{"spec":{"logLevel":"debug"}}'
```

Verify the ConfigMap updates on reconcile:

```bash
kubectl get configmap payments-staging-config \
  -n payments-staging \
  -ojsonpath='{.data.LOG_LEVEL}'
# debug
```

{{< callout type="note" >}}
Drift correction applies to all resources with `reconcile: true` in the Katalog.
{{< /callout >}}

---

## Test Cascade Deletion

Delete a CR:

```bash
kubectl delete platformnamespace platform-development
```

Verify cleanup:

```bash
kubectl get configmap -n platform-dev
kubectl get secret -n platform-dev
kubectl get serviceaccount -n platform-dev
```

All generated resources are removed automatically.

---

## Health and Observability

Check health:

```bash
curl localhost:8080/katalog/platformnamespace/health | jq
```

List all reconciled CRs:

```bash
curl localhost:8080/katalog/platformnamespace | jq
```

Check metrics:

```bash
curl localhost:8080/metrics | grep platformnamespace
```

{{< callout type="tip" >}}
Metrics integrate seamlessly with Prometheus and Grafana for production observability.
{{< /callout >}}

---

## What’s Next

- **Example 3 — Komposer**  
  Compose this Katalog with others using file, Helm, and registry sources.
