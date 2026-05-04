# Example 01 — One Project

Deploy a single application from your local machine to Kubernetes with three commands.
No CRDs to write. No Helm charts. No `kubectl apply`.

**What you learn:**

- `ork doktor` — what Orkestra detects in your project
- `ork doktor init` — generating the `.orkestra/` configuration
- `ork deploy` — the full build → push → deploy → watch pipeline
- Viewing your live deployment in the Orkestra Control Center

---

## Before you start

Copy the demo app and set up your registry:

```bash
cp -r ../app my-api
cd my-api
cp .env.example .env
```

Open `.env` and fill in your values (the defaults work for this example):

```bash
APP_NAME=my-api   # ork:cfg
PORT=8080         # ork:cfg
```

Lines tagged `# ork:cfg` become a Kubernetes ConfigMap (safe to log).
All other lines become a Kubernetes Secret (encrypted at rest).

---

## Step 1 — Connect to a cluster

**Option A — existing cluster**

```bash
kubectl config current-context
# Make sure this is the cluster you want to deploy to.
```

**Option B — let Orkestra create one (requires Go)**

You don't need a cluster yet. Jump to Step 4 and add `--dev`:

```bash
ork deploy --registry <registry> --dev
```

Orkestra creates a local `kind` cluster called `orkestra-playground` and deploys
into it. `kubectl`, `helm`, and `kind` are installed automatically.

---

## Step 2 — Examine the project

```bash
ork doktor
```

Example output:

```
Examining project...

  ✓ Dockerfile found
  ✓ Git repository — commit: a3f9c12
  ✓ Language: Go  (go.mod)
  ✓ Port: 8080
  ✓ .env — 2 variables
      2 config  (# ork:cfg)
      0 secrets (default)

Orkestra will create:
  Deployment     image built from Dockerfile, tagged :a3f9c12
  ConfigMap      my-api-config  (2 variables from .env # ork:cfg)
  Service        port 8080
  HPA            min 2 / max 10
  PDB            minAvailable: 1
  DeletionProtection  enabled

Missing dependencies:
  (none)

Run 'ork doktor init' to generate .orkestra/katalog.yaml
```

!!! note
    No Dockerfile? Add one before continuing. `ork doktor` must find it.

---

## Step 3 — Generate the configuration

```bash
ork doktor init --name my-api
```

This creates three files:

```
.orkestra/
  katalog.yaml    ← what Kubernetes resources Orkestra manages — edit freely
  app.yaml        ← the ConfigMap CR you own (set replicas, host, etc.)
  values.yaml     ← Helm values for the Orkestra operator
```

Open `.orkestra/app.yaml` and verify the port:

```yaml
data:
  port: "8080"
  replicas: "2"
  controlCenterHost: ""   # leave empty for local access
  image: ""               # set automatically by ork deploy
```

!!! tip
    `.orkestra/bundle/` is added to `.gitignore` automatically.
    The three files above should be committed.

---

## Step 4 — Deploy

```bash
ork deploy --registry ghcr.io/myorg

# Or, if you don't have a cluster yet:
ork deploy --registry ghcr.io/myorg --dev
```

!!! note
    Replace `ghcr.io/myorg` with your actual registry.
    The image name will be `<registry>/my-api:<git-sha>`.

Watch the steps print:

```
Cluster:  kind-orkestra-playground

Building my-api...
  → ghcr.io/myorg/my-api:a3f9c12
  ✓ Built (12s)
  ✓ Pushed

Generating bundle...
  ✓ Komposer merged (1 projects)
  ✓ RBAC + Katalog ConfigMap + namespace

Applying to cluster...
  ✓ Bundle applied
  ✓ Orkestra installed
  Checking runtime health... ✓

Waiting for deployment...
  ✓ Deployment ready

  App:    (no ingress — set host in app.yaml to expose externally)
  Status: Ready
  Image:  ghcr.io/myorg/my-api:a3f9c12

  Control Center → http://localhost:8081
                   kubectl port-forward svc/orkestra-cc 8081:8081 -n orkestra-system &
```

---

## Step 5 — View in the Control Center

Open the Control Center:

```bash
kubectl port-forward svc/orkestra-cc -n orkestra-system 8081:8081 &
```

Visit [http://localhost:8081](http://localhost:8081).

**Navigate to your deployment:**

1. On the landing page, find the **my-api** Katalog card
2. Click **View Resources** (top right of the card) — lists all CR instances
3. Click **my-api-orkestra** — opens the CR detail page

**What you see on the detail page:**

| Section | What it shows |
|---------|--------------|
| Status | `phase: Ready` — your deployment is live |
| Data | `port`, `replicas`, `image` from `app.yaml` |
| Children | Deployment, Service, HPA, PDB — all created by Orkestra |
| Events | Full reconcile history — resource created, image patched, rollout completed |

!!! tip
    The detail page auto-refreshes every 10 seconds.
    Click any child resource to see its own status and events.

---

## Step 6 — Verify the app is running

```bash
kubectl port-forward svc/my-api-orkestra-svc -n my-api-orkestra-ns 8080:8080 &
curl http://localhost:8080/
# Hello from my-api

curl http://localhost:8080/health
# ok
```

---

## Cleanup

```bash
cd ..
chmod +x cleanup.sh && ./cleanup.sh 01
```

---

**← Pack overview** [README](../README.md) | **Next →** [02 — Frontend + backend](../02-frontend-backend/README.md)
