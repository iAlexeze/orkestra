# ork deploy

Build the Docker image, push it to a registry, generate the Orkestra cluster bundle,
install or verify the Orkestra operator, and roll out the new image.

```bash
ork deploy --registry <registry> [flags]
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-r, --registry` | — | **Required.** Container registry prefix (e.g. `ghcr.io/myorg`) |
| `-t, --tag` | git SHA | Image tag — defaults to the short Git commit SHA, then `latest` |
| `--name` | from `app.yaml` | App name override (normally read from `.orkestra/app.yaml`) |
| `--dry-run` | false | Show what would be applied without making changes |
| `--no-ha` | false | Skip HPA and PDB (single replica) |
| `--no-secure` | false | Skip deletion protection labels and webhook |
| `--clean` | false | Add `cleanupOnShutdown: true` to deletion protection |
| `--dev` | false | Create a local [kind](https://kind.sigs.k8s.io) cluster (`orkestra-playground`) |
| `-u, --upgrade-orkestra` | false | Upgrade the Orkestra operator before deploying |
| `--orkestra-version` | latest | Pin to a specific Orkestra operator version |
| `--values` | `.orkestra/values.yaml` | Path to Helm values for the Orkestra installation |

---

## Steps

```
 1. Detect        read .orkestra/app.yaml + .env
 2. Build         docker build -t <registry>/<app>:<tag> .
 3. Push          docker push <registry>/<app>:<tag>
 4. Kompose       ork kompose -k ~/.orkestra/deploy/komposer.yaml
                  → merges all registered katalogs into one runtime katalog
 5. Bundle        ork generate bundle -k <runtime-katalog> -o .orkestra/bundle/
 6. Apply         kubectl apply -f .orkestra/bundle/
 7. Notify secret kubectl apply orkestra-notification Secret (when SMTP/Slack in .env)
 8. Orkestra      helm install/upgrade orkestra (skipped if already installed)
 9. Health check  verify orkestra-runtime is Ready
10. Patch image   kubectl patch configmap <cr> -n <ns> --patch '{"data":{"image":"..."}}'
11. Watch         kubectl rollout status deployment/<cr> -n <ns> --timeout=5m
```

---

## Usage

### Basic deploy

```bash
ork deploy --registry ghcr.io/myorg
```

Builds `:a3f9c12` (the current git commit SHA), pushes it, and deploys.

### Pin a specific tag

```bash
ork deploy --registry ghcr.io/myorg --tag v1.4.0
```

### Dry run — see what would happen

```bash
ork deploy --registry ghcr.io/myorg --dry-run
```

Prints the image that would be built and the bundle that would be applied. Makes no
changes to the cluster.

### Development mode — local kind cluster

```bash
ork deploy --registry ghcr.io/myorg --dev
```

Creates (or reuses) a local `kind` cluster named `orkestra-playground`.
Requires [Go](https://go.dev/dl) installed (`kind` is installed by `ork deploy`).

### Upgrade the Orkestra operator

```bash
ork deploy --registry ghcr.io/myorg --upgrade-orkestra
```

Runs `helm upgrade` for Orkestra before deploying the workload. Use after an
Orkestra release to pick up new features.

---

## Auto-installed dependencies

`ork deploy` checks for `kubectl` and `helm` at the start of every run.
If either is missing it installs them automatically:

| Tool | Install method |
|------|---------------|
| `kubectl` | `curl .../dl.k8s.io/release/stable.txt` → `sudo mv kubectl /usr/local/bin/` |
| `helm` | `curl get-helm-3 \| bash` |

---

## Ingress controller

When the project has a frontend (`build/`, `dist/`, `public/`, or a React/Vue/Next
`package.json`), `ork deploy` checks for an ingress controller and installs one if
none is found:

- **kind cluster** — applies the [kind-specific nginx manifest](https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml)
- **other clusters** — installs `ingress-nginx` via Helm

---

## Notification secret

When `.env` contains `SMTP_*` or `SLACK_*` variables, `ork deploy` creates an
`orkestra-notification` Secret in `orkestra-system`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: orkestra-notification
  namespace: orkestra-system
type: Opaque
stringData:
  SMTP_HOST: "smtp.example.com"
  SMTP_USER: "..."
  SLACK_WEBHOOK_URL: "https://hooks.slack.com/..."
```

This Secret is mounted into the Orkestra runtime via `runtime.extraEnvFrom`
(configured in `.orkestra/values.yaml` when `ork doktor init --notify-me` was used),
making the credentials available as env vars to `pkg/konfig`.

---

## Multi-project awareness

Every project that deploys to the same cluster is registered in
`~/.orkestra/deploy/komposer.yaml` using its absolute local filesystem path.
Before generating the bundle, `ork deploy` runs `ork kompose` to merge all registered
katalogs into a single `__runtime_katalog_do_not_edit.yml`. Orkestra sees the merged
view — not the raw Komposer file.

Any katalog that cannot be read (permission error, bad YAML, missing file) will fail
`ork kompose` immediately, surfacing the problem before anything touches the cluster.

After a successful deploy, the ready summary prints internal service URLs for every
known project so you can wire them together:

```
  Internal service URLs:
    my-api                         http://my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local:8080
                                   export MY_API_URL=http://my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local:8080
    my-frontend                    http://my-frontend-orkestra-svc.my-frontend-orkestra-ns.svc.cluster.local:3000
```

---

## Deploy state

Each successful deploy is recorded in `~/.orkestra/deploy/state.json`:

```json
{
  "projects": {
    "my-api": {
      "name": "my-api",
      "namespace": "my-api-orkestra-ns",
      "currentImage": "ghcr.io/myorg/my-api:a3f9c12",
      "previousImage": "ghcr.io/myorg/my-api:d1e2f30"
    }
  }
}
```

`previousImage` is captured **before** the new image is patched, so rollback is
always instant — no rebuild required.

---

## Deployment readiness notifications

Every generated katalog ships with a `notify:` block on the `Deploying` condition.
When replicas are not ready after the first notification interval (default 15 minutes),
Orkestra sends the `developer` team:

```
my-api is deploying but replicas are not ready yet.
Check logs: kubectl logs -n my-api-orkestra-ns -l ork.io/app=my-api --tail=50
Roll back if stuck: ork deploy rollback
```

Wire the `developer` team by running `ork doktor init --name my-api --notify-me`.
Without it, the `notify:` block is present but fires to nobody — no error.

---

## Rollback

If a deploy fails or the rollout gets stuck, roll back instantly:

```bash
ork deploy rollback
```

See [ork deploy rollback](./rollback.md) for full details.

---

**← Previous** [ork doktor](./doktor.md) | **Next →** [ork deploy rollback](./rollback.md)
