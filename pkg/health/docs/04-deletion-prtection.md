# 04 — Deletion protection in Orkestra

Deletion protection in Orkestra is a declarative safety net that ensures critical control‑plane and platform resources cannot be removed accidentally—or maliciously—while the user has explicitly asked for protection.

The Katalog is the single source of truth:

- **`security.deletionProtection.enabled: true`**  
  Orkestra registers a dedicated **deletion‑protection validating webhook**.
- **`security.deletionProtection.enabled: false`**  
  Orkestra removes that webhook and stops intercepting deletions.

When enabled, deletion protection:

- Intercepts `DELETE` operations for:
  - Managed CRDs (via `ProtectedCRDNames()`)
  - The Orkestra Deployment, Service, and Ingress
  - Orkestra’s own admission webhooks:
    - `ValidatingWebhookConfiguration` (`orkestra-validation`)
    - `MutatingWebhookConfiguration` (`orkestra-mutation`)
- Uses an `objectSelector` that matches only Orkestra‑owned resources, so it never interferes with unrelated workloads.
- Blocks deletion of any matched resource—if the webhook fires, ownership is already proven by labels.

Because the admission webhooks and the deletion‑protection webhook itself are labeled as Orkestra‑owned, they are also covered by deletion protection. The webhook controller continuously reconciles all webhook configurations against the Katalog, recreating them if they drift or are removed.

The result is a **self‑protecting, self‑healing admission control plane**: the webhooks that protect the system are themselves protected by the system, and their entire lifecycle is driven declaratively from the Katalog.

---

### Recursive protection loop diagram

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

→ Back to: [README.md](../README.md)
