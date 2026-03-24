# Example 2 — Platform Namespace

A real-world platform engineering pattern. No Go code.

Every platform team at every company provisions namespaces for their
application teams. The process is always the same — create the namespace,
copy secrets, create a ConfigMap with environment config, add a ServiceAccount.
Usually this is a script, a custom controller, or done manually.

With Orkestra it is a Katalog.

For every `PlatformNamespace` CR you apply, Orkestra automatically:
- Creates a ConfigMap with `ENVIRONMENT`, `LOG_LEVEL`, `TEAM`, and `NAMESPACE`
- Copies the registry pull secret from the `platform` namespace into the
  provisioned namespace — workloads can pull images immediately
- Creates a ServiceAccount for workload identity
- Keeps everything in sync — if `logLevel` changes in the CR, the ConfigMap
  updates on the next reconcile
- Cleans up — delete the CR and everything is removed automatically

---

## Requirements

- Kubernetes cluster with a `platform` namespace containing your pull secret
- `kubectl` configured and pointing at your cluster
- Orkestra CLI installed (`ork`)

**Before running this example, create the prerequisite secret:**

```bash
kubectl create namespace platform

kubectl create secret docker-registry registry-pull-secret \
  --namespace platform \
  --docker-server=YOUR_REGISTRY \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD
```

If you do not have a registry secret, remove the `secrets` block from
`platform-namespace-katalog.yaml` before running.

---

## Files

```
platform-namespace/
  platform-namespace-crd.yaml      CRD definition
  platform-namespace-cr.yaml       Three sample PlatformNamespace CRs
  platform-namespace-katalog.yaml  Orkestra Katalog
```

---

## Run it

**Step 1 — Apply the CRD**

```bash
kubectl apply -f platform-namespace-crd.yaml
```

**Step 2 — Start Orkestra**

```bash
ork run --katalog platform-namespace-katalog.yaml
```

**Step 3 — Apply sample CRs**

```bash
kubectl apply -f platform-namespace-cr.yaml
```

**Step 4 — Verify**

```bash
kubectl get platformnamespaces
# NAME                   TEAM       ENVIRONMENT   NAMESPACE
# payments-production    payments   production    payments-prod
# payments-staging       payments   staging       payments-staging
# platform-development   platform   development   platform-dev

# ConfigMap created in the provisioned namespace
kubectl get configmap payments-production-config -n payments-prod -oyaml
# data:
#   ENVIRONMENT: production
#   LOG_LEVEL: warn
#   NAMESPACE: payments-prod
#   TEAM: payments

# Pull secret copied
kubectl get secret registry-pull-secret -n payments-prod
# NAME                    TYPE                             DATA
# registry-pull-secret    kubernetes.io/dockerconfigjson   1

# ServiceAccount created
kubectl get serviceaccount payments-sa -n payments-prod
# NAME          SECRETS   AGE
# payments-sa   0         10s
```

---

## Test drift correction

Update the log level on a CR:
```bash
kubectl patch platformnamespace payments-staging \
  --type merge \
  -p '{"spec":{"logLevel":"debug"}}'
```

On the next reconcile, the ConfigMap in `payments-staging` updates:
```bash
kubectl get configmap payments-staging-config -n payments-staging -ojsonpath='{.data.LOG_LEVEL}'
# debug
```

---

## Test cascade deletion

```bash
kubectl delete platformnamespace platform-development

# All resources in platform-dev are cleaned up automatically:
kubectl get configmap -n platform-dev   # empty
kubectl get secret -n platform-dev      # empty
kubectl get serviceaccount -n platform-dev  # only default remains
```

---

## Health and observability

```bash
curl localhost:8080/katalog/platformnamespace/health | jq
curl localhost:8080/katalog/platformnamespace | jq
curl localhost:8080/metrics | grep platformnamespace
```

---

## What's next

- [Example 3 — Komposer](../komposer/README.md)
  Composing this Katalog with others using files and Helm sources