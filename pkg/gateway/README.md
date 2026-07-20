# pkg/gateway

`gateway/apply` implements the Orkestra Apply API — a CRUD REST surface for Kubernetes Custom Resources served by the gateway process. It lets any HTTP client create, read, or delete CRs managed by Orkestra without a kubeconfig or kubectl.

```
POST   /api/v1/apply
GET    /api/v1/resources/{kind}/{namespace}/{name}
GET    /api/v1/resources/{kind}/{namespace}
DELETE /api/v1/resources/{kind}/{namespace}/{name}
GET    /api/v1/schema/{kind}
```

## Enabling

The Apply API is off by default. Enable it in the Katalog:

```yaml
gateway:
  applyAPI:
    enabled: true
    auth:
      tokens:
        - name: ci-pipeline
          token: "${CI_ORK_TOKEN}"
```

Enable the Create button in the Control Center per CRD:

```yaml
spec:
  crds:
    application:
      idp:
        enabled: true
```

## What callers can do

| Caller | Path |
|--------|------|
| Control Center | POST + GET to render a form, submit, and display status |
| CI pipeline | `curl POST /api/v1/apply` — no kubeconfig in the runner |
| Terraform provider | create → POST, read → GET, destroy → DELETE |
| Slack bot | parse command → build CR → POST → poll GET → reply |
| Any HTTP client | any language, any platform |

Security — admission rules, namespace protection, deletion protection — is enforced exactly as it is for `kubectl apply` through the gateway webhook path. No new configuration.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand why the Apply API lives in the gateway | [docs/01-overview.md](docs/01-overview.md) |
| Understand the full apply pipeline (auth → validation → admission → SSA) | [docs/02-apply.md](docs/02-apply.md) |
| Understand the schema endpoint and how CRD schemas are resolved | [docs/03-schema.md](docs/03-schema.md) |
| Understand GET and DELETE, deletion protection, polling patterns | [docs/04-resources.md](docs/04-resources.md) |
| Understand token auth, add a new auth mode | [docs/05-auth.md](docs/05-auth.md) |
