# 04 — Deletion Protection in Orkestra

Deletion protection in Orkestra is a declarative safety mechanism that ensures critical control‑plane and platform resources cannot be removed accidentally—or maliciously—while the user has explicitly enabled protection.

The Katalog is the single source of truth:

- **`security.deletionProtection.enabled: true`**  
  Orkestra registers a dedicated **deletion‑protection validating webhook**.
- **`security.deletionProtection.enabled: false`**  
  Orkestra removes that webhook and stops intercepting deletions.

When enabled, deletion protection:

- Intercepts `DELETE` operations for:
  - Managed CRDs (via `ProtectedCRDNames()`)
  - The Orkestra Deployment, Service, and Ingress
  - Orkestra’s admission webhooks:
    - `ValidatingWebhookConfiguration` (`orkestra-admission-validation`)
    - `MutatingWebhookConfiguration` (`orkestra-admission-mutation`)
- Uses an `objectSelector` that matches only Orkestra‑owned resources, ensuring it never interferes with unrelated workloads.
- Blocks deletion of any matched resource—if the webhook fires, ownership is already proven by labels.

Because Orkestra’s admission webhooks are labeled as Orkestra‑owned, they are also protected from deletion.  
The deletion‑protection webhook itself cannot be protected by admission (Kubernetes bypasses admission when deleting admission webhooks), but the Orkestra controller continuously reconciles it and **recreates it immediately** if removed.

The result is a **self‑healing admission control plane**: the webhooks that protect the system are themselves maintained by the controller, and their lifecycle is driven declaratively from the Katalog.

---

## Protection and Reconciliation Model

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
                         │        Housekeeper (Leader)        │
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
         │ 1. ValidatingWebhookConfiguration (orkestra-admission-validation)                  │
         │ 2. MutatingWebhookConfiguration (orkestra-admission-mutation)                      │
         │ 3. ValidatingWebhookConfiguration (orkestra-deletion-protection)           │
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
→ Back to: [README.md](../README.md)
