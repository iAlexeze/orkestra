# Internal Developer Platform

An IDP is not an application. It is a contract.

A platform engineering team defines what developers can create, what rules apply, and what the system produces. Developers use that contract to ship — without knowing what runs underneath. That is the whole thing.

The runtime makes that contract a Custom Resource. Everything else follows from it.

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

The platform team writes one Katalog. That Katalog is the contract. It describes:

- What fields the CR accepts and what values are valid
- What the runtime produces when a CR is applied
- Who can reach it (token-based auth, namespace protection)
- How developers interact with it (form hints, field labels, tab order)

Orkestra already handles reconciliation, admission, and status. Adding an IDP is enabling the interface to what is already running.

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

---

## One CR, any delivery path

The reconciler does not know how a CR arrived. It reconciles when one appears.

```text
Browser form          ↓
kubectl apply         ↓
CI curl POST          ↓  →  CR in Kubernetes  →  runtime reconciles
Terraform provider    ↓
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
| `GET /api/v1/schema/{kind}` | Discover the CRD's spec schema and field hints |

Every enforcement rule — admission, namespace protection, deletion protection — is the same regardless of delivery path. There is nothing to reconfigure per caller.

---

## What developers see

A developer opens the Control Center. The `Application` row shows a `[+ Create]` button. They fill the form and click Create.

```text
┌────────────────────────────────────────────────────┐
│  Application    3 CRs    ● Healthy    [+ Create]   │
└────────────────────────────────────────────────────┘
```

The form is generated from the CRD's OpenAPI schema combined with the `idp.fields` presentation hints. No separate form builder. No schema duplication. The Control Center reads the schema from the gateway and renders it — field types determine input types, enum values become dropdowns, hints appear below the field.

The same developer working in CI uses a token and `curl`. The same developer using Terraform uses the Terraform provider. They are all talking to the same runtime through the same API. The platform team maintains one definition; developers choose how to consume it.

---

## What you do not build

- A separate portal or service catalog
- A Backstage plugin
- A custom admission webhook deployment
- A Terraform provider from scratch (~200 lines of provider code against the Apply API)
- A notification pipeline (the `external:` block in the Katalog handles Jira/Slack after deployment)
- A form schema separate from the CRD schema

All of those are either unnecessary or reduced to configuration.

---

## Try it

```bash
ork init --pack use-cases/idp
```

The example pack runs four delivery paths against the same `AppRequest` CRD: browser form, CI pipeline, Terraform, and a Jira + Slack post-deployment hook. Each example adds a few lines to the previous one. The runtime is the same throughout.

For the ecosystem progression:

```bash
ork init --pack ecosystem-composition/06-idp
```

→ [Apply API reference](../../reference/schema/02-katalog/17-katalog-applyapi.md)
