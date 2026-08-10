# Internal Developer Platform

Self-service infrastructure — the serve layer used to give developers a product interface to platform CRDs. An IDP is one of the most common uses of the self-service layer, but the mechanism is general: anything that accepts an HTTP API can be a caller.

An IDP built on Orkestra is not an application. It is a contract.

A platform engineering team defines what developers can create, what rules apply, and what the system produces. Developers use that contract to ship — without knowing what runs underneath. That is the whole thing.

The runtime makes that contract a **Custom Resource**. Everything else follows from it.

---

## Why traditional IDPs are heavy

The standard approach to building an IDP starts from the delivery mechanism — a portal, a plugin, a service catalog — and then tries to wire the operator in afterward.

The result is a stack of systems that each hold a partial view of the same contract:

- The CRD schema lives in a YAML file
- The admission rules live in a webhook deployment
- The UI lives in a Backstage plugin (or a custom React app)
- The CI integration lives in a pipeline step
- The Terraform provider is a separate project someone had to write
- The Jira/Slack hooks live in a notification service or another pipeline step

None of these know about each other. A field rename in the CRD touches all of them. Adding an enum value means updating the schema, the webhook, the form, and the docs. The coordination cost compounds with every change.

---

## The Orkestra Model

Every platform team ends up wanting the same thing: a developer describes what they want, and the right Kubernetes resources appear and stay correct — forever, without anyone watching. Big platforms already prove this pattern works: an ArgoCD `Application`, a Crossplane claim, a Terraform Cloud run are all just a CR that some controller watches and reconciles. Orkestra doesn't invent a new pattern — it generalizes that one to any CRD a platform team defines, instead of tying it to one product's shape.

**The gateway is the entry point. The CR is the internal contract.** A developer, a CI pipeline, and a Slack bot don't apply a CR — they call an API: what's available to create (`GET /api/v1/schema`), what does it need (`GET /api/v1/schema?target=...`), create or update it (`POST /api/v1/apply`), read it back (`GET /api/v1/resources/...`), list it, delete it. The CR is what the gateway builds behind that API to hand to the runtime — an implementation detail the caller never has to see.

Four pieces make that surface real, and each one is a plain Orkestra component, not a new system:

- **The Katalog is the declaration.** One file: what a caller can submit, what's valid, what gets built when it lands, who's allowed to submit it.
- **The gateway is the whole caller-facing surface.** Schema discovery, create/update, read, list, delete, permissions — one API, the same for every caller. It's also the validation and admission boundary: every request goes through the same schema check, the same `validation.rules`, the same auth tokens, before anything reaches etcd. A CR is what it produces internally, not what it hands back to you.
- **The runtime is what keeps it in sync, forever.** It watches for CRs and reconciles. One CRD or twenty attached to the same katalog — the reconcile loop doesn't change shape per kind.
- **The Control Center is a client, not a special one.** It calls the gateway's API the same way any other caller does. Anything that can consume an HTTP API — a CLI, a Slack bot, a homegrown React app — can build the same self-service experience against the same contract.

Adding an IDP is enabling the interface to what is already running, not standing up a new system.

```yaml
gateway:
  api:
    enabled: true
    auth:
      tokens:
        - name: control-center
          secretRef:
            name: ork-apply-token
            key: token
            rotateAfter: 90d
        - name: ci-pipeline
          token: "${CI_ORK_TOKEN}"

spec:
  crds:
    application:
      serve:
        enabled: true
        target: application
        fields:
          environment:
            label: "Environment"
            hint: "Production deployments require platform-team review"
            order: 1
          image:
            label: "Container Image"
            placeholder: "ghcr.io/myorg/myapp:v1.0.0"
            order: 2
```

That is the IDP. Two config blocks on the Katalog the platform team already had. The gateway self-bootstraps its token Secret on first start. The Control Center reads `serveEnabled` and `target` from the runtime and shows a `[+ Create]` button.

`target` — `application` above — is what callers actually say, not `Application` the Kind or `applications.platform.myorg.io` the plural resource. It defaults to the lowercased Kind, so most CRDs need nothing here at all; set it explicitly when the Kind is too verbose or ambiguous for a form label. Every gateway call — schema discovery, apply, read — is keyed by `target`, never by Kind or GVK. → [Target Mode](02-target-mode.md)

`fields` exposes `spec.*` — the workload data the operator computes with. Not everything a form should collect belongs there, though: a team name, a feature flag, an external ticket reference are metadata, not spec data. `serve.labels` and `serve.annotations` close that gap — a surface, on any CRD entry in the katalog, for exactly what shouldn't be forced into `spec`. They expose label and annotation keys as form fields the same way `fields` exposes spec ones, written straight to `metadata` instead.

`order` above looks like form layout, and it is — but it's also validation priority: when more than one field fails at once, only the first violation becomes the headline denial reason, evaluated in the same order the fields render in. Two fields sharing a non-zero `order` is a load-time error for that reason, not just a rendering ambiguity.

There's one more thing a self-service caller shouldn't have to decide: **which namespace**. A CRD is namespaced by default, and someone has to place a new CR *somewhere* — but a developer filling in a form, or a CI job doing a `curl`, has no business making that call. `serve.namespace` is a template expression (`'{{ teamName }}'`, or a plain literal) the gateway resolves server-side against whatever the caller submitted, and it always wins over whatever they sent — so `namespace` never appears on the form and no caller ever needs to supply one. It routes into a namespace the platform team already provisioned; it doesn't create one. A cluster-scoped CRD sidesteps the question entirely a different way — no namespace on the CR at all, with `onCreate` provisioning one as a child resource instead — two answers to the same problem, matched to two different scope choices.

→ [Additional Fields in depth](01-labels-and-annotations.md)
→ [`serve.namespace` reference](../../reference/schema/02-katalog/20-serve.md#servenamespace)

---

## One gateway, any delivery path

A browser form, a CI pipeline, and a Slack bot are the same caller as far as the gateway is concerned — each hits the same endpoints, in whatever order the interaction calls for: discover what's available, submit fields, read status back.

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/schema` | List every target available to this caller |
| `GET /api/v1/schema?target=<t>` | The flat field contract for one target — what to submit, what's required |
| `POST /api/v1/apply` | Create or update — `{"target": "<t>", ...fields}` |
| `GET /api/v1/resources/{target}/{ns}/{name}` | Read status back |
| `GET /api/v1/resources/{target}/{ns}` | List everything of that target |
| `DELETE /api/v1/resources/{target}/{ns}/{name}` | Delete |

None of these mention Kubernetes. A caller that only ever touches this table never learns what a CRD, a GVK, or `spec` even are — those exist one layer down, where the gateway hands a built CR to the runtime. The reconciler, in turn, doesn't know how that CR arrived — a form submit and a `kubectl apply` reconcile identically once the CR exists. Every enforcement rule — admission, namespace protection, deletion protection — is the same regardless of which endpoint got called. There is nothing to reconfigure per caller.

Advanced callers can skip the field contract and submit a full CR directly (`{"apiVersion": ..., "kind": ..., "spec": ...}` instead of `{"target": ...}`) — the gateway detects which shape you're using and both land on the same CR. → [Target Mode](02-target-mode.md) covers the two side by side.

---

## Not every token gets the same answer

`gateway.api.auth.tokens` in the first example declared two tokens — `control-center` and `ci-pipeline` — and said nothing about what either is allowed to do. By default, nothing distinguishes them: any valid token can call any endpoint the Gateway API exposes for a CRD. `serve.tokens`, declared per CRD, closes that gap — which operations, on which endpoints, in which namespaces, per token. A token that leaks in a build log still can't touch production, and can't be used to change what the form itself looks like.

→ [Token Scoping](03-token-scoping.md) — the full model, with a realistic multi-token example
→ [Serve token permissions](../../security/08-serve-permissions.md) — the security write-up: denial responses, what `ork validate` enforces

---

## Multiple surfaces, one CRD

A CRD can expose multiple named entry points — aliases — alongside its primary target. Each alias carries its own token restrictions and response shape, and the CR carries the alias name as a permanent annotation. The operatorBox reads that annotation during every reconcile and can create different child resources depending on which surface delivered the intent.

A `preview` alias creates a lightweight ephemeral environment. An `internal` alias returns the full CR with all provenance fields. The primary target creates the full production stack. Same CR, same operator, different consequences.

→ [Aliases and Intent Provenance](04-aliases-and-provenance.md)

---

## What developers see

A developer opens the Control Center. The `Application` row shows a `[+ Create]` button. They fill the form and click Create.

```text
┌────────────────────────────────────────────────────┐
│  Application    3 CRs    ● Healthy    [+ Create]   │
└────────────────────────────────────────────────────┘
```

The form is generated from `GET /api/v1/schema?target=application` — the same flat field contract any other caller would fetch. No separate form builder. No schema duplication, and no CRD OpenAPI schema leaking into the browser: field types determine input types, enum values become dropdowns, hints appear below the field. It's doing nothing a different client couldn't — the same developer working in CI uses a token and `curl` against the same `POST /api/v1/apply` endpoint instead.

---

## What you do not build

- A separate portal or service catalog
- A Backstage plugin
- A custom admission webhook deployment
- A notification pipeline (the `external:` block in the Katalog handles Jira/Slack after deployment)
- A form schema separate from the gateway's field contract

A Terraform provider is not one of these — nothing here ships one today. The Gateway API is a plain REST surface, so writing one is straightforward, but it is not built for you yet.

All of those are either unnecessary or reduced to configuration.

---

<!-- ## Try it

```bash
ork init --pack use-cases/idp
```

The pack runs three delivery paths against one `AppRequest` CRD — browser form, CI pipeline, and a Jira + Slack post-deployment hook — then switches to a second CRD, `PlatformResource`, to show two different answers to "one CR, many systems": an entrypoint that infers everything and always creates every tool, and a discriminator that creates exactly one. Each example adds a few lines to the previous one. The runtime is the same throughout. -->

## Inspect and play without a cluster

`ork serve` reads the Katalog directly — no gateway running, no cluster needed. Everything you declared is inspectable before you deploy:

```bash
ork serve validate          # validate all serve config
ork serve targets           # list targets and alias counts
ork serve tokens --alias preview   # effective token map for an alias
ork serve response --alias preview # what the alias hands back to callers
ork serve can-i --token ci-pipeline --target application --operation create
```

`ork serve play` goes one step further — it runs the full gateway apply chain in-process from a flat intent file. No gateway, no cluster, no CR applied. The same stages the gateway runs on every POST, locally:

```bash
ork serve play --token control-center
```

`ork serve apply` is the live flip — the same intent file sent to a real gateway. Use play as a CI gate before apply:

```bash
ork serve play -f intent.yaml --token ci-pipeline         # offline — no cluster
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"  # live
```

→ [Local Intent Testing](06-local-intent-testing.md) — the intent file format, the chain stages, and the intent-play loop
→ [Live Delivery](07-live-delivery.md) — ork serve apply, dry run, and the GitOps pattern
→ [CLI reference — ork serve](../../reference/cli/13-serve.md)

---

## Where to go next
→ [Additional Fields](01-labels-and-annotations.md)
→ [Target Mode](02-target-mode.md)
→ [Token Scoping](03-token-scoping.md)
→ [Aliases and Intent Provenance](04-aliases-and-provenance.md)
→ [Field Translation](08-field-translation.md)
→ [Local Intent Testing](06-local-intent-testing.md)
→ [Live Delivery](07-live-delivery.md)

→ [Gateway API reference](../../reference/schema/02-katalog/17-gateway-api.md)
