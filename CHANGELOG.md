# Changelog

## [Unreleased] — Control Center Redesign contd

### Added

#### Control Center
- **Raw/Enriched config viewer** – Click any Katalog name to inspect its configuration. Toggle between "Your Config" (original YAML) and "Orkestra Enriched" (runtime-resolved values with defaults applied).
- **Syntax-highlighted YAML display** – Colors for keys, strings, numbers, booleans, and null values for better readability.
- **Copy to clipboard** – One-click copy of YAML configuration with visual success feedback (green flash).
- **Server-Sent Events (SSE)** – Live page updates without full reloads. Dashboard stats refresh automatically when Katalog data changes.
- **`/api/snapshot` endpoint** – JSON API for partial DOM updates, enabling efficient client-side refreshes.
- **Clickable Katalog names** – Hover effect and inspect icon indicate interactive elements.

### Changed

- **CRD ordering** – CRDs are now sorted alphabetically in Katalog panels, ensuring consistent display across page loads.
- **CR instance ordering** – Custom resources are sorted by name in list views.
- **Theme toggle position** – Restored to correct location in navbar (right side).

### Fixed

- Random CRD order changes on page refresh – now consistently alphabetical.
- Broken breadcrumb link (`Control Panel/a` → `Control Center`).
- Theme toggle button appearing inline instead of right-aligned.

### Improved

- Modal design for config inspection – responsive, scrollable, with clear footer legends.
- Error handling for missing instances – stale data is cleared from UI.
- User feedback for copy actions – visual confirmation on success.
```

## Labels

- `enhancement`
- `feature`
- `control-center`
- `observability`
- `ui`

## Milestone

`v1.0.0`