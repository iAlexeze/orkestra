# Changelog

## Added
- Introduced `ork generate registry` (renamed from `runtime`) to better reflect its purpose.
- Added documentation pages for:
  - generate/crd
  - generate/cr
  - generate/registry
  - generate/docs
  - generate/dashboards
  - generate/rbac
  - generate/configmap
  - generate/bundle
  - generate/all
- Added table‑based command index with links for all `generate` subcommands.

## Changed
- Removed deprecated generators: `examples` and `tests` (now handled by `ork init`).
- Update `generate runtime` to `generate registry`.
- Updated `generate all` to run only: registry, docs, dashboards, rbac.
- Updated documentation to use fenced `bash` blocks for all CLI examples.
- Updated dashboards and docs pages to include development admonitions.

## Fixed
- Normalized naming across CLI docs to match actual generator behavior.
- Ensured all command pages follow the same CNCF‑grade structure and formatting.
