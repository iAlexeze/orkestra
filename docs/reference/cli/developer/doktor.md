# ork doktor

Examine the current directory and report everything Orkestra detected about it.
Run this before your first deploy to verify Orkestra can see your project correctly.

```bash
ork doktor [flags]
ork doktor init --name <app> [flags]
```

---

## ork doktor

Scans the current directory and prints a summary: language, port, `.env` variables,
what Kubernetes resources Orkestra will create, and which CLI tools are missing.

```bash
ork doktor
```

### What it checks

| Check | How it's detected |
|-------|------------------|
| Dockerfile | `Dockerfile` exists in project root |
| Git | `.git/HEAD` — prints the short commit SHA |
| Language | First match: `go.mod` Go, `package.json` Node.js, `pom.xml` Java, `requirements.txt` Python, `Gemfile` Ruby, `Cargo.toml` Rust |
| Port | `PORT` in `.env`, or language default (Go/Java/Rust `8080`, Node.js/Ruby `3000`, Python `8000`) |
| `.env` variables | Parsed from `.env`; lines tagged `# ork:cfg` become ConfigMap entries, all others become Secrets |
| Frontend detection | `build/`, `dist/`, or `public/` directories; or `package.json` with React, Vue, Angular, Next, Nuxt, or Svelte |
| SMTP / Slack | Prefixes `SMTP_` or `SLACK_` in `.env` — triggers a `--notify-me` hint |
| Missing tools | `kubectl` and `helm` availability — both are auto-installed by `ork deploy` |

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-ha` | false | Show plan without HPA and PDB (single replica) |
| `--no-secure` | false | Show plan without deletion protection |
| `--clean` | false | Show plan with `cleanupOnShutdown: true` in deletion protection |

### Example output

```
Examining project...

  ✓ Dockerfile found
  ✓ Git repository — commit: a3f9c12
  ✓ Language: Go  (go.mod)
  ✓ Port: 8080
  ✓ .env — 6 variables
      2 config  (# ork:cfg)
      4 secrets (default)

  ~ SMTP/Slack detected in .env
    Run 'ork doktor init --name my-api --notify-me' to wire notifications.

Orkestra will create:
  Deployment     image built from Dockerfile, tagged :a3f9c12
  Secret         my-api-secrets (4 variables from .env)
  ConfigMap      my-api-config  (2 variables from .env # ork:cfg)
  Service        port 8080
  HPA            min 2 / max 10
  PDB            minAvailable: 1
  DeletionProtection  enabled

Missing dependencies:
  (none)

Run 'ork doktor init' to generate .orkestra/katalog.yaml
```

---

## ork doktor init

Generate the `.orkestra/` configuration directory for a project. Creates three files
and extends `.gitignore` to exclude the build output.

```bash
ork doktor init --name <app> [flags]
```

`--name` is required. It determines the CR name (`<name>-orkestra`), namespace
(`<name>-orkestra-ns`), and all resource names derived from them.

### Files created

| File | Purpose |
|------|---------|
| `.orkestra/katalog.yaml` | Declares the Kubernetes resources Orkestra manages — edit freely |
| `.orkestra/app.yaml` | The ConfigMap CR you own — set `port`, `replicas`, `host`, `image` |
| `.orkestra/values.yaml` | Helm values for the Orkestra operator installation |

`.orkestra/bundle/` is added to `.gitignore` automatically — it contains generated
cluster manifests and should not be committed.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | — | **Required.** App name (e.g. `my-api`) |
| `--no-ha` | false | Single replica — skip HPA and PDB |
| `--no-secure` | false | Skip deletion protection labels and webhook |
| `--clean` | false | Add `cleanupOnShutdown: true` to deletion protection |
| `--add-ingress` | false | Include Ingress even when no frontend was auto-detected |
| `--notify-me` | false | Wire deployment notifications to the `developer` team |

### --notify-me

When `--notify-me` is set, `ork doktor init` does three things:

1. Adds a `notification:` block to `katalog.yaml` with a `developer` team built
   from your Git author email and any Slack channel from `.env`:

   ```yaml
   notification:
     enabled: true
     defaults:
       interval: 15m
     teams:
       developer:
         email:
           - you@example.com     # from git log -1 --pretty=format:%ae
         slack:
           - "#deployments"      # present when SLACK_WEBHOOK_URL is in .env
         message: "{{ .metadata.name }}: {{ .status.phase }}"
   ```

2. Adds `runtime.extraEnvFrom` to `values.yaml` so the Orkestra runtime can read
   SMTP and Slack credentials from the `orkestra-notification` Secret:

   ```yaml
   runtime:
     extraEnvFrom:
       - secretRef:
           name: orkestra-notification
   ```

3. The `notify:` block is always present in every generated katalog (even without
   `--notify-me`) so the deployment readiness condition is wired to `developer`
   from day one — no code or re-init required if you add channels later.

### .orkestra/app.yaml

The generated `app.yaml` is the only Kubernetes object you manage directly.
Fill in the fields before your first deploy:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-api-orkestra
  namespace: my-api-orkestra-ns
  labels:
    ork.io/app: my-api-orkestra
data:
  port: "8080"
  replicas: "2"
  minReplicas: "2"
  maxReplicas: "10"
  host: ""                    # public hostname for Ingress (leave empty if not needed)
  controlCenterHost: ""       # Orkestra Control Center hostname
  image: ""                   # set automatically by ork deploy — do not edit
```

---

## Next steps

After `ork doktor init`:

1. Review `.orkestra/katalog.yaml` — it is generated but editable
2. Fill in `.orkestra/app.yaml` — replicas, host, controlCenterHost
3. Run [`ork deploy`](./deploy.md) to build, push, and deploy

---

**← CLI Reference** [index](../index.md) | **Next →** [ork deploy](./deploy.md)
