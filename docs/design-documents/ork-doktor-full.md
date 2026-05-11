# ork doctor — Full Design Document

*Orkestra Project — April 2026*

---

## What this document covers

This is the complete design for the developer-facing layer of Orkestra. It
covers every planned feature from the rename through the stateful service
layer, with honest phasing based on what can ship clean versus what needs its
own release.

- Part 1: Core `ork doctor` expansions (v1.1)
- Part 2: `ork doctor deploy --expose` — public URL via tunnel daemon (v1.1)
- Part 3: Cleanup — `cleanupOnShutdown` and `ork cleanup` (v1.1)
- Part 4: RBAC scoped to the application namespace (v1.1)
- Part 5: `--use-compose` — docker-compose as input (v1.2)
- Part 6: Stateful services — databases, queues, admin UIs (v1.3)
- Part 7: Developer example pack (v1.1)
- Part 8: Complete developer flow

---

## Part 1: Core ork doctor expansions

### 1.1 Rename: doctor → doctor

All references to `ork doctor` become `ork doctor`. The k-theme is consistent
across Orkestra: Kubernetes, Katalog, Komposer, Kordinator. The old spelling
is kept as a hidden alias through v1 for backward compatibility.

```bash
ork doctor              # replaces: ork doctor
ork doctor init         # replaces: ork doctor init
ork doctor init --name myproject --notify-me
```

### 1.2 Git metadata and license extraction

`ork doctor init` reads project metadata before generating the Katalog.
The developer contributes nothing extra — the information is already in the
project.

**Author** — `git config user.email` in the project directory.

**License** — the LICENSE file in the project root. Detected by filename
(`LICENSE`, `LICENSE.md`, `LICENSE.txt`). The identifier is extracted from
the first line: `MIT License` → `MIT`, `Apache License 2.0` → `Apache-2.0`.
Omitted when no LICENSE file is found.

```yaml
metadata:
  name: my-app
  description: Orkestra HA deployment for my-app
  author: developer@example.com
  license: MIT
```

The same `user.email` is used as the default email recipient when
`--notify-me` is specified.

### 1.3 SMTP/Slack detection and --notify-me

`ork doctor` scans `.env` for notification credentials. When found, it
suggests the `--notify-me` flag at the end of its output — shown only when
the credentials are actually present, never as a generic suggestion.

**Detected patterns:**

| Variables | Channel |
|---|---|
| `SMTP_HOST` + `SMTP_USER` + `SMTP_PASS` | Email |
| `SLACK_WEBHOOK_URL` or `SLACK_BOT_TOKEN` | Slack |

**Detection output:**
```
  ✓ Slack credentials found (SLACK_WEBHOOK_URL)

💡 Run 'ork doctor init --notify-me' to get deployment alerts in Slack
```

**What `--notify-me` adds to the Katalog:**

A `notification:` block using the discovered credentials. Credentials are
referenced via `env.*` template expressions — they stay in the cluster Secret
generated from `.env`, not hardcoded in the Katalog.

```yaml
notification:
  enabled: true
  defaults:
    interval: 15m
    slackWebhookUrl: "{{ env.SLACK_WEBHOOK_URL }}"
  teams:
    developer:
      email: ["developer@example.com"]    # from git config user.email
      slack: ["#deployments"]
      interval: 5m
      message: >
        {{ .metadata.name }}: {{ conditionMessage .children.deployment "Available" }}
```

**Notification conditions** added to the operatorBox:

```yaml
onReconcile:
  when:
    - field: metrics.errorRatePercent
      greaterThan: "5"
      notify: [developer]

    - field: "{{ replicasReady .children.deployment }}"
      equals: "false"
      notify: [developer]
```

The developer receives alerts when their deployment is failing or pods are not
ready. The 15-minute interval prevents spam when an issue persists.

### 1.4 Dependency auto-install

`ork doctor` and `ork doctor deploy` check for required tools and install missing
ones automatically. Every installation is a static binary download to
`~/.orkestra/bin/` — no package manager, no sudo required. The deploy
commands prepend `~/.orkestra/bin/` to their child process PATH.

**Tool matrix:**

| Tool | Required for | Method |
|---|---|---|
| `kubectl` | All deployments | Binary download |
| `helm` | All deployments | Binary download |
| `kind` | `--dev` flag | Binary download |
| `metrics-server` | HPA | `helm install` in cluster |
| `ingress-nginx` | `host:` set in app.yaml | `helm install` in cluster |
| `cloudflared` | `--expose` | Binary download |
| `docker` | Build + push | Not auto-installed |

**`docker` is not auto-installed.** Docker Desktop has licensing implications
and daemon configuration that cannot be automated safely. If docker is not
found, `ork doctor` exits clearly:
```
  ✗ Docker not found
    Install Docker Desktop: https://docs.docker.com/get-docker/
```

**`kind` does not require Go.** It is a static binary downloaded from
`github.com/kubernetes-sigs/kind/releases/` directly. Previous designs
required `go install kind` — this is replaced. Most developers wanting a
local cluster do not have Go installed.

**`metrics-server`** is installed when HPA is in the Katalog (default unless
`--no-ha`). Detected via `kubectl api-resources --api-group=metrics.k8s.io`.

**`ingress-nginx`** is installed when `host:` is set in `app.yaml`. For kind
clusters, uses the kind-specific manifest. For all other clusters, uses the
standard Helm chart.

**Installation output during `ork doctor deploy`:**
```
Checking dependencies...
  ✓ docker 27.1.0
  → kubectl not found — installing...
  ✓ kubectl v1.31.0  (~/.orkestra/bin/kubectl)
  ✓ helm v3.16.0
  → metrics-server not found — installing...
  ✓ Metrics server ready
```

### 1.5 app.yaml replaces cr.yaml

The output configuration file is now `app.yaml`. `cr.yaml` is a Kubernetes
term the developer audience doesn't need to know. `app.yaml` is honest:
it is the application's configuration.

Comments are the documentation. Every field has a comment explaining what it
controls in plain language, not Kubernetes vocabulary.

```yaml
# app.yaml — your application configuration
# Apply once: kubectl apply -f .orkestra/app.yaml
# ork doctor deploy updates the image field automatically.

apiVersion: v1
kind: ConfigMap
metadata:
  name: my-app
  namespace: my-app-ns
  labels:
    ork.io/app: my-app
data:
  # How many copies of your app to run (minimum 2 for HA)
  replicas: "2"

  # The port your app listens on
  port: "8080"

  # Autoscaling — how many copies can Kubernetes spin up under load?
  minReplicas: "2"
  maxReplicas: "10"

  # Your public hostname — set this if you want a real URL
  # Example: myapp.example.com
  host: ""

  # Set by ork doctor deploy — do not edit manually
  image: ""
```

### 1.6 Scoped RBAC for generated deployments

Every deployment generated by `ork doctor init` gets a ServiceAccount, Role,
and RoleBinding scoped to its namespace. The Role grants only what the
deployment needs — nothing cluster-wide, nothing outside its namespace.

Generated RBAC in `.orkestra/bundle/rbac-app.yaml`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app
  namespace: my-app-ns
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: my-app
  namespace: my-app-ns
rules:
  # Read its own ConfigMap (app.yaml)
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: ["my-app"]
    verbs: ["get", "watch"]
  # Read its own Secret
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["my-app-secrets"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: my-app
  namespace: my-app-ns
subjects:
  - kind: ServiceAccount
    name: my-app
    namespace: my-app-ns
roleRef:
  kind: Role
  name: my-app
  apiGroup: rbac.authorization.k8s.io
```

This is separate from the Orkestra operator's own RBAC (ClusterRole, etc.).
The application's service account can read only its own ConfigMap and Secret —
nothing more, nothing outside its namespace.

### 1.7 ork doctor deploy --dev — kind cluster

```bash
ork doctor deploy --dev --registry ghcr.io/myorg
```

Lifecycle:
1. Check for `kind` in PATH or `~/.orkestra/bin/kind`; download if missing
2. Check if `kind-orkestra-dev` cluster exists; create if not
3. Set kubectl context to `kind-orkestra-dev`
4. Continue with normal deploy

```
  → Creating local cluster (kind)...
  ✓ Cluster ready: kind-orkestra-dev
```

The cluster is created once. All subsequent `ork doctor deploy --dev` calls detect
the existing cluster and skip creation. Multiple projects deploy to the same
kind cluster via the global Komposer.

---

## Part 2: ork doctor deploy --expose

### 2.1 Goal

After `ork doctor deploy`, the application runs in the cluster. On a local kind
cluster it is only accessible on localhost. `--expose` starts a background
tunnel daemon that creates a public HTTPS URL. The URL survives the deploy
command's exit and persists until explicitly stopped.

```bash
ork doctor deploy --dev --expose
#   ✓ App live at https://abc123.trycloudflare.com
#   Tunnel running — ork tunnel stop to end
```

### 2.2 Provider selection

Two providers, automatic selection based on what is available:

| Provider | Account | Free tier | Default |
|---|---|---|---|
| `cloudflared` | None | Unlimited (trycloudflare.com) | ✓ |
| `ngrok` | Required (free) | 40 req/min | Fallback |

Cloudflared is the default — no account, no token, anonymous HTTPS URL in
three seconds. ngrok is the fallback when cloudflared is not available or
explicitly requested.

```bash
ork doctor deploy --expose                             # cloudflared by default
ork doctor deploy --expose --tunnel-provider ngrok    # explicit
ork doctor deploy --expose --tunnel-token $NGROK_TOKEN # non-interactive CI
```

### 2.3 Tunnel as a background daemon

The tunnel runs detached — it outlives the `ork doctor deploy` command.

**State file: `~/.orkestra/tunnel-state.json`**
```json
{
  "provider": "cloudflared",
  "pid": 12345,
  "url": "https://abc123.trycloudflare.com",
  "localPort": 80,
  "startedAt": "2026-04-19T10:23:00Z"
}
```

`ork doctor deploy --expose` checks for an existing daemon before starting:
- Running, same port → print existing URL, skip start
- Running but PID dead (stale) → clean up, start fresh
- Not running → start new daemon

### 2.4 ork tunnel subcommands

```bash
ork tunnel status    # provider, URL, uptime, local port
ork tunnel stop      # kill daemon, remove state file
ork tunnel restart   # stop + start fresh (new URL from cloudflared)
```

**`ork tunnel status` output:**
```
  Provider: cloudflared
  URL:      https://abc123.trycloudflare.com
  Local:    http://localhost:80
  Uptime:   23m
  Status:   running
```

### 2.5 Local port detection

Detection order for what to forward:
1. Ingress controller NodePort (`kubectl get svc -n ingress-nginx` → nodePort for port 80)
2. kubectl port-forward to ingress controller → local port 8080
3. kubectl port-forward directly to app service (when no ingress)

For `--dev` (kind), the kind cluster is created with the ingress controller
already configured for NodePort 80.

### 2.6 Token storage

ngrok tokens stored at `~/.orkestra/tunnel.yaml`, permissions 0600:
```yaml
providers:
  ngrok:
    authToken: "2abc3def..."
```

Cloudflared: no token required, no storage needed.

### 2.7 Provider interface

```go
// pkg/tunnel/provider.go
type Provider interface {
    Name() string
    Available() bool
    Install(ctx context.Context) error
    Authenticate(ctx context.Context, token string) error
    Start(ctx context.Context, localPort int) (url string, pid int, err error)
    Stop(pid int) error
}
```

Implementations: `pkg/tunnel/cloudflare.go`, `pkg/tunnel/ngrok.go`.

### 2.8 Deploy output with --expose

```
Building my-app...
  ✓ Built (18s)
  ✓ Pushed ghcr.io/myorg/my-app:a3f5c2b

Applying to cluster...
  ✓ Bundle applied
  ✓ 2/2 pods ready

Starting tunnel...
  ✓ https://abc123.trycloudflare.com (cloudflared)

  App:     https://abc123.trycloudflare.com
  Status:  Ready
  Commit:  a3f5c2b

  Tunnel        → ork tunnel status
  Stop tunnel   → ork tunnel stop
  Control Center → ork control start
```

---

## Part 3: Cleanup

### 3.1 cleanupOnShutdown must clear finalizers

**Current behavior:** `cleanupOnShutdown` cleans up RBAC and webhooks when
Orkestra exits gracefully. It does not remove finalizers from resources it
manages.

**The gap:** If Orkestra shuts down while managing resources with finalizers,
those resources are stuck in `Terminating` state forever. Kubernetes will not
delete them until the finalizer is removed, and the component that set the
finalizer (Orkestra) is no longer running.

**Required behavior:** Before Orkestra exits, the housekeeper must walk all
resources it manages across all Katalogs and remove any finalizers it set.
This is blocking — the process does not exit until finalizers are cleared or
a configurable timeout (`SHUTDOWN_GRACE_PERIOD`) is reached.

The implementation walks the informer cache for each registered CRD, finds
objects with Orkestra's finalizer (`orkestra.konductor.io/managed`), and
patches them to remove it.

### 3.2 ork cleanup

For when Orkestra has already been removed from the cluster and the developer
wants to verify nothing was left behind.

```bash
ork cleanup -f .orkestra/katalog.yaml
```

**What it searches for and removes:**

1. **Finalizers** — scans every namespace for resources matching the Katalog's
   CRDs with the Orkestra finalizer attached. Removes them.

2. **Webhooks** — checks for `ValidatingWebhookConfiguration` and
   `MutatingWebhookConfiguration` named after the Katalog. Removes if found.

3. **Conversion webhooks** — checks `spec.conversion.webhook` on CRDs the
   Katalog declared. Resets conversion strategy to `None`.

4. **Orkestra ConfigMap** — the Katalog ConfigMap generated by
   `ork generate bundle`. Located in the Orkestra system namespace. Prompts
   before deletion since removing it stops reconciliation.

5. **Application RBAC** — the ServiceAccount, Role, RoleBinding generated for
   the application (not Orkestra's own RBAC). Named after the app, in the
   app namespace.

6. **Application namespace** — prompts before deleting. Deleting the namespace
   removes everything in it; this is a destructive action that requires
   explicit confirmation.

**Output:**
```
Scanning for Orkestra resources from katalog: my-app

  Finalizers:
    ✓ my-app (my-app-ns) — finalizer removed
    ✓ my-api (my-api-ns) — finalizer removed

  Webhooks:
    ~ No admission webhooks found for my-app

  Orkestra ConfigMap:
    Found: orkestra-system/my-app
    Remove? [y/N]: y
    ✓ ConfigMap removed

  Application RBAC:
    ✓ ServiceAccount my-app/my-app-ns removed
    ✓ Role my-app/my-app-ns removed
    ✓ RoleBinding my-app/my-app-ns removed

  Application namespace my-app-ns:
    Contains: 3 secrets, 2 configmaps, 1 deployment
    Delete namespace? [y/N]: y
    ✓ Namespace my-app-ns deleted

Cleanup complete.
```

**Naming convention for Orkestra's own resources:**
- Orkestra operator RBAC: `orkestra` (ClusterRole, ClusterRoleBinding,
  ServiceAccount in `orkestra-system`)
- Control Center: `orkestra-cc` (no cluster-level roles)
- Application RBAC: `<appName>` (Role, RoleBinding, ServiceAccount in app namespace)

---

## Part 4: --use-compose (v1.2)

### 4.1 Goal

Developers who have a `docker-compose.yaml` already have a complete description
of their services. `--use-compose` reads it and generates a Katalog that
deploys all stateless services as Deployments.

```bash
ork doctor init --use-compose docker-compose.yaml
ork doctor deploy --use-compose docker-compose.yaml --registry ghcr.io/myorg
```

The fast path (Dockerfile + `.env`) remains the default. `--use-compose` is
an opt-in for developers who have a compose file.

### 4.2 Service translation — stateless

Each compose service becomes a Deployment. Services using well-known
infrastructure images (postgres, mysql, redis, kafka, rabbitmq) are separated
into the stateful layer (Part 5). Everything else is treated as a stateless
application service.

**Compose service → Orkestra resources:**

```yaml
# docker-compose.yaml
services:
  web:
    build: .
    ports: ["3000:3000"]
    environment:
      - NODE_ENV=production
  api:
    image: my-registry/api:latest
    ports: ["8080:8080"]
```

Generates a Katalog with two Deployments, two Services, and two entries in
`app.yaml`.

### 4.3 Infrastructure image detection

Orkestra detects known infrastructure images by the image name, regardless of
tag or registry prefix:

| Detected image | Service type |
|---|---|
| `postgres`, `postgresql` | PostgreSQL |
| `mysql`, `mariadb` | MySQL |
| `redis` | Redis |
| `confluentinc/cp-kafka`, `bitnami/kafka` | Kafka |
| `rabbitmq` | RabbitMQ |
| `mongo`, `mongodb` | MongoDB |

When detected, `ork doctor` surfaces them during examination:

```
Examining docker-compose.yaml...

  Stateless services (Deployments):
    ✓ web    (from Dockerfile, port 3000)
    ✓ api    (my-registry/api, port 8080)

  Stateful services (StatefulSets — v1.3):
    ✓ postgres  (postgres:16)
    ✓ redis     (redis:7-alpine)

  Note: Stateful services require ork doctor v1.3 or later.
  For now, make sure postgres and redis are accessible
  in namespace my-app-ns as:
    postgres-db.my-app-ns.svc.cluster.local
    redis.my-app-ns.svc.cluster.local
```

In v1.2, stateful services are detected and flagged but not deployed.
They must exist in the namespace (deployed by other means) for the
stateless services to connect to them.

---

## Part 5: Stateful services — databases, queues, admin UIs (v1.3)

### 5.1 The problem this solves

Every team that deploys a backend with a database does the same setup: a
StatefulSet for the database, a PersistentVolumeClaim for storage, a Secret
for credentials, a Service for DNS, and optionally an admin UI. This is done
correctly once by someone who knows Kubernetes, and then copied imperfectly
by everyone else. Orkestra solves it once. Every developer who has postgres
in their compose file gets the correct setup automatically.

### 5.2 Implementation repository

Stateful service implementations live in `github.com/orkspace/orkestra-motifs`.
This is a separate repository from the main Orkestra runtime. Each service is
a Katalog fragment — a `services/postgres/katalog.yaml`, `services/redis/katalog.yaml`,
etc. — that `ork doctor` imports when a matching service is detected.

The implementations are battle-tested, maintained by the community, and versioned
independently from the Orkestra runtime. When someone improves the postgres
StatefulSet (better health checks, better shutdown handling), every user of
`ork doctor` benefits.

### 5.3 Generated resources for stateful services

**PostgreSQL example:**

From compose:
```yaml
services:
  postgres:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
```

Generates in the Katalog:

```yaml
statefulsets:
  - name: postgres-db
    image: "{{ .spec.postgresImage }}"
    volumeClaim:
      name: pgdata
      size: "{{ .spec.postgresVolumeSize }}"
    env:
      POSTGRES_USER:
        value: "{{ .spec.postgresUser }}"
      POSTGRES_PASSWORD:
        secretRef:
          name: my-app-secrets
          key: POSTGRES_PASSWORD
    reconcile: true
```

In `app.yaml`:

```yaml
  # PostgreSQL configuration
  # Image version from your docker-compose.yaml
  postgresImage: "postgres:16"

  # Storage size for your database — resize before first deploy
  postgresVolumeSize: "10Gi"

  # Database username (defaults to your GitHub username)
  postgresUser: "myusername"

  # Password is auto-generated — see my-app-secrets in cluster
```

In the generated Secret (from `.env` generation step):

```yaml
POSTGRES_PASSWORD: <randomAlphanumeric 32, once: true>
```

`once: true` means the password is generated on first deploy and never
regenerated, even across redeploys. The database password is stable.

### 5.4 Admin UIs

Each stateful service gets an admin UI deployed alongside it, exposed through
the same tunnel as the application.

| Service | Admin UI | Access |
|---|---|---|
| PostgreSQL | pgAdmin | pgAdmin URL in deploy output |
| MySQL / MariaDB | phpMyAdmin | phpMyAdmin URL |
| Redis | RedisInsight | RedisInsight URL |
| Kafka | Kafka UI | Kafka UI URL |
| RabbitMQ | RabbitMQ Management | Already built in |
| MongoDB | mongo-express | mongo-express URL |

**pgAdmin defaults:**

```yaml
pgadminEmail: "{{ git config user.email }}"  # from git config
pgadminPassword: "{{ git config user.name }}" # GitHub username
```

The developer sees this in deploy output:

```
  PostgreSQL:  postgres-db.my-app-ns.svc.cluster.local:5432
  pgAdmin:     https://pgadmin-abc123.trycloudflare.com
               Email:    developer@example.com
               Password: myusername

  Redis:       redis.my-app-ns.svc.cluster.local:6379
  RedisInsight: https://redis-ui-abc123.trycloudflare.com
```

These defaults are documented clearly. The developer can override them in
`app.yaml`. They are better defaults than what most developers set manually.

### 5.5 Storage and PV/PVC

PVCs are added to the generated Katalog. Volume size is in `app.yaml` with a
sensible default (10Gi) and a comment reminding the developer to resize before
first deploy (resizing after is harder).

Storage class is inferred from the cluster:
- Kind clusters: `standard` (local-path-provisioner, included with kind)
- Cloud clusters: detected from `kubectl get storageclass` (first default class)
- Overridable via `app.yaml`: `postgresStorageClass: "my-storage-class"`

### 5.6 Multi-service app.yaml

When compose has multiple services, `app.yaml` grows to cover all of them:

```yaml
# my-app/app.yaml

# ── Application ───────────────────────────────────────────────────────────
  port: "3000"
  replicas: "2"
  minReplicas: "2"
  maxReplicas: "10"
  host: ""
  image: ""

# ── PostgreSQL ────────────────────────────────────────────────────────────
  postgresImage: "postgres:16"
  postgresVolumeSize: "10Gi"
  postgresUser: "myusername"

# ── Redis ─────────────────────────────────────────────────────────────────
  redisImage: "redis:7-alpine"
  redisVolumeSize: "2Gi"
```

The developer sees their entire stack's configuration in one file. They change
`postgresImage: "postgres:17"` and run `ork doctor deploy` — Orkestra updates the
StatefulSet.

---

## Part 6: Developer example pack

### Pack: `developer`

```bash
ork init my-app --pack developer
```

Five examples that take a developer from zero to production-grade with
databases and admin UIs, no Kubernetes knowledge required.

| Example | What it demonstrates |
|---|---|
| `01-single-project` | Dockerfile + .env → running deployment with HPA, PDB |
| `02-multi-project` | Two services, same cluster, internal DNS |
| `03-deletion-protection` | Security block, accidental deletion prevention |
| `04-notify-me` | Slack alerts on pod failures and high error rates |
| `05-rollback` | Bad deploy → instant rollback |

Pack `developer-db` (v1.3, separate):

| Example | What it demonstrates |
|---|---|
| `01-with-postgres` | compose → StatefulSet + pgAdmin + tunnel URL |
| `02-with-redis` | Redis + RedisInsight |
| `03-full-stack` | App + Postgres + Redis, all exposed via tunnel |

### Pack hierarchy

```
beginner       → Kubernetes operators, learning the model
intermediate   → Multi-resource patterns, Komposer
advanced       → Hooks, constructors, registries
use-cases      → Full-stack platform engineering patterns
developer      → Local to production, no Kubernetes required
developer-db   → Full stack including databases (v1.3)
```

---

## Part 7: Complete developer flow

### Stateless app (v1.1)

```
Install
  curl -sSL https://install.orkestra.dev | sh
  # or: brew install orkspace/tap/ork

Examine
  cd my-app/
  ork doctor
  → detects Go app, port 8080, 12 env vars, SLACK_WEBHOOK_URL
  → suggests --notify-me

Prepare
  ork doctor init --notify-me
  → .orkestra/katalog.yaml  (with notification block)
  → .orkestra/app.yaml      (fill in host if you have a domain)
  → .gitignore updated      (.orkestra/bundle/ added)

Deploy locally
  ork doctor deploy --dev --expose
  → kind binary downloaded (~/.orkestra/bin/kind)
  → kind-orkestra-dev cluster created
  → kubectl and helm downloaded if needed
  → metrics-server installed
  → app built and pushed
  → deployed to kind cluster
  → ingress-nginx installed
  → cloudflared tunnel started
  → https://abc123.trycloudflare.com

Share
  → send URL to teammate
  → test webhook at https://abc123.trycloudflare.com/webhook

Something fails
  → Slack: "my-app: 2/2 pods not ready — CrashLoopBackOff"
  ork doctor deploy rollback
  → previous image restored, 2/2 pods ready

Second project, same cluster
  cd my-api/
  ork doctor init
  ork doctor deploy --dev --expose
  → existing kind cluster detected, reused
  → registered in ~/.orkestra/deploy/komposer.yaml
  → Orkestra picks up new CRD without restart
  → https://xyz789.trycloudflare.com

Production
  ork doctor deploy --registry ghcr.io/myorg
  → identical commands, real cluster

Clean up
  ork cleanup -f .orkestra/katalog.yaml
  → finalizers removed
  → webhooks removed
  → RBAC removed
  → namespace deleted (with confirmation)
```

### With database (v1.3)

```
cd my-app/
ork doctor init --use-compose docker-compose.yaml --notify-me
→ detects: web (Dockerfile), api (my-registry/api), postgres:16, redis:7
→ generates katalog with StatefulSets for postgres and redis
→ generates app.yaml with postgresVolumeSize: "10Gi"

ork doctor deploy --dev --expose
→ all services deployed
→ pgAdmin + RedisInsight deployed and exposed

Output:
  App:          https://abc123.trycloudflare.com
  pgAdmin:      https://pgadmin-xyz.trycloudflare.com
                Email: developer@example.com / Password: myusername
  RedisInsight: https://redis-ui-xyz.trycloudflare.com
```

---

## Phasing

### v1.1 (current sprint)

- `ork doctor` rename with backward-compatible alias
- Git metadata (author, license) in Katalog
- SMTP/Slack detection + `--notify-me`
- Dependency auto-install: kubectl, helm, kind (static binaries)
- metrics-server auto-install
- ingress-nginx auto-install (already implemented)
- `app.yaml` replacing `cr.yaml`
- Scoped RBAC for generated deployments
- `cleanupOnShutdown` finalizer removal
- `ork cleanup -f katalog.yaml`
- `ork doctor deploy --expose` with cloudflared (default) and ngrok (fallback)
- `ork tunnel status / stop / restart`
- Developer example pack (5 examples)

### v1.2 (next sprint)

- `--use-compose` for stateless services
- Multi-service `app.yaml`
- Stateful service detection with guidance

### v1.3

- `orkspace/orkestra-motifs` repository with StatefulSet implementations
- PostgreSQL + pgAdmin
- MySQL + phpMyAdmin
- Redis + RedisInsight
- Kafka + Kafka UI
- RabbitMQ Management
- PV/PVC generation with storage class detection
- Developer-db example pack