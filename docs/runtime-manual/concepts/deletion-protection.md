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
app.kubernetes.io/tag: orkestra-internal
```

The deletion‑protection webhook uses an `objectSelector` to match these labels, ensuring that only Orkestra‑owned resources are protected and that user workloads are never intercepted.

### Admission Webhooks (Dynamic Control Plane)

Orkestra’s admission surface consists of:

- ValidatingWebhookConfiguration (`orkestra-validation`)
- MutatingWebhookConfiguration (`orkestra-mutation`)

These webhook configurations are also labeled as Orkestra‑owned.  
Deletion protection prevents these admission webhooks from being deleted, ensuring that Orkestra’s validation and mutation logic cannot be disabled by removing their webhook configurations.

### The Deletion‑Protection Webhook Itself

The deletion‑protection webhook is continuously reconciled by the Orkestra controller.  
If it is removed, the controller immediately recreates it to match the desired state defined in the Katalog.

Kubernetes does not invoke admission webhooks when deleting admission webhook configurations, so the deletion‑protection webhook cannot block deletion of itself. Instead, Orkestra guarantees its existence through **controller‑level reconciliation**, which is the strongest protection model Kubernetes allows.

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
A dedicated webhook prevents removal of Orkestra’s control‑plane resources, including Orkestra’s admission webhooks.

Together, these create a **self‑healing admission control plane** that cannot be disabled accidentally or silently bypassed.

---

## Protection and Reconciliation Model

Below is the conceptual model of how Orkestra maintains a resilient admission surface:

```text
                         ┌───────────────────────────────────────────┐
                         │              Katalog (Source of Truth)    │
                         │-------------------------------------------│
                         │  security.webhook.admission.enabled       │
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
         │   app.kubernetes.io/tag=orkestra-internal                                │
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
         │                          Protection Guarantees                           │
         │──────────────────────────────────────────────────────────────────────────│
         │ 1. Admission webhooks cannot be deleted while protection is enabled      │
         │ 2. CRDs and platform resources cannot be deleted                         │
         │ 3. The deletion-protection webhook is continuously reconciled            │
         │ 4. The admission surface cannot be disabled without changing the Katalog │
         │                                                                          │
         │ Result:                                                                  │
         │   A self-healing, tamper-resistant admission plane aligned with          │
         │   Kubernetes controller best practices.                                  │
         └──────────────────────────────────────────────────────────────────────────┘
```
