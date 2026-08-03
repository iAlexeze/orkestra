# Internal Developer Platform

An IDP is not an application. It is a contract.

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

**The CR is the entry point.** Every delivery path — a browser form, `kubectl apply`, a CI pipeline, a Slack bot — ends at the same place: a CR applied to the cluster. Nothing downstream cares how it got there.

Four pieces make that entry point real, and each one is a plain Orkestra component, not a new system:

- **The Katalog is the declaration.** One file: what the CR accepts, what's valid, what gets built when one appears, who's allowed to submit one.
- **The gateway is the validation and admission surface.** Every apply — from any caller — goes through the same schema check, the same `validation.rules`, the same auth tokens, and comes back as a server-side apply on the CR. It's the error handler and the security boundary, in one place, before anything reaches etcd.
- **The runtime is what keeps it in sync, forever.** It watches for CRs and reconciles. One CRD or twenty attached to the same katalog — the reconcile loop doesn't change shape per kind.
- **The Control Center is a client, not a special one.** It calls the gateway's API the same way any other caller does. Anything that can consume an HTTP API — a CLI, a Slack bot, a homegrown React app — can build the same self-service experience against the same contract.

Adding an IDP is enabling the interface to what is already running, not standing up a new system.

```yaml
gateway:
  applyAPI:
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
      idp:
        enabled: true
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

That is the IDP. Two config blocks on the Katalog the platform team already had. The gateway self-bootstraps its token Secret on first start. The Control Center reads `idpEnabled` from the runtime and shows a `[+ Create]` button.

`fields` exposes `spec.*` — the workload data the operator computes with. Not everything a form should collect belongs there, though: a team name, a feature flag, an external ticket reference are metadata, not spec data. `idp.additionalFields` is the release valve — a surface, on any CRD entry in the katalog, for exactly what shouldn't be forced into `spec`. It exposes label and annotation keys as form fields the same way `fields` exposes spec ones, written straight to `metadata` instead.

`order` above looks like form layout, and it is — but it's also validation priority: when more than one field fails at once, only the first violation becomes the headline denial reason, evaluated in the same order the fields render in. Two fields sharing a non-zero `order` is a load-time error for that reason, not just a rendering ambiguity.

There's one more thing a self-service caller shouldn't have to decide: **which namespace**. A CRD is namespaced by default, and someone has to place a new CR *somewhere* — but a developer filling in a form, or a CI job doing a `curl`, has no business making that call. `idp.namespace` is a template expression (`'{{ teamName }}'`, or a plain literal) the gateway resolves server-side against whatever the caller submitted, and it always wins over whatever they sent — so `namespace` never appears on the form and no caller ever needs to supply one. It routes into a namespace the platform team already provisioned; it doesn't create one. A cluster-scoped CRD sidesteps the question entirely a different way — no namespace on the CR at all, with `onCreate` provisioning one as a child resource instead — two answers to the same problem, matched to two different scope choices.

→ [Additional Fields in depth](01-additional-fields.md)
→ [`idp.namespace` reference](../../reference/schema/02-katalog/02-crd-entry.md#idpnamespace)

---

## One CR, any delivery path

The reconciler does not know how a CR arrived. It reconciles when one appears.

```text
Browser form          ↓
kubectl apply         ↓
CI curl POST          ↓  →  CR in Kubernetes  →  runtime reconciles
Slack bot             ↓
GitHub webhook        ↓
```

The gateway Apply API is the uniform interface across all of those:

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/apply` | Create or update a CR |
| `GET /api/v1/resources/{kind}/{ns}/{name}` | Read CR state and status |
| `GET /api/v1/resources/{kind}/{ns}` | List all CRs of a kind |
| `DELETE /api/v1/resources/{kind}/{ns}/{name}` | Delete a CR |
| `GET /api/v1/schema` | Discover the CRD's spec schema as fieldss |
| `GET /api/v1/raw-schema` | Discover the CRD's raw spec schema and field hints |

Every enforcement rule — admission, namespace protection, deletion protection — is the same regardless of delivery path. There is nothing to reconfigure per caller.

---

## What developers see

A developer opens the Control Center. The `Application` row shows a `[+ Create]` button. They fill the form and click Create.

```text
┌────────────────────────────────────────────────────┐
│  Application    3 CRs    ● Healthy    [+ Create]   │
└────────────────────────────────────────────────────┘
```

The form is generated from the CRD's OpenAPI schema combined with the `idp.fields`/`idp.additionalFields` presentation hints. No separate form builder. No schema duplication. The Control Center reads the schema from the gateway and renders it — field types determine input types, enum values become dropdowns, hints appear below the field. It's doing nothing a different client couldn't: the same developer working in CI uses a token and `curl` against the same `POST /api/v1/apply` endpoint instead.

---

## What you do not build

- A separate portal or service catalog
- A Backstage plugin
- A custom admission webhook deployment
- A notification pipeline (the `external:` block in the Katalog handles Jira/Slack after deployment)
- A form schema separate from the CRD schema

A Terraform provider is not one of these — nothing here ships one today. The Apply API is a plain REST surface, so writing one is straightforward, but it is not built for you yet.

All of those are either unnecessary or reduced to configuration.

---

## Try it

```bash
ork init --pack use-cases/idp
```

The pack runs three delivery paths against one `AppRequest` CRD — browser form, CI pipeline, and a Jira + Slack post-deployment hook — then switches to a second CRD, `PlatformResource`, to show two different answers to "one CR, many systems": an entrypoint that infers everything and always creates every tool, and a discriminator that creates exactly one. Each example adds a few lines to the previous one. The runtime is the same throughout.

## Where to go next
→ [Additional Fields](01-additional-fields.md)
→ [Target Mode](02-target-mode.md)

→ [Apply API reference](../../reference/schema/02-katalog/17-katalog-applyapi.md)
