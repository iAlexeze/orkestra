# **Orkestra Changelog — Canonical Spec, Deletion Protection & Control Center Enhancements**

## **1. CRD Authoritative List & Getter Consolidation**
- Introduced a unified, authoritative CRD registry with strongly‑typed getters.
- Eliminated scattered CRD lookups across the codebase.
- Centralized CRD metadata, schema, and reconciliation configuration under `CRDEntry`.
- Ensured all reconcilers, health endpoints, and the Control Center derive their configuration from a single source of truth.

---

## **2. Declarative Normalize Phase (Conversion Without Webhooks)**
- Added `normalize:` block to `CRDEntry` to support **canonical spec transformation**.
- Normalization now runs **before** mutation, validation, and template rendering.
- Implemented a declarative, YAML‑driven alternative to Kubernetes conversion webhooks:
  - No TLS
  - No webhook deployments
  - No certificate rotation
  - No admission chain complexity
- Normalization supports:
  - Type‑safe field rewriting
  - Multi‑field templating
  - Lossless conversion of structured → string formats (e.g., CronJob schedules)
- Ensured normalized objects propagate through the entire reconcile pipeline.

---

## **3. Bug Fix: Stale Error Surfacing in Status**
- Identified and corrected a long‑standing issue where the first reconcile error persisted in `.status.conditions` even after successful reconciles.
- Ensured the status writer now:
  - Uses the **normalized resolver**
  - Overwrites stale Ready conditions
  - Reflects the **current** health of the resource, not historical failures
- Result: status is now accurate, self‑healing, and aligned with the actual cluster state.

---

## **4. Katalog Security Framework**
Introduced `KatalogSecurity` to unify operator‑level security controls:

### **4.1 Deletion Protection**
- Added `deletionProtection:` block to katalog security configuration.
- Supports:
  - Validating webhook registration
  - Protection of Orkestra CRDs
  - Protection of Orkestra’s own deployment
- Behavior:
  - Enabled by default when block is present
  - Explicit `enabled: false` disables protection
- Added `DeletionProtectionConfig` with clear semantics:
  - `nil` → not declared → disabled
  - `enabled: true` → protection active
  - `enabled: false` → explicitly disabled

### **4.2 RBAC Auto‑Apply**
- Added `rbac:` block with:
  - `enabled`
  - `cleanupOnShutdown`
- Supports ephemeral operators and test environments.

---

## **5. Deletion Protection Metrics & CRD Health Integration**
- Added full metric suite for deletion protection:
  - total
  - success
  - failure
  - latency
- Mirrored structure of validation/mutation metrics.
- Integrated into CRD health endpoint under:

```json
{
  "validation": { ... },
  "mutation": { ... },
  "protection": { ... }
}
```

- Ensured Control Center consumes and displays these metrics.

---

## **6. Control Center Enhancements**
### **6.1 Deletion Protection Indicators**
- Added lock/unlock icons to katalog list and detail views.
- Replaced redundant “operational” label.
- Added tooltips:
  - “Deletion protection enabled”
  - “Deletion protection disabled”

### **6.2 Deletion Protection Stats**
- Added a new stats block in CRD detail view.
- Matches layout and behavior of validation/mutation stats.
- Added compact number formatting (e.g., `5k`, `12k`) with hover tooltips.

### **6.3 YAML Viewer Improvements**
- Removed legacy `<>` icon.
- Added a clear **“View YAML”** button for:
  - Katalogs
  - CRDs
- Uses the existing YAML modal viewer.
- Improves discoverability and UX consistency.

### **6.4 Overflow & Layout Hardening**
- Improved stat container wrapping.
- Ensured large metric sets do not break layout.
- Reused queueDepth tooltip and formatting logic.

---

## **7. Documentation Updates**
Updated documentation across:

### **7.1 Notes / Developer Docs**
- Added explanation of normalize pipeline.
- Added examples of structured → canonical spec conversion.
- Documented resolver lifecycle and status patching behavior.

### **7.2 Health / Metrics Docs**
- Added deletion protection metrics.
- Updated CRD health endpoint schema.
- Added examples of new JSON output.

### **7.3 Reconciler Docs**
- Documented normalize → mutation → validation → reconcile order.
- Added guidance for writing shape‑safe templates.
- Added examples for multi‑version CRDs without webhooks.

---

## **Apache 2.0 License**
- Added Apache 2.0 license to the project, formalizing open-source distribution and usage terms.


# **Summary**
This release introduces:

- A fully declarative, webhook‑free canonicalization pipeline  
- A unified security model with deletion protection  
- Accurate, self‑healing status reporting  
- Richer CRD health metrics  
- A significantly improved Control Center UI  
- Cleaner, more discoverable YAML viewing  
- Stronger documentation and developer ergonomics  