# pkg/doktor

The doktor package turns a project directory into a production-grade Kubernetes deployment — without the developer writing a single pod spec.

It reads what the developer already has (`Dockerfile`, `.env`, `.git`) and produces everything Kubernetes needs: a Katalog, an application ConfigMap, and a bundle of Secrets and ConfigMaps derived from the environment file.

## What the developer already has

```
my-app/
  Dockerfile        ← how to build
  .env              ← config and secrets
  .git/             ← what version to deploy
```

That is enough. `pkg/doktor` produces the rest.

## What the package produces

| Output | File | How |
|--------|------|-----|
| Orkestra Katalog | `.orkestra/katalog.yaml` | `Init()` → `buildKatalog()` |
| Application ConfigMap | `.orkestra/app.yaml` | `Init()` → `buildCR()` |
| Kubernetes Secret | `.orkestra/bundle/app-secrets.yaml` | `GenerateBundle()` |
| Kubernetes ConfigMap | `.orkestra/bundle/app-config.yaml` | `GenerateBundle()` |

## Package structure

| File | Responsibility |
|------|---------------|
| `detect.go` | Read the project directory — language, port, git commit, frontend |
| `envfile.go` | Parse `.env`, split into secrets and config vars |
| `generate.go` | Write `.orkestra/katalog.yaml` and `.orkestra/app.yaml`; `GenerateOptions.OutDir` enables per-app subdirectory output for multi-app projects |
| `bundle.go` | Write `.orkestra/bundle/app-secrets.yaml` and `app-config.yaml` |
| `compose.go` | Parse `docker-compose.yaml`; classify services as stateless (Deployments) or stateful (Motif-backed); extract build context and Dockerfile paths |
| `ingress.go` | Detect ingress controller and Orkestra presence on the cluster |
| `helm.go` | `helm install` / `helm upgrade` wrappers for the Orkestra chart |
| `kind.go` | Create and manage a local `orkestra-playground` kind cluster for `--dev` |
| `runtime.go` | `CheckRuntimeHealth`, `FetchRuntimeLogs`, `KatalogChanged`, `RestartOrkestra` |
| `state.go` | `~/.orkestra/deploy/state.json` — per-machine deploy history, rollback support |
| `komposer.go` | `~/.orkestra/deploy/komposer.yaml` — aggregate Katalog paths from all deployed projects |

Image building and pushing moved to [`pkg/buildx`](../buildx/README.md) — builder auto-detection (Docker / Podman / Buildah), Dockerfile selection, and compose-based builds.

## The `.env` contract

Variables in `.env` become Kubernetes resources — never baked into the image.

```bash
DATABASE_URL=postgres://user:pass@host/db   # → Secret
JWT_SECRET=abc123xyz                         # → Secret

PORT=8080       # ork:cfg → ConfigMap (not a secret)
LOG_LEVEL=info  # ork:cfg → ConfigMap
```

Variables tagged `# ork:cfg` on the same line become a ConfigMap. All others become a Secret.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand project detection | [docs/01-detection.md](docs/01-detection.md) |
| Understand `.env` parsing | [docs/02-envfile.md](docs/02-envfile.md) |
| Understand Katalog generation (incl. multi-app, compose, Role/RoleBinding) | [docs/03-generation.md](docs/03-generation.md) |
| Understand bundle generation | [docs/04-bundle.md](docs/04-bundle.md) |
| Understand cluster operations, health checks, deploy state | [docs/05-deploy.md](docs/05-deploy.md) |
| Understand image building and builder selection | [`pkg/buildx`](../buildx/README.md) |
