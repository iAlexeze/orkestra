# 06 — Pattern zoo: a complete data platform

You have written a WebApp operator, a Cache operator, and a PlatformApp operator with admission policies. Each runs independently. Now compose all of them — alongside seven production-grade data infrastructure patterns from the Orkestra Registry — into a single runtime.

One binary. Nine operators. A complete platform.

Postgres, MySQL, MongoDB, Redis, Kafka, RabbitMQ, a deployment stack, your webapp, your cache — all managed by one process with isolated reconcile queues, shared health observability, and zero controllers written for the infrastructure layer.

---

## What you are composing

| Pattern | Kind | Source |
|---------|------|--------|
| `postgres:v1.0.0` | `Postgres` | Orkestra Registry |
| `mysql:v1.0.0` | `MySQL` | Orkestra Registry |
| `mongodb:v1.0.0` | `MongoDB` | Orkestra Registry |
| `redis:v1.0.0` | `Redis` | Orkestra Registry |
| `kafka:v1.0.0` | `Kafka` | Orkestra Registry |
| `rabbitmq:v1.0.0` | `RabbitMQ` | Orkestra Registry |
| `deployment-stack:v1.0.0` | `Application` | Orkestra Registry |
| local webapp katalog | `WebApp` | step 02 |
| local cache katalog | `Cache` | step 03 |

---

## Inspect before you pull

The patterns for this example come from the official Orkestra Registry. Unset your registry vars so `ork patterns` and `ork inspect` resolve there by default:

```bash
unset ORK_REGISTRY
unset ORK_MOTIFS_REGISTRY
```

Every pattern ships with simulate and E2E results baked into the artifact. Browse and inspect before pulling:

```bash
ork patterns
```

```bash
ork inspect postgres:v1.0.0
ork inspect rabbitmq:v1.0.0
```

Each inspection shows simulate status, E2E status, assertion counts, and when the tests last ran — without touching a cluster.

---

## Pull all patterns

```bash
ork pull -f komposer.yaml
```

Downloads all seven OCI patterns and their motif dependencies to the local cache. Every subsequent command — `ork template`, `ork validate`, `ork simulate` — resolves offline from here.

---

## Review what nine operators compose to

```bash
ork template
ork validate
```

`ork template` shows the full merged operator configuration: nine CRDs, all motif expansions, all admission rules, all resource templates — exactly what the runtime will receive. Review it before generating anything.

---

## Deploy

**1. Apply all CRDs**

```bash
kubectl apply -f ./crds/
kubectl apply -f ../02-katalog-api/crd.yaml
kubectl apply -f ../03-katalog-cache/crd.yaml
```

**2. Generate and apply the bundle**

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

**3. Install Orkestra**

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

**4. Verify**

```bash
kubectl rollout status deployment/orkestra-runtime -n orkestra-system
```

---

## Run the platform

Create an instance of every operator:

```bash
kubectl apply -f cr-postgres.yaml
kubectl apply -f cr-mysql.yaml
kubectl apply -f cr-mongodb.yaml
kubectl apply -f cr-redis.yaml
kubectl apply -f cr-kafka.yaml
kubectl apply -f cr-rabbitmq.yaml
kubectl apply -f cr-deployment-stack.yaml
kubectl apply -f ../02-katalog-api/cr.yaml
kubectl apply -f ../03-katalog-cache/cr.yaml
```

Watch one process reconcile all nine:

```bash
kubectl get postgres,mysql,mongodb,redis,kafka,rabbitmq,applications,webapp,cache -A
```

---

## Observe the platform in the Control Center

```bash
kubectl port-forward svc/orkestra-cc -n orkestra-system 8081:8081
# open http://localhost:8081
```

The Control Center shows every operator side by side: reconcile cycle counts, queue depth, last error, resource health — one dashboard for the entire platform. Each CRD has its own isolated queue. A spike in Kafka reconcile pressure does not slow down Postgres. A failed MongoDB reconcile does not affect Redis.

This is the operational view of what you just built.

---

> **Note:** On a single-node cluster (kind), some database pods may stay Pending due to CPU pressure from running nine operators simultaneously. This is expected — `kubectl describe pod <name>` will show `Insufficient cpu`. Apply only the CRs you need, or provision a multi-node cluster for the full zoo.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [07-upgrade](../07-upgrade/README.md) — explore pattern upgrade strategies
