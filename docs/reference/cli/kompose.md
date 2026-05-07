# ork kompose

Resolve a `Komposer` file into a fully merged `Katalog`.

```
ork kompose --file <komposer.yaml> [flags]
```

`ork kompose` reads a `komposer.yaml`, validates that it is `kind: Komposer`, resolves all referenced sources, merges them into a single Katalog, validates the result, and prints or writes the merged output.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | Path to `komposer.yaml` (required, exactly one) |
| `-o, --output <file>` | Write merged katalog to file instead of stdout |

---

## Usage

Merge a Komposer and print the merged Katalog:

```
ork kompose --file komposer.yaml
```

Write the merged Katalog to a file:

```
ork kompose --file komposer.yaml --output merged.yaml
```

---

## Behavior

- Validates that the input file is `kind: Komposer`.
- Resolves all sources declared in the Komposer.
- Produces a single merged Katalog with:
  - `apiVersion` preserved from the Komposer
  - `kind: Katalog`
  - `metadata` from the Komposer
  - merged and validated `spec`
- Prunes empty fields before output.
- Writes to stdout unless `--output` is provided.

---

## Notes

- Exactly one `--file` file must be provided.
- The merged Katalog is fully validated and ready for:
  - `ork validate`
  - `ork template`
  - `ork run`