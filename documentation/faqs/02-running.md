# Running Orkestra

---

## Can Orkestra manage multiple CRDs?

Yes — any number. This is the point.

Each CRD in a Katalog gets its own complete, isolated operator stack:

- Dedicated informer watching its exact GVK and API version
- Dedicated workqueue with independent depth and backoff
- Dedicated worker pool — other CRDs cannot consume its workers
- Dedicated health endpoint at `/katalog/{crd}/health`
- Dedicated Prometheus metrics labeled by GVK

All of these operator stacks run inside one Orkestra process. The isolation is at
the logic level. The shared infrastructure — API server connection, informer factory,
health server, leader election — is paid once.

!!! tip "The economics"
    15 separate operator processes: ~750 MB–3 GB memory, 15 health endpoints,
    15 metric schemas, 15 upgrade procedures.
    Orkestra managing 15 CRDs: ~50 MB memory, 1 health server, 1 metric schema,
    1 upgrade procedure.

---

## How do I start Orkestra?

Locally, for development:

```bash
ork run
# Orkestra reads katalog.yaml from the current directory and starts the runtime.
```

In a cluster, via Helm:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set runtime.katalog.existingConfigMap=my-katalog-configmap
```

See [Deploying](../deploying.md) for full cluster setup including TLS, RBAC, and production tuning.

---

## What does `ork validate` do?

`ork validate` runs the complete Katalog loading sequence without starting the runtime.
It surfaces every configuration error — bad YAML, unknown kinds, circular dependencies,
missing registry files, empty pattern files — before any cluster changes are made.

```bash
ork validate

✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 3 / resync: 15s
    validation: 2 rules / mutation: 1 rule

✗ application
    error: circular dependency: application → namespace → application
```

!!! tip "Run in CI"
    `ork validate` exits with a non-zero code on any error. Add it to your CI
    pipeline to catch Katalog errors before they reach the cluster:
    ```yaml
    - name: Validate Katalog
      run: ork validate
    ```
    It requires no cluster connection — safe to run in any CI environment.

---

## Does Orkestra require cert-manager?

No. Orkestra needs TLS certificates for its HTTPS server (used by conversion
and admission webhooks) when `ENABLE_CONVERSION=true` or `ENABLE_ADMISSION_WEBHOOK=true`.
Where those certificates come from is your choice.

| Approach | Suitable for |
|---|---|
| Self-signed (generated at startup) | Development and testing |
| cert-manager `Certificate` resource | Production — automated renewal |
| External PKI / corporate CA | Enterprise environments with existing PKI |
| Cloud provider managed certs | Cloud-native deployments |

If no certificate is provided, Orkestra generates a self-signed certificate at startup and uses it automatically. This is the default behaviour — you do not need to configure anything to get TLS working locally or in development. For production, replace the self-signed cert with one from the table above.

The Helm chart includes optional cert-manager integration:

```yaml
certManager:
  enabled: true   # chart creates a Certificate resource and mounts the Secret
```

!!! note "Conversion and webhooks share one certificate"
    `/convert`, `/validate`, and `/mutate` all run on the same HTTPS server on
    `:8443` with the same TLS certificate. One certificate covers all three endpoints.

---

## What environment variables does Orkestra read?

| Variable | Default | Description |
|---|---|---|
| `ORK_PORT` | `8080` | HTTP server port |
| `ENABLE_CONVERSION` | `false` | Enable the `/convert` HTTPS endpoint |
| `ENABLE_ADMISSION_WEBHOOK` | `false` | Enable `/validate` and `/mutate` (requires `ENABLE_CONVERSION`) |
| `TLS_CERT` | — | Path to TLS certificate |
| `TLS_KEY` | — | Path to TLS key |
| `ORK_REGISTRY` | — | Default registry URL for `imports.registry` entries without explicit URL |
| `DEFAULT_WORKERS` | `3` | Worker count per CRD when not set in Katalog |
| `DEFAULT_RESYNC` | `15s` | Resync interval when not set in Katalog |
| `MAX_QUEUE_DEPTH` | `100` | Max queue depth when not set in Katalog |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `NAMESPACE` | — | Namespace where Orkestra runs — used in webhook configurations |
| `ORK_SERVICE_NAME` | `orkestra` | Service name for webhook clientConfig |
| `CONVERSION_WINDOW` | `1000` | Rolling window size for conversion and admission latency percentiles |

---

## What RBAC permissions does Orkestra need?

Orkestra needs a ClusterRole with:

```yaml
rules:
  # Watch and manage every CRD it is configured to handle
  - apiGroups: ["*"]
    resources: ["*"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Leader election
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "create", "update"]

  # Emit Kubernetes events
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]

  # Webhook configuration (when ENABLE_ADMISSION_WEBHOOK=true)
  - apiGroups: ["admissionregistration.k8s.io"]
    resources:
      - validatingwebhookconfigurations
      - mutatingwebhookconfigurations
    verbs: ["get", "create", "update", "patch"]
```

The `["*"]` rule is broad and appropriate for development. For production, scope it to the specific API groups your CRDs use.

**The Helm chart does not manage ClusterRoles.** It deploys the Orkestra runtime (Deployment + Service). To generate the correct RBAC for your specific Katalog, use:

```bash
ork generate bundle --for runtime
```

This produces a scoped ClusterRole, ClusterRoleBinding, and a ConfigMap containing your Katalog — ready to apply to the cluster.

---

## How do I debug a CRD in production?

Use the **Control Center** — it gives you a full view of all CRDs, worker pools, queue depth, reconcile metrics, and dependency health without any additional tooling.

For quick terminal diagnostics, the runtime exposes HTTP endpoints:

```bash
# CRD health — 200 OK or 503 degraded
curl localhost:8080/katalog/website/health | jq

# Full CRD detail — stats, queue depth, active warnings
curl localhost:8080/katalog/website | jq

# All managed CRDs
curl localhost:8080/katalog | jq

# Prometheus metrics
curl localhost:8080/metrics | grep website
```

!!! tip "Port-forwarding in-cluster"
    When Orkestra runs in a cluster, port-forward before hitting the endpoints:
    ```bash
    kubectl port-forward svc/orkestra-runtime 8080:8080 -n orkestra-system
    ```

The most common issues:

| Symptom | Likely cause |
|---|---|
| `/health` returns 503 | CRD degraded — check reconcile error rate in `/katalog/{crd}` |
| Resource not created | `when:` condition not met — check CR fields vs condition |
| Webhook rejection | Validation rule firing — read the error message in `kubectl apply` output |
| Stuck in terminating | `onDelete` Job blocked — check Job status in the CR's namespace |
| Old field values | Reconciler not running — check if CRD is enabled and healthy |

---

## Is Orkestra safe for production?

Yes. Orkestra is designed for and demonstrated in production.

- **Leader election** — only one instance actively reconciles; followers maintain warm caches for instant failover
- **safeReconcile** — panics in any reconciler are caught; other CRDs are unaffected
- **Per-CRD failure domains** — a degraded CRD does not affect others
- **Graceful shutdown** — in-flight reconciles complete before the process exits
- **Conversion in production** — 62 conversions, 0 failures, sub-millisecond latency

!!! note "Failover time"
    Worst-case leader failover is 15 seconds (the lease duration). In practice,
    a follower on a healthy node with a warm cache starts reconciling within
    16–17 seconds of a leader crash. During this window, CRs are not modified —
    they are queued and processed when the new leader starts.

See [Trust and Failure Model](/publications/trust-and-failure-model/) for every
failure mode, what it means, and how Orkestra handles it.

---

## Next

- **[Patterns](./03-patterns.md)** — validation, mutation, built-in kinds
- **[Ecosystem](./04-ecosystem.md)** — comparisons and the Kubernetes roadmap
- **[Deploying](../deploying.md)** — full cluster setup
