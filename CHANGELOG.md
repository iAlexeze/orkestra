# Changelog (Beginner Pack Flow Fixes)

## Fixes & Improvements (Beginner Pack)

**Status patching not working**  
- Added `status` subresource to all example CRDs in the beginner pack  
- Ensures the runtime can patch `.status` fields without API server rejections  
- Fixes reconcile loops that previously failed silently

**Control Center namespace display**  
- Cluster‑scoped resources were showing an empty namespace field  
- Updated the CRD info handler to correctly identify cluster‑scoped CRDs  
- Control Center now displays:  
  ```
  Namespace: cluster-scoped
  ```

**SecretDistribution & ConfigMapDistribution examples**  
- Updated beginner katalogs to use `namespaced: false`  
- Aligns with the actual CRD definitions  
- Fixes apply errors and ensures consistent behavior across packs

**CRD Info Handler improvements**  
- Enhanced the handler that feeds CRD metadata to the Control Center  
- Now reports:  
  - scope (Namespaced / Cluster)  
  - group/version/kind  
  - status subresource availability  
  - schema presence  
- Enables richer UI rendering and more accurate runtime insights

---

## Result

The entire beginner pack flow now works **end‑to‑end**: