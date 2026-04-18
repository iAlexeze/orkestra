# HealthServer

`health.HealthServer` is the HTTP(S) server that provides all external interfaces for a running Orkestra instance. It serves health probes, Prometheus metrics, the Katalog API, and — when enabled — the conversion and admission webhook endpoints.

---

## Responsibilities

- `/health` and `/ready` probes for Kubernetes liveness and readiness
- `/katalog` and `/katalog/{crd}` operator health and statistics API
- `/metrics` Prometheus exposition endpoint
- `/convert` CRD version conversion webhook (when `ENABLE_CONVERSION=true`)
- `/validate` admission validation webhook (when `ENABLE_ADMISSION_WEBHOOK=true`)
- `/mutate` admission mutation webhook (when `ENABLE_ADMISSION_WEBHOOK=true`)
- `/deletion-protection` admission webhook for guarding CRD and Orkestra resource deletion (when `security.deletionProtection.enabled: true`)
- `/namespace-protection` admission webhook for enforcing per-CRD namespace allow/restrict rules (when `security.namespaceProtection.enabled: true`)

---

## Two servers, one TLS certificate

HealthServer runs two HTTP servers:

**HTTP server** (`h.server`, default `:8080`)
Serves `/health`, `/ready`, and `/metrics`. No TLS. Intended for internal cluster traffic — Kubernetes probes and Prometheus scraping.

**HTTPS server** (`h.hookSrv`, fixed `:8443`)
Serves `/convert`, `/validate`, and `/mutate`. Requires TLS certificates. Intended for calls from the Kubernetes API server, which requires HTTPS for webhook endpoints.

Both servers share the same `HealthServer` instance and access the same registries. The certificate used for `:8443` is also used as the `caBundle` in the webhook configurations Orkestra registers at startup.

!!! note "ENABLE_CONVERSION and ENABLE_ADMISSION_WEBHOOK share the HTTPS server"
    Orkestra starts a single HTTPS server whenever **either** `ENABLE_CONVERSION=true`
    or `ENABLE_ADMISSION_WEBHOOK=true` is set. Conversion and admission webhooks both run
    behind this shared TLS endpoint.

    - If only conversion is enabled → `/convert` is served over HTTPS.  
    - If only webhooks are enabled → `/validate` and `/mutate` are served.  
    - If both are enabled → all three endpoints are registered on the same server.

    Because this server is always HTTPS, **both features require TLS**:
    whenever `ENABLE_CONVERSION=true` or `ENABLE_ADMISSION_WEBHOOK=true` is set, Orkestra
    expects valid `TLS_CERT` and `TLS_KEY`. If they are missing, startup fails
    fast with an error similar to:

    ```text
    https server error: TLS_CERT is required when conversion or webhooks are enabled
    ```

    Always set `TLS_CERT` and `TLS_KEY` whenever you enable conversion, webhooks,
    or both.
---

## Startup sequence

```go
func (h *HealthServer) Start(ctx context.Context) error
```

1. Validates TLS configuration (if conversion or webhooks enabled)
2. Registers `/health`, `/ready`, `/metrics` on the HTTP mux
3. Starts the HTTP server in a goroutine
4. If `ENABLE_CONVERSION=true`: registers `/convert` on the HTTPS mux, starts HTTPS server
5. If `ENABLE_ADMISSION_WEBHOOK=true`: registers `/validate` and `/mutate` on the HTTPS mux, then registers webhook configurations with the API server in a background goroutine

!!! warning "Webhook registration is best-effort"
    The `RegisterWebhooks` call in step 5 runs in a goroutine and does not block startup.
    If it fails (e.g. insufficient RBAC), the HTTPS endpoints are still reachable but the
    API server does not know to call them. Check logs for:
    ```
    admission webhooks: webhook configuration registration failed
    ```
    This is the most common startup issue when enabling admission webhooks for the first time.

---

## Health and ready probes

**`/health`** — returns 200 when the server has started and considers itself healthy. Returns 503 when `h.healthy` is false. Unhealthy state is set when a critical CRD degrades (if `critical: true` is declared) or when Orkestra calls `h.Unhealthy()`.

**`/ready`** — returns 200 when all CRDs have started and the runtime is accepting reconcile events. Returns 503 until `h.SetReady()` is called, which happens after the dependency graph is fully started.

The distinction matters for rolling deployments. Kubernetes will not route traffic to a pod until `/ready` returns 200. A pod that is healthy but not yet ready — informers syncing, dependency ordering in progress — is kept out of rotation.

---

## The Katalog API

**`GET /katalog`** — returns a JSON array of all managed CRDs with their health state, reconcile statistics, and configuration. Includes `providerCount` per CRD when providers are declared. Used by `ork status`.

**`GET /katalog/{crd}`** — returns the full JSON detail for one CRD. Includes worker count, queue depth, error rate, reconcile totals, conversion stats, admission stats, and (when providers are declared) per-provider error rates. Used by `ork describe` and direct health monitoring.

**`GET /katalog/{crd}/health`** — returns `200 OK` with `{"healthy":true}` or `503` with `{"healthy":false}`. Used by external health checks and `ork status`.

The handler for these endpoints reads from `CRDHealth` structs that are updated by the reconciler workers on every reconcile cycle. The reads are lock-free where possible — atomic operations for counters.

---

## Conversion handler

```go
func (h *HealthServer) conversionHandler(w http.ResponseWriter, r *http.Request)
```

Receives `ConversionReview` from the Kubernetes API server. For each object in the review:

1. Unmarshals the object to `map[string]interface{}`
2. Extracts bare version strings from full apiVersion strings
3. Looks up `ConversionRules` from `h.conversionRegistry` by Kind
4. Finds the `(from, to)` path in the rules
5. Resolves the path spec using `orktmpl.NewResolverFromMap(obj)`
6. Sets `apiVersion` to the target apiVersion string
7. Returns the converted objects in the `ConversionReview` response

See [Versioning](../runtime-manual/concepts/versioning.md) for the full conversion design.

---

## Admission handlers

```go
func (h *HealthServer) validationHandler(w http.ResponseWriter, r *http.Request)
func (h *HealthServer) mutationHandler(w http.ResponseWriter, r *http.Request)
```

Both receive `AdmissionReview` from the Kubernetes API server. The GVR from the request is used to look up rules from `h.admissionRegistry`. Evaluation results feed both the response and the Prometheus metrics.

Key differences from conversion:
- Validation returns `allowed: true/false` in the response
- Mutation returns a JSON patch (RFC 6902) computed from field changes
- Both record to `AdmissionStats` and Prometheus
- Both must never return errors that block the API server — failures are logged and allowed through

---

## Deletion protection handler

```go
func (h *HealthServer) deletionProtectionHandler(w http.ResponseWriter, r *http.Request)
```

Registered at `/deletion-protection` on the HTTPS mux when `security.deletionProtection.enabled: true`. Intercepts `DELETE` on:

- `customresourcedefinitions` — blocks deletion of CRDs managed by this operator
- Orkestra's own deployment, service, ingress, and admission webhook configurations — narrowed by `ObjectSelector`

`failurePolicy: Fail` — if Orkestra is unreachable, the DELETE is blocked, not allowed through. Deletion attempts on protected resources return HTTP 403 with a human-readable message.

Stats are recorded to `ProtectionStats` (exposed at `/katalog/{crd}`) and the `orkestra_deletion_protection_blocked_total` Prometheus counter.

---

## Namespace protection handler

```go
func (h *HealthServer) namespaceProtectionHandler(w http.ResponseWriter, r *http.Request)
```

Registered at `/namespace-protection` on the HTTPS mux when `security.namespaceProtection.enabled: true` **and** at least one CRD declares `allowedNamespaces` or `restrictedNamespaces`.

Intercepts `CREATE` and `UPDATE` on those CRDs. For each request:

1. Looks up the CRD's namespace rules from `h.namespaceRuleMap` (keyed `plural.group`)
2. Evaluates `NamespaceRules.IsNamespaceAllowed(ns)`:
   - If `allowedNamespaces` is set → namespace must be in the list
   - If `restrictedNamespaces` is set → namespace must not be in the list
   - If neither → allow
3. Blocked requests return HTTP 403 with a message pointing to `allowedNamespaces`/`restrictedNamespaces`

`failurePolicy: Fail` (configurable via `security.namespaceProtection.failurePolicy`) — if Orkestra is unreachable, CREATE/UPDATE is blocked. This ensures namespace rules remain enforced even during transient outages.

Stats are recorded to `namespaceStats` (`*ProtectionStats`, same shape as deletion protection) and the `orkestra_namespace_protection_blocked_total` Prometheus counter. Both are exposed at `/katalog/{crd}` under `namespaceProtection`.

The `namespaceRuleMap` is built once in `NewHealthServer` from `katalog.NamespaceProtectionRuleMap()` and is read-only at runtime — rule changes require an operator restart.

### Webhook registration

The namespace protection `ValidatingWebhookConfiguration` (`orkestra-namespace-protection`) is registered at startup (in a background goroutine) and continuously reconciled by the webhook controller. The controller uses `katalog.NamespaceProtectionGVRs()` to build the admission rules — only CRDs with declared namespace rules are included. No webhook is registered when running outside the cluster (`ork run` mode).

---

## ConversionStats, AdmissionStats, ProtectionStats, and ProviderStats

`ConversionStats` and `AdmissionStats` are in-process rolling window trackers. They accumulate statistics in memory using ring buffers. `GetStats()` on each returns a snapshot embedded in the `/katalog/{crd}` JSON response.

`ProtectionStats` (used for both deletion protection and namespace protection) is a simple atomic counter tracking total, blocked, and allowed requests since startup. Two separate instances are maintained — `h.protectionStats` for deletion protection and `h.namespaceStats` for namespace protection. Both are exposed in the `/katalog/{crd}` response under `protection` and `namespaceProtection` respectively.

`ProviderStats` tracks per-provider reconcile and delete call totals and errors since operator startup. `GetSnapshot()` returns one `ProviderStatEntry` per provider that has been called — each entry contains the provider name, total calls, error count, and error rate. Unlike conversion/admission stats, there is no rolling window — provider stats accumulate for the operator's lifetime.

All are not a replacement for Prometheus — they reset on restart. They serve the use case of "what is happening right now in this running instance" rather than "what has happened over the past 30 days."

---

## Environment variables

| Variable | Default | Effect |
|---|---|---|
| `ORK_PORT` | `8080` | HTTP server port |
| `ENABLE_CONVERSION` | `false` | Start HTTPS server and serve `/convert` |
| `ENABLE_ADMISSION_WEBHOOK` | `false` | Serve `/validate` and `/mutate`, register webhook configs |
| `TLS_CERT` | — | Path to TLS certificate (required when conversion or webhooks enabled) |
| `TLS_KEY` | — | Path to TLS key |
| `CONVERSION_WINDOW` | `1000` | Rolling window size for latency percentile calculations |
| `ORKESTRA_SERVICE_NAME` | `orkestra` | Service name used in webhook clientConfig |
| `NAMESPACE` | — | Namespace where Orkestra runs — used in webhook clientConfig |

---

## Adding a new endpoint

1. Add the handler method to `HealthServer`
2. Register in `Start()` on the appropriate mux (`h.mux` for HTTP, `h.hookMux` for HTTPS)
3. Log the registration
4. Add to the endpoint table in this document

!!! tip
    Use `h.logRoutesMiddleware(handler)` when wrapping a handler for debug logging.
    This middleware logs the path, method, and duration of every request when `LOG_LEVEL=debug`.
