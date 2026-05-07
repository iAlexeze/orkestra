# Developer Pack — Local to Production in Minutes

This pack is for developers who want to run their application on Kubernetes
without writing operator code, managing Helm values, or configuring monitoring.

Three commands cover everything:

```
ork doctor          ← understand what Orkestra sees in your project
ork doctor init     ← generate .orkestra/ config
ork deploy          ← build, push, deploy, watch
```

---

## Examples

Work through them in order — each builds on the last.

| # | What you learn |
|---|---------------|
| [01 — One project](./01-one-project/README.md) | First deploy: examine, init, deploy, view in Control Center |
| [02 — Frontend + backend](./02-frontend-backend/README.md) | Multi-project deploy; internal service URLs |
| [03 — Rollback and Ingress](./03-rollback-ingress/README.md) | Instant rollback; public URL via `--add-ingress` |
| [04 — Notifications](./04-notify/README.md) | Get alerted on your phone/Slack when a deploy stalls |
| [05 — Deletion protection](./05-deletion-protection/README.md) | Prevent accidental deletes of your running app |

---

## The demo application

Every example deploys the same Go HTTP server from `app/`.
It serves `GET /` and `GET /health` and reads its name and port from `.env`.

```
app/
  main.go       simple HTTP server
  go.mod
  Dockerfile
  .env.example  copy to .env and fill in
```

Example 02 adds a minimal nginx frontend from `frontend/`.

---

## Dependencies

| Tool | Required | Notes |
|------|----------|-------|
| `ork` CLI | Yes | Install: `curl get.orkestra.sh \| bash` |
| Docker | Yes | Build and push images |
| A container registry | Yes | Pass via `--registry` (e.g. `ghcr.io/myorg`) |
| Go | Only for `--dev` | Needed to install `kind` — see note below |
| `kubectl` | Auto-installed | `ork deploy` installs it if missing |
| `helm` | Auto-installed | `ork deploy` installs it if missing |

!!! note "About Go and --dev"
    If you don't have a Kubernetes cluster, `ork deploy --dev` creates a local
    `kind` cluster called `orkestra-playground`. `ork deploy` installs `kind`
    automatically — but `kind` requires Go. Install Go from https://go.dev/dl/
    before using `--dev`. Everything else (`kubectl`, `helm`, `kind`) is
    handled for you.

---

## Cluster setup (same in every example)

Every example starts with the same choice:

**Option A — Use an existing cluster**

Check that your current context points to the right cluster:

```bash
kubectl config current-context
```

If it does, continue. `ork deploy` will deploy to that cluster.

**Option B — Let Orkestra create a local cluster (--dev)**

```bash
# Requires Go — see Dependencies above
ork deploy --registry <registry> --dev
```

`ork deploy --dev` creates a `kind` cluster named `orkestra-playground` on the
first run and reuses it on subsequent runs. Kind and all cluster tooling are
installed automatically.

---

## Control Center

After every successful deploy, Orkestra prints a Control Center link.
If no external hostname is configured, access it locally:

```bash
kubectl port-forward svc/orkestra-cc -n orkestra-system 8081:8081 &
```

Open [http://localhost:8081](http://localhost:8081).

In every example you will be directed to the Control Center to observe the
deployment in real time — without typing `kubectl` commands.

---

**Start here →** [01 — One project](./01-one-project/README.md)
