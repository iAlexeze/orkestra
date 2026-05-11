# Example: App + PostgreSQL

Deploy a Go application with a PostgreSQL database and pgAdmin — no Kubernetes
knowledge required.

## What you need

| Tool | Install |
|---|---|
| Docker | https://docs.docker.com/get-docker/ |
| `ork` | `curl -sSL https://get.orkestra.sh \| sh` |
| A cluster | Existing cluster **or** use `ork doctor deploy --dev` to create one |

A registry to push your image to (e.g. `ghcr.io/yourorg`).

## Step 1 — Examine your project

```bash
cd examples/developer/06-docker-compose-with-postgres

cp .env.example .env
ork doctor
```

Because there is a `docker-compose.yaml` in this example directory, `ork doctor`
will show the infrastructure services it detected:

```
  ✓ docker-compose.yaml
  💡 Infrastructure services detected in docker-compose.yaml:
      postgres (postgres:16) → postgres Motif + pgAdmin
  Run 'ork doctor init --name my-app --use-compose docker-compose.yaml' to include them
```

## Step 2 — Generate the configuration

```bash
ork doctor init --name my-app --use-compose ../docker-compose.yaml
```

`ork doctor init` generates three files:

- `.orkestra/katalog.yaml` — the complete Orkestra operator declaration,
  including a Motif import for postgres
- `.orkestra/app.yaml` — your configuration (port, replicas, postgres image, volume size)
- `.orkestra/values.yaml` — Helm values for the Orkestra operator

Open `.orkestra/app.yaml`. You will see:

```yaml
data:
  port: "8080"
  replicas: "2"

  # PostgreSQL (from docker-compose.yaml)
  postgresImage: "postgres:16"
  postgresVolumeSize: "10Gi"   # resize before first deploy if needed
  postgresUser: "myapp"
  adminEmail: "developer@example.com"
  adminPassword: "admin"
```

Change `postgresVolumeSize` if you need more space. Change the admin credentials
if you want. That is the only Kubernetes-free configuration surface for the entire
stack.

## Step 3 — Deploy

**With an existing cluster:**
```bash
ork doctor deploy --registry ghcr.io/yourorg
```

**With a local kind cluster (created automatically):**
```bash
ork doctor deploy --registry ghcr.io/yourorg --dev
```

Orkestra:

1. Builds your Docker image tagged with the current Git commit SHA
2. Pushes to the registry
3. Generates a Kubernetes Secret from your `.env` secrets
4. Expands the postgres Motif into StatefulSet, Services, pgAdmin
5. Installs or verifies the Orkestra operator
6. Applies everything to the cluster
7. Waits for all pods to be ready

## Step 4 — What's running

```
  ✓ 2/2 pods ready (web)
  ✓ 1/1 pod ready (postgres)
  ✓ pgAdmin deployed

  Internal URLs (for wiring to other projects):
    export MY_APP_URL=http://my-app-orkestra-svc.my-app-orkestra-ns.svc.cluster.local:8080
    export POSTGRES_URL=postgres://myapp@my-app-orkestra-postgres.my-app-orkestra-ns.svc.cluster.local:5432

  Control Center:
    kubectl port-forward svc/orkestra-cc 8081:8081 -n orkestra-system &
    open http://localhost:8081
```

## Step 5 — View in Control Center

Open the Control Center:

```bash
kubectl port-forward svc/orkestra-cc 8081:8081 -n orkestra-system &
```

Then go to `http://localhost:8081`. You will see:

- Your Katalog listed on the left
- Click it → see the CR (`my-app-orkestra`)
- Click **Resources** (top right) → see every resource Orkestra created:
  Deployment, StatefulSet, Services, Secret, ConfigMap, HPA, PDB
- Click any resource → see its events, conditions, and current status

You never wrote any of these resources. Orkestra created them from the Motif.

## Step 6 — Access pgAdmin

```bash
kubectl port-forward svc/my-app-orkestra-pgadmin-svc 5050:80 -n my-app-orkestra-ns &
open http://localhost:5050
```

Login with the credentials from `app.yaml` (`adminEmail` / `adminPassword`).

Add a server:
- Host: `my-app-orkestra-postgres.my-app-orkestra-ns.svc.cluster.local`
- Port: `5432`
- User: your `postgresUser` value
- Password: look up the generated secret:
  ```bash
  kubectl get secret my-app-orkestra-secrets -n my-app-orkestra-ns -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d
  ```

## Cleanup

```bash
ork cleanup -k .orkestra/katalog.yaml
```

This removes finalizers, RBAC, the Orkestra ConfigMap, and (with confirmation)
the application namespace. The postgres PVC is deleted with the namespace.

---

The postgres Motif is defined in `motifs/postgres/motif.yaml` in this directory.
In production it will be fetched from the Orkestra Motif registry automatically.
The file here lets you validate it locally:

```bash
ork validate -k motifs/postgres/motif.yaml
```
