# Merger — Developer Documentation

This directory explains how the `pkg/merger` package resolves and merges Katalog and Komposer YAML files into the unified CRD map the operator boots from.

## Documents

| File | What it covers |
|------|----------------|
| [01-architecture.md](01-architecture.md) | The full merge pipeline from entry-point files to a ready `*Merger` |
| [02-kinds.md](02-kinds.md) | Katalog vs Komposer — what each kind can and cannot declare |
| [03-sources.md](03-sources.md) | Source types (file, Helm, registry) — resolution order and auth |
| [04-deduplication.md](04-deduplication.md) | How duplicate CRD names are detected and why they are errors |
| [05-top-level-accumulation.md](05-top-level-accumulation.md) | How security, notification, and providers are merged across sources |

Read them in order the first time. For a quick reference when adding a new source type, jump to [03-sources.md](03-sources.md).
