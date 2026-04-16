# CHANGELOG

## Added
- **Webhook reconciliation metrics**  
  Introduced Prometheus counters for:
  - `webhook_reconciliations_total`
  - `webhook_reconciliation_failures_total`  
  Enables cluster‑level observability into admission + deletion‑protection webhook lifecycle.

- **Webhook reconciliation stats (UI‑visible)**  
  Added in‑memory `WebhookStats` module mirroring `ConversionStats`, `AdmissionStats`, and `ProtectionStats`.  
  Exposed through the Control Center for real‑time visibility.

- **Recursive deletion protection for all Orkestra‑managed resources**  
  Deletion protection now covers:
  - CRDs managed by Orkestra  
  - Orkestra Deployment, Service, Ingress  
  - **ValidatingWebhookConfiguration (orkestra-validation)**  
  - **MutatingWebhookConfiguration (orkestra-mutation)**  
  - **Deletion‑protection webhook itself**  
  This creates a self‑protecting, self‑healing admission control plane.

- **UI integration for webhook reconciliation stats**  
  Wired `WebhookStats` into the CRD handler so the Control Center displays:
  - Failed Reconciliation
  - Successful Reconciliation

## Changed
- **Webhook controller now records both metrics and stats**  
  Each reconciliation cycle increments:
  - UI stats (`Reconciled`, `Failed`)
  - Prometheus metrics (labeled `"validation"`, `"mutation"`, `"deletion-protection"`, `"controller"`)

- **Admission webhook configurations now carry Orkestra ownership labels**  
  Ensures they are included in deletion protection via `objectSelector`.

## Security / Protection
- **Two‑level protection model implemented**
  1. **Admission protection**  
     Validation + mutation webhooks enforce CRD‑level rules.
  2. **Deletion protection**  
     Dedicated validating webhook prevents deletion of Orkestra‑owned resources, including the admission webhooks themselves.

  This forms a **recursive protection loop**:  
  the system that protects the platform is itself protected by the platform.