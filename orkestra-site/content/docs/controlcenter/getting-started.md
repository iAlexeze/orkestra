---
title: "Getting Started"
weight: 11
---

# Getting Started

The Orkestra Control Center is bundled with the `ork` CLI and can be launched immediately without configuration.

## Starting the Control Center

The simplest way to run the Control Center:

```bash
ork control start
```

Defaults:

- Port: `8081`
- Runtime URLs: `http://localhost:8080`

Visit:

```
http://localhost:8081
```

You will be presented with the login page.

## Authentication

The Control Center uses a lightweight login system backed by a signed session cookie.
> Find out how it is configured in in the [`environment variables`](#environment-variables) section

### Session Behavior

- Session is stored in a signed, HttpOnly cookie  
- Cookie expires when the browser is closed  
- `/logout` clears the session immediately  

## Starting With No Runtimes

You can start the Control Center without any preconfigured runtime URLs:

```bash
ork control start --ignore-default
```

This is useful when:

- You want to add runtimes from the UI  
- You are monitoring multiple clusters  
- You are deploying the Control Center in Kubernetes  

## Managing Runtimes in the UI

The Control Center includes a runtime management panel where you can:

- Add new runtime URLs  
- Update existing URLs  
- Delete runtimes  
- See online/offline status  
- Validate URLs before saving  

Each runtime is checked via:

```
<runtime-url>/health
```

## Navigating the Control Center

### Control Center (Global View)

Shows:

- All Katalogs across all runtimes  
- Health summaries  
- Navigation to each Katalog  

### Control Panel (Per‑Katalog View)

Shows:

- CRD cards  
- Worker pools  
- Queue pressure  
- Error rates  
- RBAC  
- Dependencies  
- Enriched configuration  

### CRD Detail View

Shows:

- Worker pool visualization  
- Queue depth  
- Reconcile metrics  
- Admission webhook metrics  
- Version conversion metrics  
- Dependencies  
- RBAC  
- Reconciler configuration  

---

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port` | `8081` | Port to serve the Control Center on |
| `-u, --urls` | `http://localhost:8080` | Comma-separated list of Orkestra runtime URLs |
| `--refresh` | `10s` | Interval for fetching fresh Katalog data |
| `--log-level` | `info` | Log level (debug, info, warn, error) |
| `--ignore-default` | `false` | Do not start with the default runtime URLs |
| `--version` | - | Show version information |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ORK_CC_PORT` | Override the default port | `8081` |
| `ORK_CC_REFRESH` | Override refresh interval | `10s` |
| `LOG_LEVEL` | Set log level | `info` |
| `IGNORE_DEFAULT` | Do not start with the default runtime URLs | `false` |
| `ADMIN_USERNAME` | Login username | `admin` |
| `ADMIN_PASSWORD` | Login password | `admin` |
| `SESSION_SECRET` | Secret used to sign session cookies | `dev-secret` |

---
