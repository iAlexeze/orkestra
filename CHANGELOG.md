# 📝 Changelog — CLI Refactor & Simplification

### [Unreleased] — CLI Modernization & Cleanup

#### Removed
- **Removed the entire `inspect` subsystem** (`get`, `describe`, `status`, `events`, `reconcile`), which is now fully superseded by the Orkestra Control Center (CC).  
  These commands duplicated kubectl functionality and added unnecessary cognitive load.  
  CC now provides richer, aggregated, real‑time introspection, making the inspect package obsolete.

#### Changed
- **Streamlined CLI surface** to focus exclusively on authoring, generation, validation, and platform lifecycle.  
  Orkestra CLI is now intentionally scoped to:
  - `init`
  - `validate`
  - `template`
  - `generate`
  - `kompose`
  - `run`
  - `diff`
  - `control`
  - `upgrade`
  - `version`

- **Shadowed and hid global flags** (`--kubeconfig`, `--katalog`, `--debug`, `--verbose`) for commands that should not inherit them (`init`, `diff`, `upgrade`, etc.).  
  This results in a cleaner, more intuitive help output and prevents irrelevant flags from leaking into non‑cluster commands.

- **Improved `init` UX** by allowing utility flags (`--list-packs`, `--clear-cache`, `--refresh-cache`) to run without requiring a project name.

#### Added
- **Caching system for example packs** under `~/.orkestra/packs/`, enabling instant repeated `ork init` and offline usage.
- **New cache management flags**:
  - `--refresh-cache` — force re-download of example packs  
  - `--clear-cache` — remove all cached packs  
- **Automatic shell completion installation** (bash, zsh, fish) integrated into the installer, with opt‑out via `ORK_SKIP_COMPLETION=true`.
