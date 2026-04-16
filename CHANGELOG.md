# CHANGELOG

## Repository migrated from personal namespace to the official Orkspace organization.
All module paths, imports, workflows, documentation links, container image references, and installation endpoints have been updated to reflect the new canonical home of the project. This change establishes Orkestra as an organization‑owned platform rather than a personal repository, and prepares the ecosystem for v1 stability, multi‑maintainer governance, and long‑term evolution.

**Key changes:**

- Updated Go module paths from `github.com/ialexeze/orkestra` to `github.com/orkspace/orkestra`.  
- Updated Control Center module path to `github.com/orkspace/orkestra-cc`.  
- Updated all internal imports across the codebase to use the new organization namespace.  
- Updated ldflags targets for both Orkestra and the Control Center to match the new version packages.  
- Updated GitHub Actions workflows to reference the new repository locations, image registries, and release endpoints.  
- Updated installer script to download artifacts from `github.com/orkspace/orkestra`.  
- Updated Helm chart repository, documentation links, and Homebrew tap references to the Orkspace organization.  
- Updated container image names to `ghcr.io/orkspace/orkestra` and `ghcr.io/orkspace/orkestra-cc`.  
- Removed all legacy references to the previous personal namespace.  
- Performed a full repository‑wide refactor affecting over 2,500 lines to ensure consistency, correctness, and future maintainability.

**Impact:**

This is the largest structural change in the project to date.  
It formalizes Orkestra’s identity under the Orkspace organization, aligns all tooling and distribution channels with the new namespace, and sets the foundation for the v1 release and ecosystem growth.
