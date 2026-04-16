Here is the **rewritten documentation** for deletion protection, using **only `#`, `##`, `###` headers**, and allowing **bold** inside paragraphs.  
No `**` around headers.  
Clean, publication‑grade, CNCF‑style.

---

# Deletion Protection in Orkestra

Deletion protection is Orkestra’s declarative safety mechanism that prevents accidental or unauthorized removal of critical control‑plane resources. When enabled, Orkestra installs a dedicated validating admission webhook that intercepts `DELETE` operations on Orkestra‑owned resources and blocks them before they reach the Kubernetes API server’s persistence layer.

The feature is controlled entirely through the Katalog:

```yaml
security:
  deletionProtection:
    enabled: true
```

When disabled, Orkestra removes the deletion‑protection webhook and stops intercepting deletions. This ensures the feature is explicit, reversible, and fully declarative.

---

## What Deletion Protection Guards

Deletion protection covers two categories of resources: **platform resources** and **admission webhooks**.

### Platform Resources (Static Control Plane)

These are the core components that allow Orkestra to run:

- Orkestra Deployment  
- Orkestra Service  
- Orkestra Ingress  

These resources carry Orkestra ownership labels:

```yaml
app.kubernetes.io/name: orkestra
app.kubernetes.io/component: orkestra-internal
```

The deletion‑protection webhook uses an `objectSelector` to match these labels, ensuring that only Orkestra‑owned resources are protected and that user workloads are never intercepted.

### Admission Webhooks (Dynamic Control Plane)

Orkestra’s admission surface consists of:

- ValidatingWebhookConfiguration (`orkestra-validation`)
- MutatingWebhookConfiguration (`orkestra-mutation`)
- ValidatingWebhookConfiguration (`orkestra-delete-protection`)

These webhook configurations are also labeled as Orkestra‑owned. This means the deletion‑protection webhook protects the admission webhooks themselves, preventing the admission surface from being disabled by deletion.

This creates a self‑reinforcing safety model: the system that protects the platform is itself protected by the platform.

---

## How Deletion Protection Works

Deletion protection is implemented as a dedicated `ValidatingWebhookConfiguration` that intercepts `DELETE` operations for specific GVRs:

- `customresourcedefinitions`  
- `deployments`  
- `services`  
- `ingresses`  
- `validatingwebhookconfigurations`  
- `mutatingwebhookconfigurations`  

Two strategies are used:

### CRD Protection (Broad Rule)

Intercepts all CRD deletions, but the handler filters to only Orkestra‑managed CRDs via `ProtectedCRDNames()`.

### Orkestra Resource Protection (Scoped Rule)

Intercepts deletions only for resources carrying Orkestra’s ownership labels.  
This ensures:

- No cluster‑wide interception  
- No impact on user workloads  
- No accidental blocking of unrelated controllers  

---

## Two‑Level Protection Model

Deletion protection forms the second layer of Orkestra’s safety model.

### Level 1: Admission Protection  
Validation and mutation webhooks enforce CRD‑level rules and defaults.

### Level 2: Deletion Protection  
A dedicated webhook prevents removal of Orkestra’s control‑plane resources, including the admission webhooks themselves.

Together, these create a **self‑protecting, self‑healing admission control plane**.

---

## Recursive Protection Loop

Below is the conceptual model of how Orkestra protects itself:

```text
                         ┌───────────────────────────────────────────┐
                         │              Katalog (Source of Truth)    │
                         │-------------------------------------------│
                         │  security.webhook.admission.enabled                 │
                         │  security.deletionProtection.enabled      │
                         └───────────────────────────────────────────┘
                                           │
                                           ▼
                         ┌───────────────────────────────────────────┐
                         │        Webhook Controller (Leader)        │
                         │-------------------------------------------│
                         │  • Reconciles admission webhooks          │
                         │  • Reconciles deletion-protection webhook │
                         │  • Ensures specs match Katalog            │
                         │  • Recreates if missing                   │
                         └───────────────────────────────────────────┘
                                           │
                                           ▼
         ┌──────────────────────────────────────────────────────────────────────────┐
         │                         Webhook Configurations                           │
         │──────────────────────────────────────────────────────────────────────────│
         │ 1. ValidatingWebhookConfiguration (orkestra-validation)                  │
         │ 2. MutatingWebhookConfiguration (orkestra-mutation)                      │
         │ 3. ValidatingWebhookConfiguration (orkestra-delete-protection)           │
         │                                                                          │
         │ All carry labels:                                                        │
         │   app.kubernetes.io/name=orkestra                                        │
         │   app.kubernetes.io/component=orkestra-internal                          │
         └──────────────────────────────────────────────────────────────────────────┘
                                           │
                                           ▼
                    ┌──────────────────────────────────────────────────────┐
                    │   Deletion-Protection Webhook (ValidatingWebhook)    │
                    │──────────────────────────────────────────────────────│
                    │  Rules:                                              │
                    │    • CRDs                                            │
                    │    • Deployments                                     │
                    │    • Services                                        │
                    │    • Ingresses                                       │
                    │    • ValidatingWebhookConfigurations                 │
                    │    • MutatingWebhookConfigurations                   │
                    │                                                      │
                    │  ObjectSelector: matches Orkestra labels             │
                    │                                                      │
                    │  → Only protects Orkestra-owned resources            │
                    └──────────────────────────────────────────────────────┘
                                           │
                                           ▼
         ┌──────────────────────────────────────────────────────────────────────────┐
         │                          Recursive Protection                            │
         │──────────────────────────────────────────────────────────────────────────│
         │ 1. Deletion-protection webhook protects admission webhooks               │
         │ 2. Admission webhooks protect CRDs                                       │
         │ 3. Deletion-protection webhook protects itself (it has Orkestra labels)  │
         │ 4. Webhook controller reconciles all of them continuously                │
         │                                                                          │
         │ Result:                                                                  │
         │   A self-protecting, self-healing, self-referential admission plane      │
         │   that cannot be disabled accidentally or maliciously.                   │
         └──────────────────────────────────────────────────────────────────────────┘
```
