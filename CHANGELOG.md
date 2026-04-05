# Changelog

## [Unreleased] — (April 6, 2026)

### **Added**
- Introduced `DependsOnMap` with a unified YAML unmarshalling layer supporting all three dependency formats:
  - **List form:**  
    ```yaml
    dependsOn:
      - database
    ```
  - **Simple map:**  
    ```yaml
    dependsOn:
      database: healthy
    ```
  - **Structured map:**  
    ```yaml
    dependsOn:
      database:
        condition: healthy
    ```
- Added `DependsOnMap.Names()` for deterministic ordering of dependency keys.

### **Changed**
- Migrated `spec.crds` from a list of CRD entries to a **map-based schema**:
  ```yaml
  crds:
    website:
      apiTypes:
        ...
  ```
  The map key is now the authoritative CRD name, injected into `CRDEntry` during load.
- Updated `KatalogSpec` to reflect the new CRD map structure:
  ```go
  CRDs map[string]CRDEntry `yaml:"crds"`
  ```
- Normalized dependency conditions:
  - List entries default to `started`
  - Map entries default to `healthy` when unspecified

### **Improved**
- Simplified merging, validation, and tooling by eliminating CRD name duplication.
- Increased readability and expressiveness of dependency graphs.
- Reduced ambiguity in dependency state handling across the reconciler pipeline.

### **Breaking Changes**
- `spec.crds` is no longer a list; existing Katalogs using the old format must be updated.
- Any tooling or generators relying on the old list structure must be adapted to the new map-based schema.

