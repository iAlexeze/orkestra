# ork notes

Browse and search the built-in Orkestra note functions without leaving the terminal.

Notes are the template functions available in every Katalog expression — `status.fields`, `mutation`, `validation`, `onCreate`, `onReconcile`, and anywhere else `{{ }}` is evaluated.

```bash
ork notes [--domain <domain>] [--no-pager]
ork notes search <term>
ork notes show <name>
ork notes domains
```

> **Dev binary only.** `ork notes` is not included in the production runtime image (built with `-tags runtime`). It is available in the full `ork` binary used during development.

---

## Subcommands

### `ork notes`

List all 115 built-in notes as a paginated table.

```text
DOMAIN        NAME              DESCRIPTION
──────        ────              ───────────
collections   asList            Convert input to []interface{}.
collections   asMap             Convert input to map[string]interface{}.
cron          cronToMap         Convert a cron string to the structured map shape.
kubernetes    resourceExists    Return true when the object exists in the template context.
strings       toLower           Convert a string to all lowercase.
...
```

Output is piped through `less` when stdout is a terminal. Use `--no-pager` to print directly.

### `ork notes --domain <domain>`

Filter notes by domain.

```bash
ork notes --domain kubernetes
ork notes --domain cron --no-pager
```

### `ork notes search <term>`

Full-text search across note names, descriptions, and examples. Case-insensitive.

```bash
ork notes search replica
ork notes search "cron expression"
```

### `ork notes show <name>`

Show full documentation for a single note: domain, description, example.

```bash
ork notes show cronToMap
ork notes show replicasReady
ork notes show resourceExists
```

Output:

```text
────────────────────────────────────────────────────────────────
  replicasReady
────────────────────────────────────────────────────────────────
  Domain:      replica
  Description: Return true when readyReplicas equals the desired replica count.

  Example:
    # when:
    #   - field: "{{ allReplicasReady.children.deployment }}"
    #     equals: "true"
────────────────────────────────────────────────────────────────
```

### `ork notes domains`

List all available domains with note counts.

```text
Available domains:

  collections       9 notes
  conditional       7 notes
  container         3 notes
  cron             13 notes
  fields            5 notes
  job               3 notes
  kubernetes       17 notes
  math              9 notes
  quantity          4 notes
  random            3 notes
  replica           5 notes
  safeaccess        4 notes
  service           5 notes
  strings          15 notes
  types            13 notes
```

---

## Flags

| Flag | Applies to | Description |
|------|-----------|-------------|
| `--domain <name>` | `ork notes` | Filter by domain |
| `--no-pager` | `ork notes`, `search`, `show` | Print directly without paging through `less` |

---

## Examples

```bash
# List everything
ork notes

# All kubernetes notes
ork notes --domain kubernetes

# Find notes related to rollback or failure
ork notes search rollback

# What does isSynced do?
ork notes show isSynced

# Pipe into grep without pager
ork notes --no-pager | grep replica
```

---

## How the catalog is built

The note documentation is embedded in the binary at build time. The source of truth is the Markdown files in `pkg/note/docs/`. The catalog is regenerated with:

```bash
make generate-notes
```

This re-parses every `### \`noteName\`` heading in the doc files and emits `pkg/note/catalog_generated.go`. The catalog stays in sync with the docs automatically — add a doc entry, run the target, and `ork notes` reflects the change on the next build.
