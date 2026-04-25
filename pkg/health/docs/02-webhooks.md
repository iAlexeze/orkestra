# 02 — Webhooks

The HTTPS server handles three webhook types. All share TLS configuration; each type is gated by Katalog declarations.

## Conversion webhook — `/convert`

Converts CRs between hub and spoke versions. Registered when the Katalog declares `conversion.paths`.

Request flow:
1. API server sends a `ConversionReview` with a list of objects and a `desiredAPIVersion`.
2. Handler selects the conversion path from `conversionRegistry`.
3. Path applies the declared field mappings, then runs `normalize` if declared.
4. Handler returns `ConversionReview` with all objects converted.

Stats tracked:
- `Total`, `Success`, `Failures`
- `AvgLatencyMs`, `P95LatencyMs` — rolling window, size from `ConversionWindow` config

→ See `conversion.go`, `conversion_stats.go`

## Admission webhooks — `/validate` and `/mutate`

Registered when the Katalog declares `admission.validate` or `admission.mutate` rules.

Both endpoints decode an `AdmissionReview`, evaluate rules against the object, and return an `AdmissionResponse`.

**Validation** (`/validate`):
- Evaluates CEL expressions from `admission.validate.rules`
- Returns `allowed: false` with a human-readable reason on the first failing rule
- All passing → `allowed: true`

**Mutation** (`/mutate`):
- Applies JSON patches from `admission.mutate.patches`
- Returns a base64-encoded JSON patch
- No match → returns `allowed: true` with an empty patch (no-op)

Stats tracked per endpoint:
- `ValidationTotal`, `ValidationAllowed`, `ValidationDenied`, `ValidationWarned`
- `MutationTotal`, `MutationApplied`, `MutationSkipped`
- Latency percentiles: `Avg`, `P95`, `Max`

→ See `admission_handlers.go`, `admission_evaluation.go`, `admission_stats.go`

## Deletion protection webhook — `/deletion-protection`

Intercepts DELETE requests for CRDs and Orkestra's own Kubernetes resources (deployment, service, ingress).

Two webhook entries are registered in a single `ValidatingWebhookConfiguration`:

**Entry 1 — CRD protection** (`protect.crds.orkestra.orkspace.io`):
- Rule: all DELETE on `customresourcedefinitions` (apiextensions.k8s.io/v1)
- No ObjectSelector: the rule is broad; the handler narrows via `ProtectedCRDNames()`
- A CRD from a different operator passes through even though the webhook intercepted it

**Entry 2 — Orkestra resource protection** (`protect.resources.orkestra.orkspace.io`):
- Rules: DELETE on `deployments`, `services`, `ingresses`
- ObjectSelector: `app.kubernetes.io/name: orkestra` + `app.kubernetes.io/tag: orkestra-internal`
- Any resource in the cluster without these labels is never intercepted
- If the webhook fires, the ObjectSelector already confirmed it is ours — handler always blocks

Stats tracked: `Total`, `Blocked`, `Allowed` (cumulative since startup, in-memory via `ProtectionStats`).

The webhook is only registered when running inside a cluster. With `ork run` (local mode), there is no reachable Service — the handler would be unreachable and `failurePolicy: Fail` would block all protected-resource deletions.

→ See `deletion_protection_handler.go`, `protection_stats.go`, `pkg/katalog/deletion_protection.go`

## Webhook registration

`RegisterAdmissionWebhooks` is called in a goroutine after `Start()`:
- Creates or updates `ValidatingWebhookConfiguration` / `MutatingWebhookConfiguration`
- Requires RBAC on `admissionregistration.k8s.io`
- Best-effort: failure is logged but does not block operator startup

`UnregisterAdmissionWebhooks` is called in `Shutdown()`:
- Removes the webhook configurations
- Ensures the API server stops sending requests to a dead endpoint

→ Next: [03-stats.md](03-stats.md)
