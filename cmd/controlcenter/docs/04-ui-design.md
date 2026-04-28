# 04 — UI Design System

The Control Center UI is entirely server-rendered Go HTML templates with no JavaScript framework. Assets are embedded into the binary via `//go:embed`.

## File layout

```
cc/assets/
  templates/
    index.html       Dashboard landing — katalog cards
    katalog.html     Katalog control panel — CRD grid, health panels
    crd.html         CRD detail — workers, queues, webhooks, events
    cr_list.html     CR instance list for one CRD
    cr_detail.html   Single CR instance with children and events
    docs.html        Documentation landing — katalog + CRD index
    crd_docs.html    CRD documentation page — prose reference
    metrics.html     (placeholder, not currently linked)
  static/
    css/
      style.css      Dashboard design system (used by index/katalog/crd/cr pages)
      ork-docs.css   Documentation design system (used by docs/crd_docs pages)
    js/
      ork-docs.js    Docs: theme toggle, sidebar accordion, scroll-spy TOC
      (other JS files for dashboard interactivity)
    logo.png
```

## Two CSS files, one palette

The dashboard and documentation pages share the same Orkestra brand palette but use separate CSS files because their layout grids are fundamentally different (sidebar+main-panel vs sidebar+content+TOC).

**Do not mix the two.** Dashboard pages (`index.html`, `katalog.html`, `crd.html`, `cr_list.html`, `cr_detail.html`) link only `style.css`. Documentation pages (`docs.html`, `crd_docs.html`) link only `ork-docs.css`.

## Brand palette

All colors are defined as CSS custom properties in `:root` (dark, the default) and `[data-theme="light"]`.

| Variable | Dark value | Role |
|----------|-----------|------|
| `--bg-base` | `#0c0c14` | Page background |
| `--bg-surface` | `#111120` | Cards, panels |
| `--bg-elevated` | `#181828` | Card footers, table headers |
| `--bg-hover` | `#1e1e30` | Hover state backgrounds |
| `--bg-code` | `#0e0e1c` | Code blocks, YAML viewer |
| `--sidebar-bg` | `#08080f` | Sidebar (slightly darker than base) |
| `--accent` | `#7c3aed` | Primary interactive elements |
| `--accent-hover` | `#a78bfa` | Hovered accent elements |
| `--accent-glow` | `rgba(124,58,237,0.15)` | Focus rings, glow effects |
| `--text-primary` | `#e8e8f8` | Body text |
| `--text-secondary` | `#9898b8` | Secondary labels |
| `--text-muted` | `#5a5a78` | Hints, timestamps, placeholders |
| `--color-healthy` | `#34d399` | Healthy state |
| `--color-pending` | `#fbbf24` | Pending state |
| `--color-degraded` | `#f87171` | Degraded/error state |
| `--color-started` | `#60a5fa` | Started/running state |

Each status color also has `-bg` (10% opacity) and `-border` (25% opacity) variants for badge backgrounds and borders.

**Never hardcode hex values** outside of the `:root` / `[data-theme="light"]` blocks. Use `var(--…)` everywhere else.

## Typography

```css
--font-sans: 'Inter', system-ui, -apple-system, 'Segoe UI', sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
```

Inter and JetBrains Mono are loaded from Google Fonts in the `<head>` of each template that uses `ork-docs.css`. `style.css` relies on system font fallbacks — Inter is present if the user's system has it, otherwise `system-ui` takes over.

## Theme toggle

Both CSS files implement dark/light theming via `[data-theme]` on `<html>`. The active theme is persisted to `localStorage` under the key `cc-theme`. Each template has an inline `<script>` that reads this key before the page renders to prevent flash:

```html
<script>(function(){var t=localStorage.getItem('cc-theme')||'dark';
  document.documentElement.setAttribute('data-theme',t);})();</script>
```

## Template functions

Custom Go template functions are registered in `cc/template_func.go` and `cc/cr_template_func.go`. When adding new logic, prefer template functions over embedding Go expressions in HTML — it keeps the templates readable.

## Embedding

All assets are embedded at compile time by the `//go:embed` directive in `cc/controlcenter.go`:

```go
//go:embed assets/templates/*.html assets/static/* assets/static/css/* assets/static/js/*
var Assets embed.FS
```

Adding a new CSS or JS file under `assets/static/css/` or `assets/static/js/` is picked up automatically by the glob. Adding a new template under `assets/templates/` is also automatic. No registration step needed.

→ Next: [05-runtime-manager.md](05-runtime-manager.md)
