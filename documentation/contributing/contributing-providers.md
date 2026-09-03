# Contributing Providers

Providers let an operator manage external infrastructure — cloud resources, databases, message queues — alongside Kubernetes resources, all from the same operatorBox declaration.

---

## Current providers

| Provider | Package | Supported kinds |
|----------|---------|-----------------|
| Google Cloud | `pkg/provider/google` | (see package header) |
| AWS | `pkg/provider/aws` | S3, RDS, Route53 |
| Azure | `pkg/provider/azure` | Blob storage, Service Bus, SQL Database |
| Redis | `pkg/provider/redis` | ACL users, server config |
| PostgreSQL | `pkg/provider/postgres` | (see package header) |
| MySQL | `pkg/provider/mysql` | (see package header) |
| MongoDB | `pkg/provider/mongo` | (see package header) |

Each provider is a standalone Go package that is **not imported by the runtime binary**. Users add the provider as a dependency in their own `go.mod` and register it via their `typeregistry` init.

---

## Where contribution is needed

### Expanding existing providers

Each provider currently handles a small set of resource kinds. Extending an existing provider means adding a new `case` in its `Reconcile` and `Delete` switch. For example:

- **AWS**: SQS queues, ElastiCache clusters, EKS node groups, IAM roles
- **Azure**: Key Vault secrets, Container Registry, AKS node pools
- **Google**: GCS buckets, Cloud SQL, Pub/Sub topics, GKE node pools
- **Redis**: Pub/Sub channel config, key TTL policies
- **Postgres / MySQL / MongoDB**: user management, database creation, schema migrations

### New providers

Orkestra does not yet have providers for:

- **Vault** (HashiCorp) — secret leasing, dynamic credentials
- **Kafka** — topic and ACL management
- **Elasticsearch / OpenSearch** — index and mapping management
- **Cloudflare** — DNS records, Workers KV
- **Datadog** — monitor and dashboard provisioning
- **GitHub** — repository and team management via the GitHub API

---

## How to add a provider

### 1. Create the package

```text
pkg/provider/<name>/
  provider.go
```

### 2. Implement the interface

```go
type Provider interface {
    Name() string
    Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error
    Delete(ctx context.Context, req orktypes.DeleteRequest) error
}
```

`ReconcileRequest` carries the resource kind, name, namespace, and a free-form `Spec map[string]interface{}` parsed from the Katalog YAML. The provider is responsible for interpreting that map.

### 3. Add `NewFromAuth`

```go
func NewFromAuth(ctx context.Context, auth map[string]string) (*Provider, error)
```

`auth` is the key/value map from `providers[].auth` in the Katalog. Support `$ENV_VAR` expansion for any secret values.

### 4. Document supported kinds

Add a table to the package-level comment listing every `kind` your provider handles and what fields it expects in `Spec`. Follow the pattern in `pkg/provider/aws/provider.go`.

### 5. Write tests

Use `go test` with real credentials in a CI environment variable, or use provider-specific fake/mock clients. Integration tests are preferred over mocks.

---

## Provider conventions

- Never panic — return wrapped errors.
- Be idempotent: reconcile is called on every sync cycle, not just on changes.
- Treat missing `auth` keys as an error at construction time (`NewFromAuth`), not at reconcile time.
- Log with `logger.Info()` / `logger.Debug()` — never `fmt.Println`.
- Use the `$ENV_VAR` expansion from `utils.ExpandEnv` for auth values.
