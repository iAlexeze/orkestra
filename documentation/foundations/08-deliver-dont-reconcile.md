# Deliver, Don't Reconcile

The Gateway delivers. The Runtime reconciles. Neither does the other's job.

This is not an implementation detail. It is a load-bearing principle — the one that lets you compose the Gateway with any reconciliation layer, bring your own controllers when the Runtime is not deployed, and enforce consistent intake rules regardless of how intent arrives.

The Orkestra Runtime is not an operator. It is the infrastructure that reads your Katalog declarations and runs the operators they describe — turning CRD definitions into working controllers. The Gateway knows none of that. It knows only what arrived, whether it was valid, and where it belongs.

---

## Everything passes through the gate

When the Gateway is enabled, it registers admission webhooks with the Kubernetes API server. From that point forward, every CR create and update passes through it — regardless of the source.

`kubectl apply`. A CI pipeline. The Orkestra Gateway API. A GitOps controller. An operator provisioning a child resource. A platform engineer patching a field directly. All of them reach the API server. All of them trigger the webhook. All of them go through the gate.

This is not a soft convention. It is how Kubernetes admission webhooks work. The API server calls the webhook before persisting any resource. There is no path around it — not for `kubectl`, not for CI, not for any controller with cluster access.

The Gateway enforces the same rules for everyone:

- **Namespace restrictions** — the CR must land in a namespace the CRD permits.
- **Validation rules** — field constraints, uniqueness checks, cross-field conditions.
- **Mutation rules** — defaults, label injection, field normalization.
- **Deletion protection** — a resource marked as protected cannot be deleted without explicit intent.
- **Provenance stamping** — every accepted CR carries annotations that record how it arrived.

A developer applying via `kubectl` gets the same validation as a CI pipeline submitting via the Gateway API. The gate does not ask where you came from. It asks whether you are valid.

---

## What delivery means

Delivery is the act of taking intent from a caller and placing a valid CR in the cluster. The Gateway owns this completely:

1. The CR arrives at the API server.
2. The webhook fires. The Gateway validates, mutates if configured, and either admits or rejects.
3. If admitted, the CR is persisted. Provenance annotations are stamped.
4. The Gateway returns its verdict to the caller.

That is where the Gateway's responsibility ends. It has no interest in what happens next. It does not watch the CR after admission. It does not know whether a controller picks it up, whether reconciliation succeeds or fails, or whether the CR is ever acted on at all.

The Gateway delivered the intent. Its job is done.

---

## What reconciliation means

Reconciliation is the act of making the cluster state match what the CR declares. When the Orkestra Runtime is deployed, it reads the CRD declarations in your Katalog and runs the operators they describe — watching for CRs of the right kind and acting on them. When the Runtime is not deployed, something else does that: Argo CD, Crossplane, Flux, or a controller you wrote.

Whatever is doing the reconciling, it is not the Gateway.

The reconciler does not know how the CR was delivered. It does not know whether it came through `kubectl apply`, the Gateway API, or a CI pipeline. It does not know which token was used, which alias was named, or which validation rules ran. By the time the CR reaches a watch loop, all of that context has been resolved to provenance annotations on the CR — and what the reconciler does with those annotations, if anything, is entirely up to it.

The reconciler owns the outcome. The Gateway owned the intake. Neither needs to understand the other.

---

## Bring your own reconciler

Because delivery and reconciliation are separate, the Gateway composes with any reconciliation layer.

When the Orkestra Runtime is present, it turns your Katalog declarations into running operators and handles the full reconciliation lifecycle. When it is not — because you are adopting the Gateway incrementally, or because your CRDs are already handled by Argo CD, Crossplane, Flux, or a controller of your own — the Gateway still does its job. It does not care what reconciles the CR, or whether anything does.

This means you can adopt the Gateway's intake layer without changing your reconciliation stack. Teams that already have controllers keep them. They get token-gated access control, provenance annotations, alias-scoped response shaping, and universal validation enforcement — layered onto whatever is already running.

The delivery contract is independent of the reconciliation contract. That is what makes it composable.

---

## The runtime adds a second enforcement pass

When the Orkestra Runtime is deployed alongside the Gateway, the same admission rules declared in the Katalog are enforced again at reconcile time — before the Runtime creates any child resource, on every cycle.

The Runtime does this unconditionally. It does not know whether the Gateway ran at admission time. It does not check. It enforces because the rules are its rules to enforce.

This gives two independent enforcement passes. If the Gateway was unavailable when a CR was applied, the Runtime catches the violation on the first reconcile. If the Runtime restarts mid-cycle, the Gateway already caught it at admission. A CR that passes one layer runs into the other.

But the Runtime is optional. The Gateway enforces at admission whether or not a Runtime is present. You can deploy the Gateway against a cluster where the Runtime is not installed, against a cluster where a different operator entirely handles reconciliation, and the enforcement holds.

---

## Why this boundary matters

Mixing delivery and reconciliation into the same component is how you end up with a system that cannot be replaced, cannot be composed, and cannot be understood in parts.

A component that both admits intent and reconciles state knows too much. It knows the validation rules and the child resource templates. It knows what the CR means at intake and what it means at runtime. It accumulates coupling — and then it becomes the only path through which anything can happen, because it has captured both sides.

Orkestra draws a hard line. The Gateway knows admission — what is valid, what is permitted, who is allowed. The Runtime knows reconciliation — what the Katalog declares, what resources to create, how to handle drift, when to report status. Neither side reaches across the line.

The result is a system where you can reason about delivery independently of reconciliation, change your reconciler without changing your intake rules, and compose the gateway with infrastructure you did not build and do not control.

---

## The gate is the trust boundary

When the Gateway is enabled, the admission webhook is the trust boundary for your cluster — for every caller, every tool, every controller.

A CR that is in your cluster passed the gate. Its provenance is stamped in its annotations. The namespace it is in was validated. The fields in its spec satisfied every rule that was declared. You know this not because you trust the caller, but because the API server enforced it before accepting the CR.

This is a stronger guarantee than any convention about how developers should apply resources. Conventions are bypassed. Admission webhooks are not.

The delivery boundary is enforced. The reconciliation boundary is yours to own.
