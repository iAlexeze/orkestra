# Migration output

```bash
grep -rn "TODO(ork migrate)" .
```

Work through each TODO in order:

1. Add the Orkestra imports flagged at the top of the reconciler file
2. Replace `r.Status().Update` with `r.kube.PatchStatus`
3. Remove `SetupWithManager` and `main.go` — Orkestra provides the informer and workqueue

---

## Step 1 — Generate the type registry

```bash
make registry
```

---

## Step 2 — Build

```bash
make build
```

---

## Step 3 — Validate

```bash
make validate
```

---

## Step 4 — Simulate

```bash
ork simulate
```

---

## Step 5 — Release

Build the production runtime binary, package it into a Docker image, and push:

```bash
make release IMAGE=ghcr.io/myorg/my-operator:v0.1.0
```

---

## Step 6 — Push the Katalog

```bash
ork push .
```

---

## Inspect

```bash
ork inspect my-operator:v0.1.0
```

---

## Step 7 — Generate bundle and deploy

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

Install Orkestra with your custom runtime image:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --set runtime.image.repository=ghcr.io/myorg/my-operator \
  --set runtime.image.tag=v0.1.0 \
  --namespace orkestra-system --create-namespace \
  --wait --timeout 120s
```
