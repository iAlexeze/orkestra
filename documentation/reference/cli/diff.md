# ork diff

Show a unified diff between two files.

```bash
ork diff <file1> <file2>
```

`ork diff` compares two files line‑by‑line and prints a colorized unified diff.  
It works with **any** files, not only Orkestra‑generated artifacts.

This is useful for reviewing changes in:

- generated RBAC files  
- generated CRDs  
- generated Katalogs  
- rendered templates  
- any YAML or text file  

---

## Flags

| Flag | Description |
|------|-------------|
| `--verbose` | Show unchanged lines in the diff output |

---

## Usage

Compare two files:

```bash
ork diff old.yaml new.yaml
```

Verbose mode (includes matching lines):

```bash
ork diff old.yaml new.yaml --verbose
```

---

## Behavior

- Reads both files into memory.
- Splits them into lines.
- Produces a unified diff with:
  - `-` lines in red  
  - `+` lines in green  
  - unchanged lines (only when `--verbose` is used)
- Prints the diff to stdout.

---

## Notes

- Works with any file type (YAML, JSON, text, generated artifacts, etc.).
- Ideal for comparing Orkestra‑generated files across versions or CI runs.
- Does not require Kubernetes access.
