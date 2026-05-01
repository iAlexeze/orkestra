# Orkestra Interactive Tooling – Design Document (Complete)

**Version:** 1.0  
**Status:** Approved  
**Last Updated:** 2026-04-30  

---

## Table of Contents

1. [Overview](#1-overview)  
2. [Guiding Principles](#2-guiding-principles)  
3. [Architecture](#3-architecture)  
4. [Playground Frontend](#4-playground-frontend)  
5. [Notes Browser](#5-notes-browser)  
6. [CLI Commands](#6-cli-commands)  
7. [AI Assistant (`ork ask`) – Future Capability](#7-ai-assistant-ork-ask--future-capability)  
8. [Implementation Plan](#8-implementation-plan)  
9. [Security Considerations](#9-security-considerations)  
10. [Conclusion](#10-conclusion)

---

## 1. Overview

Orkestra’s core innovation is the declarative **Katalog** – a YAML file that fully defines an operator’s behaviour. To make Katalogs accessible to a wider audience, Orkestra provides a suite of interactive tools that lower the barrier to entry, accelerate prototyping, and serve as a bridge between the CLI and the web.

All tools share a common foundation: they reuse the exact Go packages that power the `ork` CLI (`pkg/merger`, `pkg/katalog`, `pkg/note`, `pkg/generate`, etc.). **No logic is duplicated.**

### 1.1 Components

| Component               | Description                                                                                          | Delivery |
|-------------------------|------------------------------------------------------------------------------------------------------|----------|
| **Playground (Web UI)** | Browser‑based Katalog editor with validation, example loading, and notes browser.                   | Embedded in the Control Center (`/playground`) and as a standalone `ork playground` command. |
| **Notes Browser**       | Searchable, filterable reference for all built‑in notes (same data as `ork notes`).                  | Available both in the CLI (`ork notes`) and as a panel inside the Playground. |
| **Explain Command**     | CLI and API to get human‑readable documentation for any Kubernetes field or Katalog construct.      | Future release; backend will be reused in the Playground. |
| **AI Assistant**        | Generate complete operators (declarative or typed) from natural language prompts, with CI integration. | Future release; will leverage the same LLM and validation pipeline. |

---

## 2. Guiding Principles

- **Reuse, don’t rewrite** – All interactive features are thin wrappers around existing Go libraries.  
- **Detachable** – The Playground can run inside the Control Center, as a standalone command, or even as a static Wasm app.  
- **CLI‑web parity** – Every CLI command (`ork notes`, `ork validate`, `ork generate katalog`) has a corresponding web API.  
- **Security‑conscious** – The Playground respects the Control Center’s authentication settings; `ork playground` is localhost‑only by default.  

---

## 3. Architecture

### 3.1 Shared Backend (Go Libraries)

```
┌─────────────────────────────────────────────────────────────┐
│                     Orkestra Go modules                      │
├──────────────┬──────────────┬──────────────┬────────────────┤
│ pkg/merger   │ pkg/katalog  │ pkg/note     │ pkg/generate   │
│ (merge,      │ (validation, │ (notes data) │ (scaffolding)  │
│  parse YAML) │  enrichment) │              │                │
└──────────────┴──────────────┴──────────────┴────────────────┘
                              ▲
                              │ imported by
              ┌───────────────┼───────────────┐
              │               │               │
        ┌─────┴─────┐   ┌─────┴─────┐   ┌─────┴─────┐
        │  Control  │   │   `ork`   │   │ Playground│
        │  Center   │   │   CLI    │   │  (server) │
        └───────────┘   └──────────┘   └───────────┘
```

### 3.2 Playback Modes

- **Embedded mode** – Runs inside the Control Center’s HTTP server (route `/playground`). Shares the same binary and can reuse authentication.  
- **Standalone mode** – `ork playground start` starts a dedicated HTTP server (default port `8081`). Serves the same frontend and API.  

Both modes call the same backend functions.

### 3.3 API Endpoints (Playground)

All endpoints are mounted under `/api/playground/`.

| Endpoint    | Method | Description |
|-------------|--------|-------------|
| `/validate` | POST   | Accepts YAML (Katalog or Komposer), returns validation errors/warnings. |
| `/examples` | GET    | Returns list of packs and examples (from embedded `examples/` directory). |
| `/example`  | GET    | With query `?pack=…&example=…` returns the example `katalog.yaml` content. |
| `/schema`   | GET    | With query `?kind=Katalog` or `?kind=Komposer`, returns a scaffold YAML. |
| `/notes`    | GET    | Returns the same notes metadata as `ork notes` (JSON). |
| `/explain`  | POST   | (Future) Accepts a field path (e.g., `spec.crds.website.workers`) and returns markdown documentation. |

---

## 4. Playground Frontend

### 4.1 Layout

```
+-------------------------------------------------------------------+
|  [Load Example ▼]  [Load Schema ▼]  [Validate]    [Login] (if enabled) |
+--------------------+----------------------------------------------+
|                    |                                              |
|  Code Editor       |  Validation Results / Notes Browser         |
|  (Monaco)          |                                              |
|                    |  - Error list with line numbers             |
|                    |  - Warnings                                  |
|                    |  - Success message ("Ship it!")              |
|                    |                                              |
|                    |  [Notes] tab (collapsible)                   |
|                    |    - Search box, domain filter               |
|                    |    - Click note → insert usage               |
+--------------------+----------------------------------------------+
```

### 4.2 Functionality

- **Editor** – Monaco CDN, YAML mode, line numbers.  
- **Load Examples** – Two‑level dropdown: pack (beginner, advanced…) → example (01‑hello‑website, …). Fetches from `/api/playground/example` and replaces editor content.  
- **Load Schema** – Two options: “Katalog scaffold” and “Komposer scaffold”. Uses `/api/playground/schema` to get a fully‑commented template.  
- **Validate** – Sends editor content to `/api/playground/validate` and displays errors in the right panel.  
- **Notes Browser** – Fetches notes metadata from `/api/playground/notes`. Renders a searchable table with domain filter. Clicking a note inserts its signature as a comment or example into the editor.  

### 4.3 Login Integration

- When `ENABLE_LOGIN=true` (Control Center mode), the Playground shows a **Login** button in the top‑right corner. Clicking redirects to the standard Control Center login page. Validation is still allowed without login; publishing (future) requires authentication.  
- In standalone `ork playground` mode, login is disabled by default (though can be configured via environment variables if needed).  

---

## 5. Notes Browser

The notes data is generated by `make generate-notes` and lives in `pkg/note/catalog_generated.go`. The same data is used by:

- `ork notes` CLI (table, search, show).  
- Control Center Playground (web interface).  

The Playground’s **Notes** tab provides:

- Search by name, description, keywords.  
- Filter by domain (`strings`, `cron`, `kubernetes`, etc.).  
- Click on a note to see full documentation (similar to `ork notes show`) and an **Insert** button that adds the note’s usage example into the editor (at cursor position).  

**This is a zero‑cost feature** – the metadata is already compiled into the binary; the Playground just exposes it via JSON.

---

## 6. CLI Commands

### 6.1 `ork notes` (existing)

Remains unchanged. Provides table, search, show, and domains subcommands.

### 6.2 `ork playground start` (new)

Starts a local HTTP server serving the Playground frontend and API.

**Flags:**

| Flag               | Default | Description |
|--------------------|---------|-------------|
| `--port`           | `8081`  | Listening port. |
| `--no-open`        | `false` | Do not open browser automatically. |
| `--enable-login`   | `false` | Require authentication (uses same credentials as Control Center; only relevant if shared config is available). |

**Example:**
```bash
ork playground start --port 3000
```

The command is excluded from the production runtime binary via `//go:build !runtime`.

### 6.3 `ork explain` (future)

**Subcommands:**

- `ork explain <resource>.<field>` – e.g., `ork explain spec.crds.website.workers`  
- `ork explain note <note-name>` – shows same as `ork notes show`.  
- `ork explain katalog` – prints the Katalog schema documentation.  

**Implementation** will parse the OpenAPI schema of the CRD (for Kubernetes resources) or the internal Katalog schema. For notes, it uses the same metadata as `ork notes`. This command is a natural companion to the Playground’s explain API.

---

## 7. AI Assistant (`ork ask`) – Future Capability

### 7.1 The challenge with AI‑assisted operator development

Traditional operators are complex software projects. AI models struggle to generate correct, production‑ready controller code because they must handle many moving parts: informers, workqueues, leader election, error handling, status updates, and more. The result is often incomplete or subtly broken code that still requires extensive human editing.

### 7.2 How Orkestra changes the game

Orkestra reduces operator development to a **Katalog** – a structured YAML file that fits the pattern AI excels at generating. The runtime logic is completely separate and already proven. Therefore, AI only needs to fill in the blanks of a known schema, not invent an entire program.

`ork ask` leverages this to generate complete, validated operators from natural language prompts – for both declarative (dynamic) and typed (hooks/constructors) operators.

### 7.3 Capabilities

- **Declarative Katalog generation** – From a prompt like *“PostgreSQL operator with StatefulSet, Service, and Secret”*, `ork ask` produces a ready‑to‑use `katalog.yaml`, `crd.yaml`, and `cr.yaml`.  
- **Typed operator generation** – If the user requests a hook or custom constructor, `ork ask` also generates:  
  - Go type definitions (`api/v1alpha1/types.go`)  
  - Hook stubs (`hooks/myhook.go`) or constructor stub (`reconciler/reconciler.go`)  
  - The necessary `go.mod` and `main.go`  
  - Then runs `ork generate registry`, `go build`, and the full E2E test flow (as in the advanced typed examples).  
- **Validated output** – All generated YAML is passed through `pkg/merger` and `pkg/katalog.ValidateConfig` before output, ensuring correctness.  
- **GitHub Action integration** – The `ork ask` command can be run in CI, with options:  
  - `--pr-comment` – Posts the generated files as a comment on the pull request.  
  - `--create-branch` – Creates a feature branch and commits the operator.  
  - `--e2e` – Spins up a Kind cluster, applies the bundle, deploys Orkestra, runs the operator, and verifies resource creation, then posts a summary.  
- **Privacy & cost** – Requires user’s own LLM API key (stored as a secret). Supports both cloud (OpenAI, Anthropic) and local models (Ollama).  

### 7.4 Workflow example

```bash
# Locally
ork ask "Create a MongoDB operator that deploys a StatefulSet with persistence, a Service, and a Secret for credentials." --output ./my-operator

# In CI (GitHub Action)
- name: Generate operator from PR description
  uses: orkspace/orkestra-action@v1
  with:
    ask: ${{ github.event.pull_request.body }}
    pr-comment: true
    e2e: true
    api-key: ${{ secrets.OPENAI_API_KEY }}
```

### 7.5 Why this is transformative

- **Lowest barrier to operator development anywhere** – Describe what you want, get a working operator.  
- **No Go required for 95% of cases** – The declarative path is fully automated.  
- **Typed operators still possible** – AI fills the Go stubs; the complex reconciler logic remains generic.  
- **Instant testing** – The CI integration gives immediate confidence.  

---

## 8. Implementation Plan

### Phase 1 (v1.1) – Playground MVP

- Add `/playground` route and `/api/playground/validate` endpoint to Control Center.  
- Simple frontend (no Monaco yet, just `<textarea>`) with example list from embedded files.  
- Implement validation by calling `pkg/merger.Merge()` and `pkg/katalog.ValidateConfig()`.  
- Reuse existing Control Center assets for styling.  

### Phase 2 (v1.2) – Enhanced Playground

- Upgrade frontend to Monaco Editor with YAML support.  
- Implement Notes browser (calls `/api/playground/notes` and rendered with `fetch`).  
- Add schema scaffolding (`/api/playground/schema`).  
- Introduce `ork playground start` command.  

### Phase 3 (v2.0) – AI Assistant & Explain

- Implement `/api/playground/explain` and `ork explain`.  
- Add `/api/playground/ai/generate` endpoint (stub returning example YAML).  
- Prototype `ork ask` CLI (requires LLM API key).  
- Integrate `ork ask` into GitHub Action with E2E testing option.  

---

## 9. Security Considerations

- The Playground API endpoints do **not** modify the cluster or registry. Validation is read‑only.  
- In embedded mode, the same authentication as the Control Center applies; unauthenticated users can still validate but cannot publish.  
- In standalone mode, the server binds to `127.0.0.1` by default and does not expose to the network unless configured.  
- AI features require user‑provided API keys; keys are never stored by Orkestra (only passed to the LLM provider).  

---

## 10. Conclusion

The Orkestra Playground, Notes Browser, Explain command, and `ork ask` form a cohesive ecosystem that makes Katalog‑based operator development accessible to everyone – from beginners trying out their first operator to experts prototyping complex compositions. By reusing existing Go packages, the implementation cost is low, while the user experience gain is high. Together, they bridge the gap between CLI tooling and web‑based interactivity without compromising Orkestra’s core principle: **declarative power, hidden complexity**.

The inclusion of AI‑assisted generation (`ork ask`) positions Orkestra as a forward‑looking platform that will redefine how operators are created – from writing code to describing intent. This is not just a feature; it’s a paradigm shift for the Kubernetes operator ecosystem. 🚀