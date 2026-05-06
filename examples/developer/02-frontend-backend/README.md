# Example 02 — Frontend + Backend

Deploy two projects to the same cluster. Orkestra tracks both in a single Komposer
and prints internal service URLs so you can wire them together without hard-coding
anything.

**What you learn:**

- Deploying multiple projects to the same cluster
- How Orkestra's Komposer merges katalogs from different projects
- Internal service URL discovery after deploy
- How the frontend proxies to the backend

!!! note
    Complete [Example 01](../01-one-project/README.md) first.
    `my-api` must already be deployed.

---

## The setup

```
my-api/        ← backend (from example 01, already deployed)
my-frontend/   ← nginx frontend (new in this example, from ../frontend/)
```

---

## Step 1 — Prepare the frontend

```bash
cp -r ../frontend my-frontend
cd my-frontend
```

Create `.env`:

```bash
cat > .env << 'EOF'
APP_NAME=my-frontend   # ork:cfg
PORT=3000              # ork:cfg
EOF
```

The frontend's `nginx.conf` already proxies `/api/` to
`my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local:8080`.
This is the internal DNS name Orkestra assigns to every backend service.

!!! tip
    You don't need to hardcode IP addresses. Orkestra creates a Service with a
    predictable DNS name: `<app>-orkestra-svc.<app>-orkestra-ns.svc.cluster.local`.

---

## Step 2 — Examine

```bash
ork doctor
```

```
Examining project...

  ✓ Dockerfile found
  ✓ Language: unknown   (no go.mod/package.json found — that's fine)
  ✓ Port: 3000
  ✓ .env — 2 variables
      2 config  (# ork:cfg)
      0 secrets (default)

Orkestra will create:
  Deployment     image built from Dockerfile, tagged :latest
  ...
  Ingress        my-frontend.local      (frontend detected)
```

!!! note
    Orkestra detects the frontend automatically because the `public/` or `dist/`
    directory was found, or because `package.json` references a known framework.
    For a plain nginx container, pass `--add-ingress` to `ork doctor init`
    to include an Ingress regardless.

---

## Step 3 — Init

```bash
ork doctor init --name my-frontend --add-ingress
```

!!! tip
    `--add-ingress` forces an Ingress into the katalog even when Orkestra
    doesn't auto-detect a frontend. Safe to use here — the nginx container
    serves a browser UI.

---

## Step 4 — Deploy the frontend

```bash
ork deploy --registry ghcr.io/myorg
# Add --dev if you used it in example 01
```

Watch the Komposer line — it now sees two projects:

```
  ✓ Komposer merged (2 projects)
```

After `Deployment ready`, the summary prints internal service URLs for every
project Orkestra knows about:

```
  Internal service URLs:
    my-api           http://my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local:8080
                     export MY_API_URL=http://my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local:8080

    my-frontend      http://my-frontend-orkestra-svc.my-frontend-orkestra-ns.svc.cluster.local:3000
                     export MY_FRONTEND_URL=http://...
```

Copy and run the `export` lines if you want to use them in other scripts.

---

## Step 5 — View both projects in the Control Center

1. Open [http://localhost:8081](http://localhost:8081)
2. The landing page now shows **two** Katalog cards: `my-api` and `my-frontend`
3. Click **View Resources** on `my-frontend`
4. Click the CR — observe:
   - **Children** section shows Deployment, Service, **and Ingress**
   - **Status** section shows `phase: Ready`

Switch to `my-api`:

5. Click back, open `my-api` → View Resources → click the CR
6. Notice the Ingress is absent — `my-api` is a backend, no Ingress was generated

---

## Step 6 — Test the connection

Port-forward the frontend and call the proxied backend:

```bash
kubectl port-forward svc/my-frontend-orkestra-svc -n my-frontend-orkestra-ns 3000:3000 &
curl http://localhost:3000/api/
# Hello from my-api
```

The request flows: `browser → nginx → internal DNS → my-api pod`.
No IP addresses, no hard-coded endpoints.

---

## Cleanup

```bash
cd ../..
chmod +x cleanup.sh && ./cleanup.sh 02
```

---

**← Previous** [01 — One project](../01-one-project/README.md) | **Next →** [03 — Rollback and Ingress](../03-rollback-ingress/README.md)
