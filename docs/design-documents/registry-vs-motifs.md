# orkestra-registry vs orkestra-motifs

*Orkestra Project — April 2026*

---

## The short answer

| | orkestra-registry | orkestra-motifs |
|---|---|---|
| **Unit** | Operator pattern (Katalog) | Infrastructure deployment (StatefulSet + Service) |
| **Audience** | Platform engineers | Developers |
| **Consumer knows** | CRDs, operators, Kubernetes | Docker, docker-compose |
| **Mechanism** | Komposer sources → `ork registry pull` | `ork deploy --use-compose` |
| **Complexity** | Embraced | Hidden |
| **CRD required** | Yes | No |
| **Reconciliation** | Yes — Orkestra manages lifecycle | No — applied once, Kubernetes manages |
| **Repository** | `github.com/orkspace/orkestra-registry` | `github.com/orkspace/orkestra-motifs` |

---

## What each one is

### orkestra-registry

The registry stores reusable operator *behaviors* — complete Katalogs that
define how a CRD should be managed. A platform engineer pushes a postgres
operator that creates StatefulSets, manages credentials, runs backups, monitors
health, and handles ordered deletion. Another platform engineer pulls it and
wires it into their platform using the Komposer.

The unit is an operator. The consumer is someone who understands CRDs and
wants to manage the full lifecycle of a stateful service as a first-class
Kubernetes resource. They declare a `PostgresCluster` CR and expect the
operator to handle everything that flows from that declaration.

```bash
# Platform engineer workflow
ork registry push postgres-operator:v14 ./postgres/
ork registry pull postgres-operator:v14

# In komposer.yaml
sources:
  registry:
    - url: oci://ghcr.io/orkspace/orkestra-registry/postgres-operator:v14
```

The consumer's mental model: *I have a PostgresCluster CRD in my cluster and
a CR that declares my postgres configuration. The operator handles the rest.*

### orkestra-motifs

The motifs repository stores known infrastructure *deployments* — complete,
correct StatefulSet manifests for postgres, mysql, redis, kafka, rabbitmq,
and their companion admin UIs. No CRD. No operator. No Katalog. Just the
Kubernetes resources that make the service run.

`ork doktor` imports these automatically when it detects a known infrastructure
image in `docker-compose.yaml`. The developer never sees the StatefulSet.
They see postgres in `app.yaml` and pgAdmin in the deploy output.

```bash
# Developer workflow — nothing explicit
ork doktor init --use-compose docker-compose.yaml
# ork doktor detects postgres:16
# imports postgres manifest from orkestra-motifs
# adds postgresImage + postgresVolumeSize to app.yaml

ork deploy --dev --expose
# postgres StatefulSet applied
# pgAdmin deployed
# both URLs printed
```

The consumer's mental model: *I have postgres in my docker-compose. I want it
running in my cluster. I do not care how.*

---

## Why they are separate

### The audience is different

A platform engineer importing a postgres operator wants control. They want to
configure PodDisruptionBudgets, set replication topology, define custom backup
schedules, integrate with their existing secret management. The operator pattern
gives them this — the Katalog is declarative and fully configurable.

A developer wants postgres running and pgAdmin accessible. They do not want to
think about PodDisruptionBudgets or replication. The motifs pattern gives
them this — it is opinionated, hardened, and zero-configuration.

Trying to serve both audiences with the same artifact produces a configuration
surface that is too complex for developers and too opinionated for platform
engineers.

### The programming model is different

The registry operates through Orkestra's reconciliation engine. A platform
engineer pulls a postgres operator pattern, the Katalog is loaded, Orkestra
manages the CR lifecycle — creating resources, correcting drift, handling
deletion with finalizers, emitting events, tracking health. This requires
understanding the operatorbox model, CRD lifecycle, and Orkestra's declarative
layer.

The motifs repository operates through direct application. A developer runs
`ork deploy` and the StatefulSet is applied with `kubectl apply`. Kubernetes
manages it from that point. No Orkestra reconciliation. No finalizers. No
custom CRD. Just a postgres pod running in a namespace.

### The complexity contract is different

The registry *embraces* Kubernetes complexity. Platform engineers using it
understand CRDs, namespaces, RBAC, and the operator pattern. The registry
gives them reusable, high-quality implementations of that complexity.

The motifs repository *hides* Kubernetes complexity. Developers using it
(via `ork doktor`) never see a StatefulSet, PVC, or headless Service. The
motifs repository encapsulates that complexity and presents only what the
developer needs: a running database, a URL to the admin UI, and the connection
string.

---

## Where they share ground

Both repositories are:
- Versioned alongside Orkestra releases
- Community-contributed via PR
- Validated in CI against a live kind cluster
- Distributed via the Orkestra release infrastructure

A platform engineer who wants to use the postgres StatefulSet from
`orkestra-motifs` as a base for a richer operator can — they fork the
StatefulSet manifest and wrap it in a Katalog, then publish the result to
the registry. The motifs layer feeds the registry.

---

## Examples side by side

### postgres in the registry (platform engineer)

```yaml
# komposer.yaml
sources:
  registry:
    - url: oci://ghcr.io/orkspace/orkestra-registry/postgres-operator:v14

# PostgresCluster CR (applied by platform engineer)
apiVersion: data.platform.io/v1alpha1
kind: PostgresCluster
metadata:
  name: payments-db
spec:
  version: "16"
  replicas: 3
  storage: 100Gi
  backup:
    schedule: "0 2 * * *"
    s3Bucket: "my-backups"
  resources:
    requests:
      cpu: "2"
      memory: "8Gi"
```

The platform engineer controls topology, backup, resources, and replication.
Orkestra reconciles the PostgresCluster CR and manages the full lifecycle.

---

### postgres in orkestra-motifs (developer)

```yaml
# app.yaml (generated by ork doktor, filled in by developer)
  postgresImage: "postgres:16"
  postgresVolumeSize: "10Gi"
  postgresUser: "myusername"
  # password auto-generated — see my-app-secrets in cluster
```

The developer changes the image version or volume size. `ork deploy` applies
the change. No CR. No operator. No reconciliation loop.

---

## Decision rule

**Use the registry when:**
- The service is a first-class resource in your platform
- You need custom lifecycle management (backup, failover, version upgrades)
- You want CRD-based configuration that platform users interact with directly
- You are building infrastructure for other teams to consume

**Use orkestra-motifs when:**
- You are a developer who wants the service running, not managed
- You are using `ork doktor` and the service was in your docker-compose
- You don't need custom lifecycle management
- You want the simplest path to a running database with a UI

---

## The relationship

```
orkestra-motifs
  ↓ provides battle-tested StatefulSet implementations
  ↓ developers use directly via ork doktor

orkestra-registry
  ↓ platform engineers wrap service manifests in Katalogs
  ↓ add lifecycle management (backup, failover, monitoring)
  ↓ publish as operator patterns
  ↓ other platform engineers pull and compose
```

The motifs layer is the raw material. The registry layer is the value-added
operator built on top. Both are valid end states depending on who the consumer
is and what they need.