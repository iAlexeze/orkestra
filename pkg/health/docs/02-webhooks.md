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

Intercepts DELETE requests for CRDs and the operator's own deployment.

Two-level filtering:
1. **Webhook rule** — broad: intercepts all `customresourcedefinitions` and `deployments` DELETEs.
2. **Handler** — narrow: only denies CRDs that appear in `ProtectedCRDNames()` (this Katalog's CRDs).

A CRD from a different operator or namespace passes through even though the webhook intercepted it.

The webhook is only registered when running inside a cluster. With `ork run` (local mode), there is no reachable Service — the handler would be unreachable and `failurePolicy: Fail` would block all CRD deletions.

→ See `deletion_protection_handler.go`, `pkg/katalog/deletion_protection.go`

## Webhook registration

`RegisterWebhooks` is called in a goroutine after `Start()`:
- Creates or updates `ValidatingWebhookConfiguration` / `MutatingWebhookConfiguration`
- Requires RBAC on `admissionregistration.k8s.io`
- Best-effort: failure is logged but does not block operator startup

`UnregisterWebhooks` is called in `Shutdown()`:
- Removes the webhook configurations
- Ensures the API server stops sending requests to a dead endpoint

→ Next: [03-stats.md](03-stats.md)
