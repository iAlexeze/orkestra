# orkestra-services: Do It Once, Use It Everywhere

*Orkestra Project — April 2026*

---

## The unifying insight

orkestra-services does not store StatefulSet manifests. It stores **Katalog
motifs** — partial Katalog declarations with template expressions that
produce complete deployments when instantiated. The same motif serves two
audiences through two different access surfaces. The implementation happens
once. The reuse is structural, not incidental.

```
orkestra-services/postgres/motif.yaml   ← implemented once

    ↓ Developer path                        ↓ Platform engineer path
    ork doctor reads it                     Registry wraps it with a CRD
    ConfigMap-as-CRD                        PostgresCluster CRD
    app.yaml provides values                CR spec provides values
    kubectl apply (no Orkestra needed)      Orkestra manages full lifecycle
```

The template expressions are identical in both paths:
`{{ .spec.postgresImage }}`, `{{ .spec.volumeSize }}`, `{{ .spec.postgresUser }}`

The data source differs. The motif does not.

---

## What a Katalog motif is

A motif is a partial Katalog — it declares resources but does not declare
a CRD entry. It cannot run alone. It must be instantiated by a consumer that
provides the CRD context and the value bindings.

```yaml
# orkestra-services/postgres/motif.yaml

motif:
  name: postgres
  version: v16
  description: >
    PostgreSQL StatefulSet with PVC, headless Service, and pgAdmin.
    Compatible with postgres:14 through postgres:17.

  inputs:
    # Declared inputs — must be satisfied by the consumer
    - name: postgresImage
      description: PostgreSQL image and version
      default: "postgres:16"
    - name: postgresUser
      description: Database superuser name
      default: "postgres"
    - name: volumeSize
      description: PVC storage size
      default: "10Gi"
    - name: passwordSecretRef
      description: Name of the secret containing POSTGRES_PASSWORD

  resources:
    statefulsets:
      - name: "{{ .metadata.name }}-postgres"
        image: "{{ .spec.postgresImage }}"
        replicas: 1
        serviceName: "{{ .metadata.name }}-postgres-headless"
        volumeClaims:
          - name: pgdata
            size: "{{ .spec.volumeSize }}"
            mountPath: /var/lib/postgresql/data
        env:
          POSTGRES_USER:
            value: "{{ .spec.postgresUser }}"
          POSTGRES_PASSWORD:
            secretKeyRef:
              name: "{{ .spec.passwordSecretRef }}"
              key: POSTGRES_PASSWORD
        reconcile: true

    services:
      - name: "{{ .metadata.name }}-postgres-headless"
        clusterIP: None
        port: 5432
        reconcile: true
      - name: "{{ .metadata.name }}-postgres"
        port: 5432
        reconcile: true

    # pgAdmin companion
    deployments:
      - name: "{{ .metadata.name }}-pgadmin"
        image: "dpage/pgadmin4:latest"
        env:
          PGADMIN_DEFAULT_EMAIL:
            value: "{{ .spec.adminEmail }}"
          PGADMIN_DEFAULT_PASSWORD:
            value: "{{ .spec.adminPassword }}"
        reconcile: true

    services:
      - name: "{{ .metadata.name }}-pgadmin-svc"
        port: 80
        targetPort: 80
        reconcile: true

  status:
    fields:
      - path: postgresReady
        value: "{{ replicasReady .children.statefulset }}"
      - path: connectionString
        value: >
          postgres://{{ .spec.postgresUser }}@{{ .metadata.name }}-postgres.{{ .metadata.namespace }}.svc.cluster.local:5432
```

This motif is complete. It describes postgres with pgAdmin fully. What it
does not do is declare where the values come from. That is the consumer's job.

---

## Consumer 1: ork doctor (developer path)

ork doctor reads the motif and instantiates it with a ConfigMap-as-CRD.
The developer never sees the motif, the Katalog, or the StatefulSet.

**What ork doctor generates from the postgres motif:**

In `.orkestra/katalog.yaml`:

```yaml
spec:
  crds:
    my-app:
      apiTypes:
        kind: ConfigMap
      labelSelector:
        ork.io/app: my-app
      allowedNamespaces: ["my-app-ns"]

      operatorBox:
        # motif inputs bound to ConfigMap fields
        onReconcile:
          when:
            - field: spec.postgresImage
              operator: exists

          # Expanded from motif — not written by developer
          statefulsets:
            - name: "{{ .metadata.name }}-postgres"
              image: "{{ .spec.postgresImage }}"
              ...
```

In `app.yaml` (the developer's only interface):

```yaml
data:
  # PostgreSQL
  # Image version from your docker-compose.yaml
  postgresImage: "postgres:16"

  # Storage size for your database
  postgresVolumeSize: "10Gi"

  # Database username
  postgresUser: "myusername"

  # Admin UI credentials (defaults — change if you want)
  adminEmail: "developer@example.com"
  adminPassword: "myusername"
```

The developer changes `postgresVolumeSize: "20Gi"` before first deploy. They
run `ork doctor deploy`. Postgres is running. pgAdmin is accessible. They never wrote
a StatefulSet.

**Application of resources:** Direct `kubectl apply`. No Orkestra operator
needed for the services layer in the developer path. The ConfigMap-based
operator (their app) manages the application resources. The stateful services
are applied once and Kubernetes manages them from that point.

---

## Consumer 2: orkestra-registry (platform engineer path)

A platform engineer takes the same motif and wraps it with a proper CRD.
The result is a full operator pattern published to the registry.

```yaml
# orkestra-registry/postgres-operator/katalog.yaml

spec:
  crds:
    postgres:
      apiTypes:
        group: data.platform.io
        version: v1alpha1
        kind: PostgresCluster
        plural: postgresclusters

      operatorBox:
        default: true

        # motif imported and bound to CRD spec fields
        imports:
          - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
            bindings:
              postgresImage: "{{ .spec.image }}"
              postgresUser:  "{{ .spec.user | default \"postgres\" }}"
              volumeSize:    "{{ .spec.storage }}"
              adminEmail:    "{{ .spec.admin.email }}"
              adminPassword: "{{ .spec.admin.password }}"
              passwordSecretRef: "{{ .metadata.name }}-credentials"

        # Platform engineer adds what the developer path doesn't need:
        # Backup, monitoring, deletion protection, ordered shutdown
        onDelete:
          ordered: true
          groups:
            - jobs:
                - name: "{{ .metadata.name }}-final-backup"
            - statefulsets:
                - name: "{{ .metadata.name }}-postgres"

        security:
          deletionProtection:
            enabled: true
```

The platform engineer's consumer declares a CR:

```yaml
apiVersion: data.platform.io/v1alpha1
kind: PostgresCluster
metadata:
  name: payments-db
  namespace: payments-ns
spec:
  image: "postgres:16"
  user: "payments"
  storage: "100Gi"
  admin:
    email: "dba@company.io"
    password: "securepassword"
```

Orkestra reconciles the PostgresCluster CR. The same postgres StatefulSet and
pgAdmin from the motif are deployed. The platform engineer gets lifecycle
management, deletion protection, ordered shutdown, and health tracking on top.

---

## The `imports:` mechanism

The `imports:` block is new. It allows a Katalog to reference a motif and
bind its inputs to local values. The motif is fetched from the registry
(OCI) or from the local cache (`~/.orkestra/registry/`).

```yaml
imports:
  - motif: oci://ghcr.io/orkspace/orkestra-services/postgres:v16
    bindings:
      postgresImage: "{{ .spec.image }}"
      volumeSize: "{{ .spec.storage }}"
```

The binding values are template expressions evaluated in the context of the
CRD (the CR being reconciled). The motif's template expressions reference
`.spec.*` — these are resolved using the bindings as an intermediate layer.

For the developer path, ork doctor resolves the bindings at generation time
against the ConfigMap fields. The generated Katalog contains the expanded
resources directly — no import at runtime. This means no Orkestra version
dependency for the developer's stateful services.

For the platform engineer path, imports are resolved at Orkestra startup when
the Katalog is loaded. The motif is fetched, bindings are compiled, and the
resulting resource group is added to the operatorbox. Live imports — the
motif can be updated independently of the operator.

---

## The repository structure

```
github.com/orkspace/orkestra-services/
  README.md
  postgres/
    motif.yaml     ← the primitive
    v14/              ← version-pinned snapshots
    v15/
    v16/
    README.md
    example/
      cr.yaml         ← example ConfigMap (developer path)
      crd.yaml        ← example CRD (platform path)
  mysql/
    motif.yaml
    ...
  redis/
    motif.yaml
    ...
  kafka/
    motif.yaml
    ...
  rabbitmq/
    motif.yaml
    ...
```

Each service has one canonical motif. Version directories contain snapshots
for pinned imports. The README for each service documents both usage paths —
direct (developer) and via operator (platform engineer).

---

## The full audience map

```
Developer
  docker-compose.yaml
       ↓
  ork doctor --use-compose
       ↓ reads motif from orkestra-services
       ↓ generates Katalog + app.yaml
       ↓ applies directly
  Running postgres + pgAdmin
  No CRD. No operator knowledge.

Platform engineer (consuming)
  imports motif from orkestra-services into their Katalog
  publishes operator to orkestra-registry
  teams use PostgresCluster CRD
  Full lifecycle management

Platform engineer (building)
  writes motif in orkestra-services
  every consumer — developer and platform engineer — benefits
  one implementation, infinite reuse
```

---

## ork doctor as platform engineer

ork doctor does not expose the operator pattern. It *is* the operator pattern,
operating autonomously on behalf of the developer.

A platform engineer who onboards a new service to their Kubernetes platform:
- Reads the team's requirements (what the developer needs)
- Designs the Katalog (what resources to create)
- Creates the ConfigMap CR interface (how the developer configures it)
- Deploys and maintains it

ork doctor does exactly this, automatically, for every developer who runs
`ork doctor init`. The developer provides the docker-compose.yaml (their
requirements). ork doctor designs the Katalog, creates the app.yaml interface,
and manages the deployment.

The developer is inside the operator pattern without knowing it. The ConfigMap
is their CR. app.yaml is their CR spec. `ork doctor deploy` is their `kubectl apply`.
The Control Center is their operator dashboard.

The pattern is identical. Only the vocabulary changes.

---

## Implementation sequence

**v1.3 — Motif specification and postgres**

1. Define the `motifs.yaml` schema — inputs, resources, status fields
2. Implement `pkg/motifs/` — motifs loader, binding resolver, resource expander
3. Implement `ork doctor --use-compose` with motifs import (developer path)
4. Implement `imports:` in Katalog loader (platform path)
5. Ship `orkestra-services/postgres` with motifs, README, examples
6. Developer example: `01-with-postgres` using `--use-compose`

**v1.4 — Additional services**

MySQL, Redis, Kafka, RabbitMQ — one sprint per service, in order of demand.
Each follows the same pattern: motifs first, developer example, registry
operator pattern.

**v1.5 — Motif registry**

Motifs become first-class OCI artifacts in the registry alongside full
operator patterns. `ork registry push postgres:v16 ./postgres/` pushes the
motif. `ork registry list --type motif` filters to motifs only.
Platform engineers can discover, pull, and compose motifs without reading
the source repository.