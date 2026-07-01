# Any Language – Generate an Orkestra Katalog from Python, Go, or Node.js

Orkestra only cares about the final Katalog YAML. You can generate it using any programming language.

This directory contains three language examples that produce identical `katalog.yaml` files, which can then be used to deploy a simple web application (Deployment + Service + optional Ingress) on Kubernetes.

## Prerequisites

- Orkestra CLI installed (`ork`)
- `kubectl` configured for a cluster (or use `ork doctor deploy --dev` to create a Kind cluster)
- For each language:
  - **Python**: `python3` and `pyyaml` (`pip install pyyaml`)
  - **Go**: `go` and `gopkg.in/yaml.v3` (fetched automatically)
  - **Node.js**: `node` and `npm`; install `js-yaml` locally (`npm install`)

## Shared files

- `crd.yaml` – CustomResourceDefinition for a `WebApp`
- `cr.yaml` – Example custom resource (`my-webapp`)

These are the same for all three examples.

## Step 1 – Apply the CRD

```bash
kubectl apply -f crd.yaml
```

(You only need to do this once; the CRD is shared.)

## Step 2 – Generate Katalogs using different languages (each to a separate file)

### Python

```bash
python generate_katalog.py --name my-webapp --image nginx:1.25 --port 8080 --replicas 3 --ingress myapp.example.com > python-katalog.yaml
```

### Go

```bash
go run generate_katalog.go -name my-webapp -image nginx:1.25 -port 8080 -replicas 3 -ingress myapp.example.com > go-katalog.yaml
```

### Node.js

```bash
npm install   # installs js-yaml
node generate_katalog.js --name my-webapp --image nginx:1.25 --port 8080 --replicas 3 --ingress myapp.example.com > node-katalog.yaml
```

## Step 3 – Validate each generated Katalog

```bash
ork validate -f python-katalog.yaml
ork validate -f go-katalog.yaml
ork validate -f node-katalog.yaml
```

## Step 4 – Run the operator with the first Katalog (e.g., Python)

```bash
ork run -f python-katalog.yaml
```

(Keep this terminal open.)

## Step 5 – Apply the custom resource (in another terminal)

```bash
kubectl apply -f cr.yaml
```

Watch the operator logs – you should see it creating a Deployment, Service, and optionally an Ingress.

## Step 6 – Verify resources

```bash
kubectl get deployments,services,ingresses -n default
```

- Deployment name: `my-webapp-deploy`
- Service name: `my-webapp-svc`
- Ingress name: `my-webapp-ingress`

## Step 7 – Stop the Python operator (Ctrl+C) and run the Go operator

```bash
# In the first terminal, stop the Python operator with Ctrl+C
ork run -f go-katalog.yaml
```

The operator will continue managing the existing `my-webapp` custom resource – no need to reapply the CR.

## Step 8 – Try Node.js the same way

Stop the Go operator and start the Node.js one:

```bash
ork run -f node-katalog.yaml
```

All three produce the same behaviour because they all create the same Katalog structure.

## Cleanup

```bash
kubectl delete -f cr.yaml
kubectl delete -f crd.yaml
# Stop the runtime with Ctrl+C
```

## Why this matters

- **No lock‑in** – Write your operator specification in any language you prefer.
- **Perfect for CI/CD** – Generate the Katalog dynamically based on environment variables or external APIs.
- **Learn by analogy** – Use the script that matches your team’s primary language.
- **Seamless switching** – Orkestra doesn't care which language generated the Katalog; the runtime works identically.
