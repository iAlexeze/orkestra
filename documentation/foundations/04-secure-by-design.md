# Secure by Design

The typical pattern for security in infrastructure tooling is additive: build the system first, then restrict what it can do. Add a flag to disable the admin endpoint. Strip permissions in the production role. Document which commands are "for development only." The result is a system that is secure if you remember to configure it correctly — and fragile everywhere you forget.

Orkestra inverts this. Security is not a layer added afterward. It is a property of the design — present by default at every layer, from the binaries that ship to the permissions they request to the rules that govern what reaches the cluster.

The goal is simple: the system should be trustworthy by construction, not by configuration.

---

## The binary surface is minimal by construction

Orkestra ships three compiled binaries from a single codebase. Go build tags separate them at compile time — not behind a flag, not at runtime.

| Binary | What it can do |
|--------|---------------|
| `ork` (developer CLI) | Everything — validate, generate, simulate, e2e, template, init, run |
| `ork` (runtime, `//go:build runtime`) | `ork run` only |
| `ork-gateway` (gateway, `//go:build gateway`) | `ork gate` only |

The runtime binary cannot generate RBAC bundles. It cannot scaffold operators. It cannot enumerate registered CRDs or exfiltrate Katalog definitions. That code does not exist in the binary.

This is a structural guarantee, not a permissions check. A permissions check can be misconfigured. A compile-time exclusion cannot.

The developer CLI is intentionally feature-complete — `ork validate`, `ork simulate`, `ork e2e`, `ork generate`, `ork template` are all available locally. Nothing is held back from the engineer writing and testing patterns. What is held back is from the production process running them.

---

## Two trust domains that cannot cross

The Runtime and Gateway run as two separate processes with separate Kubernetes ServiceAccounts and separate ClusterRoles. Neither carries the permissions of the other.

The **Runtime** reconciles custom resources. It reads CRs, applies templates, manages the resources declared in `onCreate` and `onReconcile` blocks, and emits events. It has no permissions to touch webhook configurations or TLS certificates.

The **Gateway** serves admission webhooks — validation, mutation, deletion protection, and version conversion. It manages TLS automatically. It has no permissions to reconcile CRs or manage the resources your operator controls.

A compromise of the Runtime cannot touch webhook infrastructure. A compromise of the Gateway cannot touch your CRs. The blast radius of either failure is bounded by what that process was ever permitted to do.

When a feature is disabled in the Katalog, the Gateway removes the corresponding webhook configuration — the security surface shrinks automatically to match the declared intent.

---

## CRD isolation: each operator runs in its own cell

Each CRD declared in a Katalog runs inside its own **OperatorBox** — an isolated runtime cell with its own informer, event queue, worker pool, health state, and reconciler instance. Nothing is shared.

A panic in one reconciler — any unrecovered Go panic — is caught, logged with the full stack trace, and requeued with backoff. The affected OperatorBox retries. Every other OperatorBox continues uninterrupted.

Queue pressure in one OperatorBox does not affect reconcile latency in another. A misbehaving CRD does not destabilize the rest of the platform.

Communication between OperatorBoxes is always opt-in via a `cross:` declaration in the Katalog. What is not declared does not happen.

---

## RBAC is derived, not authored

Orkestra never auto-creates permissions. Every right your operator has is computed from what you declared in the Katalog and reviewed by you before it reaches the cluster.

```bash
ork validate --full    # see exactly what permissions will be requested
ork generate bundle    # produce the bundle containing those permissions
kubectl apply -f bundle.yaml
```

The generated bundle contains three separate ClusterRoles — one for the Runtime, one for the Gateway, one for the Control Center. They do not overlap. Gateway permissions are only generated if the features that require them are declared: no validation rules means no `admissionregistration.k8s.io` entries in the bundle at all.

Traditional operators often ship with:

```yaml
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

The Orkestra bundle contains:

```yaml
- apiGroups: ["platform.orkestra.io"]
  resources: ["websites"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["platform.orkestra.io"]
  resources: ["websites/status"]
  verbs: ["get", "update", "patch"]
```

Only the API groups you declared. Only the resource kinds those groups produce. Only the verbs those resources need. Built-in resources (Deployments, Services, ConfigMaps) only appear if your Katalog actually uses them.

The bundle diffs cleanly in GitOps workflows. Every change to the Katalog produces a visible, reviewable diff in the bundle before it reaches the cluster.

---

## The containers are hardened by default

Both production binaries run in a distroless base image: no shell, no package manager, no curl, no tar, no standard Unix utilities. An attacker who gains code execution in the container has a single static binary and nothing to pivot with.

The Helm chart applies a hardened security context to every pod by default:

```yaml
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  capabilities:
    drop: ["ALL"]
seccompProfile:
  type: RuntimeDefault
```

These are on by default. Nothing needs to be configured to get them. They apply to the runtime, the gateway, and the control center.

The same security profiles are available to workloads you declare in your Katalog:

```yaml
securityContext:
  profile: hardened
podSecurity:
  profile: hardened
```

`hardened` maps to `readOnlyRootFilesystem: true`, `runAsNonRoot: true`, `capabilities.drop: ["ALL"]`, UID 65534. A one-line declaration. No need to repeat the same five fields across every Deployment in the Katalog.

---

## Each layer is designed for the one below it to fail

This is the principle that ties every security property together.

The level-triggered reconciler assumes it will be interrupted — crash, SIGKILL, node failure, all produce a partial state that the next reconcile corrects. The panic recovery in each OperatorBox assumes individual reconciles will fail. The isolated worker pools assume one CRD's failure will happen. The leader election assumes the entire process will crash. The admission layer assumes the reconciler may not see every CR. The production binary assumes someone might try to misuse whatever surface is exposed.

**Each layer is designed for the one below it to fail, and to remain correct when it does.**

The result is a system where trustworthy behavior does not depend on everything going right. It depends on the guarantees holding even when things go wrong.

---

## Validation runs at five independent points

Security rules declared in a Katalog are not enforced once. They are enforced at every layer where enforcement is possible:

```text
1. Parse time         — strict YAML: unknown fields are errors, not silent defaults
2. ork validate       — offline: schema, templates, dependency graph, namespace rules
3. ork simulate       — offline: reconcile loop against in-memory state
4. Admission webhook  — live: Gateway intercepts CREATE/UPDATE before etcd storage
5. Reconcile time     — live: Runtime re-checks every rule on every reconcile cycle
```

A `deny` rule that catches a bad CR at admission time will also catch it if the CR somehow bypasses the webhook. The Runtime enforces rules independently of the Gateway. Both layers must fail independently before a rule is violated.

Each enforcement point assumes the one before it may be absent or imperfect. This is not redundancy — it is how the system stays correct when parts of it fail.

---

## Orkestra's own credentials are never hardcoded

A Katalog is a behavioral contract — a description of what the operator does. It is committed to source control, reviewed in pull requests, distributed as a versioned OCI artifact. Anywhere Orkestra itself needs a credential — to fetch a private source, to send a notification, to pull from a protected registry — the credential is named, not embedded.

The pattern is consistent: the YAML names an environment variable; the runtime resolves it.

**File source authentication** — when a Katalog imports a private file over HTTPS or from GitHub:

```yaml
files:
  - url: https://private.host/platform-policy.yaml
    auth:
      type: bearer
      fromEnv: PLATFORM_TOKEN

  - url: https://github.com/myorg/private-registry
    auth:
      type: github
      fromEnv: GITHUB_TOKEN

  - url: https://internal.host/policy.yaml
    auth:
      type: basic
      usernameFromEnv: REGISTRY_USER
      passwordFromEnv: REGISTRY_PASSWORD
```

**Registry source authentication** — when a Komposer pulls from a private OCI registry or a private Git registry:

```yaml
imports:
  registry:
    - url: registry.myorg.com/operators/postgres@v14-hardened
      oci: true
      auth:
        type: basic
        usernameFromEnv: REGISTRY_USER
        passwordFromEnv: REGISTRY_PASSWORD

    - url: https://github.com/myorg/private-registry
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
```

**Notification credentials** — SMTP credentials (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`) are read from the runtime's process environment at startup. They are never declared in the Katalog YAML. The Slack webhook URL is declared per team in the notification block; it should be treated with the same care as any other credential — injected via the Helm chart's `runtime.env` block rather than committed to source control.

In each case, the YAML file that ships in the OCI artifact contains no credential. The artifact is the same in every environment. Only the environment it runs in differs.

---

## Where this appears across the documentation

- **[Binaries & build tags](../security/01-binaries.md)** — complete command matrix by binary
- **[RBAC](../security/02-rbac.md)** — full bundle contents and `--for` flag for component-scoped generation
- **[Admission webhooks](../security/03-admission.md)** — validation, mutation, and conversion details
- **[Deletion protection](../security/04-deletion-protection.md)** — protecting CRs and Orkestra's own infrastructure
- **[Validation pipeline](../security/06-validation-pipeline.md)** — all five enforcement points in detail
- **[Pod security](../security/07-pod-security.md)** — `baseline`, `restricted`, `hardened` profiles
- **[Trust and Failure Model](../publications/02-trust-and-failure-model.md)** — how the security layers compound
