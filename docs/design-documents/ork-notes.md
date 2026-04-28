# Design Document: `ork notes` CLI – Discoverable Notes Documentation

## Overview

The `ork notes` command makes the notes library discoverable directly from the terminal. Users can list, search, and read detailed documentation for any note without leaving their shell. The CLI becomes self‑documenting, reducing the need to browse web docs for routine tasks.

## Goals

- **Discoverability** – List all notes with brief descriptions and domains.
- **Search** – Find notes by name, description, or domain.
- **Offline** – All documentation is embedded in the binary (no network calls).
- **Synced** – Documentation stays in sync with the code via a `make generate-notes` step.
- **Pagination** – Handle long output gracefully with terminal paging.

## Non‑Goals

- Editing or running notes from the CLI.
- Dynamic documentation fetching from the internet.
- Replacing the main documentation website – this is a complementary interface.

## Data Model

Each note is represented by:

```go
type NoteInfo struct {
    Name        string   // e.g., "cronToMap", "ternary"
    Domain      string   // e.g., "cron", "strings", "kubernetes"
    Description string   // one‑line summary
    Example     string   // short usage example (optional)
    SeeAlso     []string // related note names
    FullDocURL  string   // pointer to docs website (e.g., "/notes/cron#cronToMap")
}
```

The data is stored in a generated Go file: `pkg/note/builtin_notes.go` (created by `make generate-notes`). This file is compiled into the `ork` binary.

## Generation Pipeline (`make generate-notes`)

The source of truth is the existing documentation markdown files in `pkg/note/docs/` (or a central `docs/notes/` directory). Each markdown file contains front matter or structured sections that the generator parses.

**Input:** Markdown files with consistent structure:

```markdown
---
note: cronToMap
domain: cron
description: Convert a cron expression string to a structured map of fields.
see_also: cronFromMap, cronFromAny
---

## Example

`{{ cronToMap "*/5 * * * *" }}` → `{minute:"*/5", hour:"*", ...}`
```

**Output:** `pkg/note/builtin_notes.go` containing a map:

```go
package note

var BuiltinNotes = map[string]NoteInfo{
    "cronToMap": {
        Name:        "cronToMap",
        Domain:      "cron",
        Description: "Convert a cron expression string to a structured map of fields.",
        Example:     "{{ cronToMap \"*/5 * * * *\" }} → map[minute:*/5 hour:* ...]",
        SeeAlso:     []string{"cronFromMap", "cronFromAny"},
        FullDocURL:  "/notes/cron#cronToMap",
    },
    // ...
}
```

If no structured data is present, the generator can fall back to parsing function comments.

## CLI Design

### `ork notes`

Lists all notes in a paginated table:

```
DOMAIN      NAME                 DESCRIPTION
cron        cronToMap            Convert a cron expression string to a structured map of fields.
cron        cronFromMap          Reconstruct a cron expression from a structured map.
strings     camelToKebab         Convert CamelCase to kebab-case.
...
```

Pagination via `less` (or `more` if `less` unavailable). Respects `--no-pager` flag.

### `ork notes --domain <domain>`

Filters notes by domain (e.g., `ork notes --domain cron`).

### `ork notes search <term>`

Full‑text search across `Name`, `Description`, and `Example`. Returns matching notes in the same tabular format.

### `ork notes show <note>`

Displays full documentation for a single note:

```
────────────────────────────────────────────────────────────────
  cronToMap
────────────────────────────────────────────────────────────────
  Domain: cron
  Description:
    Convert a cron expression string to a structured map of fields.
  Example:
    {{ cronToMap "*/5 * * * *" }} → {minute:"*/5", hour:"*", ...}
  See also:
    cronFromMap, cronFromAny
  Full documentation:
    https://docs.orkestra.io/notes/cron#cronToMap
────────────────────────────────────────────────────────────────
```

## Implementation Steps

### 1. Add metadata extraction to `make generate-notes`

- Create a Go script or small CLI tool (e.g., `./hack/generate-notes`) that reads markdown files from `pkg/note/docs/`.
- Parse front matter (YAML) or note headings.
- Emit `pkg/note/builtin_notes.go` with the `NoteInfo` map.

### 2. Integrate generation into the build

```makefile
# Makefile
.PHONY: generate-notes
generate-notes:
	go run ./hack/generate-notes -input ./pkg/note/docs -output ./pkg/note/builtin_notes.go

build: generate-notes
	# ... existing build steps
```

### 3. Implement the CLI commands

- Create `cmd/cli/notes.go` with cobra commands.
- Use `github.com/zyedidia/gopager` or exec `less` directly.
- Implement filtering and search in Go (no need for external search engine).

### 4. Embed the generated data

In `pkg/note/notes_cli.go` (or similar), expose a function:

```go
func GetNoteInfo(name string) (NoteInfo, bool)
func ListNotes() []NoteInfo
func ListByDomain(domain string) []NoteInfo
func SearchNotes(term string) []NoteInfo
```

The `BuiltinNotes` map is used as the backend.

### 5. Pagination

Check if stdout is a terminal:

```go
if terminal.IsTerminal(int(os.Stdout.Fd())) && !noPager {
    pager := exec.Command("less", "-RFX")
    pager.Stdin = strings.NewReader(output)
    pager.Stdout = os.Stdout
    pager.Run()
} else {
    fmt.Print(output)
}
```

### 6. Testing

- Unit tests for note data parsing (if generator is unit‑tested).
- CLI integration tests (using `cobra.Command` test helpers) to verify output formatting.

## Contributor Workflow

When adding a new note:

1. Implement the note function and its unit test.
2. Add documentation in the appropriate `.md` file in `pkg/note/docs/` with front matter (or update existing doc).
3. Run `make generate-notes` to refresh `builtin_notes.go`.
4. Commit both the note and the generated file.
5. The CI pipeline will ensure `make generate-notes` is up‑to‑date.

## Maintenance

- The generator is idempotent – re‑running `make generate-notes` overwrites the output file.
- If a note is removed, its entry in the generated file disappears automatically.
- The markdown files remain the primary source of truth for documentation; the generator just extracts metadata.

## Security & Performance

- No external network calls – all data is embedded.
- The generated map is loaded into memory at `ork` start, which is negligible (a few KB).
- Pagination uses external processes (`less`), which is acceptable for a CLI tool.

## Future Enhancements (Post‑v1)

- **Interactive search** with `fzf` integration (optional flag `--fzf`).
- **Completion** – `ork notes show <tab>` suggests note names.
- **Export** – `ork notes export json` for tooling.
- **Integration with `ork init`** – offer notes examples when generating a Katalog.

## Conclusion

The `ork notes` command makes the richness of the notes library directly accessible from the CLI, with zero friction. By generating the documentation metadata from existing Markdown files, the system stays in sync automatically. This feature is low‑effort, high‑impact and will significantly improve the developer experience for Orkestra users.