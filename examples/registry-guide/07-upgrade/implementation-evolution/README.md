# Implementation Evolution

The `web-service` motif gains two new inputs in v1.1.0: `probeProfile` and `probePath`. The API team upgrades. The platform team does not. Both continue running in the same runtime under separate namespace panels.

No CRD API change. No migration. No coordination between teams.

---

## Step 1 — Publish the new motif

The motif author publishes `web-service:v1.1.0` with backwards-compatible defaults.

→ [01-motif-v1.1.0/](01-motif-v1.1.0/README.md)

---

## Step 2 — API team upgrades

The API team bumps their katalog to follow the motif. They publish `webapp-operator:v1.1.0` and their CR explicitly sets `probePath: /ready` to use the new input.

→ [02-api-team-upgrades/](02-api-team-upgrades/README.md)

---

## Step 3 — Platform team stays

The platform team's `platform-app-operator:v1.0.0` still imports `web-service:v1.0.0`. Their CRs have no `probePath` field. The motif default `/health` applies on every reconcile. Nothing breaks. Nothing changes. This is the correct state — they stay on their schedule.

`platform-app-operator:v1.0.0` is already in the registry from [04-katalog-platform](../../04-katalog-platform/README.md#publish).

---

## Compose and deploy

Both katalogs are in the registry. The komposer imports them. Each published katalog carries its own `metadata.namespace` — the runtime treats them as independent tenant scopes.

```yaml
# komposer.yaml
imports:
  registry:
    - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.1.0        # api-team
    - oci://ghcr.io/myorg/katalogs/platform-app-operator:v1.0.0  # platform-team
```

> Update `myorg` to the actual path.

Pull the imports:

```bash
ork pull -f komposer.yaml
```

> Run this in the `registry-guide/07-upgrade/implementation-evolution` directory.

---


Apply CRDs for both teams:

```bash
kubectl apply -f 02-api-team-upgrades/crd.yaml
kubectl apply -f ../../04-katalog-platform/crd.yaml
```

Generate the bundle and deploy:

```bash
ork generate bundle -f komposer.yaml -o bundle.yaml
kubectl apply -f bundle.yaml

helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set runtime.configMap=implementation-upgrade \
  --wait --timeout 120s
```

---

## Verify

Apply CRs for both teams:

```bash
kubectl apply -f 02-api-team-upgrades/cr.yaml
kubectl apply -f ../../04-katalog-platform/cr.yaml
```

Check both CRD types:

```bash
kubectl get webapp,platformapp
# NAME                          AGE
# webapp.rkguide.demo/my-webapp       45s
# platformapp.rkguide.demo/my-platform-app   48s
```

Inspect the Deployments to see the version gap:

```bash
kubectl get deployments
# NAME               READY   UP-TO-DATE   AVAILABLE   AGE
# my-webapp          2/2     2            2           60s
# my-platform-app    2/2     2            2           63s

kubectl get deployment my-webapp -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.httpGet.path}' && echo
# /ready

kubectl get deployment my-platform-app -o jsonpath='{.spec.template.spec.containers[0].readinessProbe.httpGet.path}' && echo
# /health
```

Same cluster. Same runtime binary. Same motif underneath. Different probe paths — because the API team upgraded and the platform team did not.

---

## Control Center

```bash
ork proxy
# open http://localhost:8081
```

Two namespace panels:

- `api-team` — `webapp-operator:v1.1.0` — WebApp CRD — probes at `/ready`
- `platform-team` — `platform-app-operator:v1.0.0` — PlatformApp CRD — probes at `/health`

Each team sees their own operators. Neither sees the other's.

---

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

→ Next: [api-evolution](../api-evolution/README.md) — when the CRD field layout itself needs to change
