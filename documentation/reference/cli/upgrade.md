# ork upgrade

Upgrade the Orkestra CLI to the latest release or to a specific version.

```bash
ork upgrade [flags]
```

This command downloads the appropriate release artifacts for your OS/architecture, installs the updated `ork` binary, and optionally installs the `orkcc` Control Center binary.

---

## Flags

| Flag | Description |
|------|-------------|
| `--version <tag>` | Upgrade to a specific version (e.g., `v1.4.0`) |
| `--runtime-only` | Upgrade only the `ork` CLI (skip installing `orkcc`) |
| `--check` | Check whether a newer version is available |

---

## Usage

Upgrade to the latest version:

```bash
ork upgrade
```

Upgrade to a specific version:

```bash
ork upgrade --version v1.4.0
```

Upgrade only the runtime binary:

```bash
ork upgrade --runtime-only
```

Check for updates without installing:

```bash
ork upgrade --check
```

---

## Behavior

- Detects the current platform (`linux_amd64`, `linux_arm64`, `darwin_amd64`, `darwin_arm64`).
- Resolves the target version:
  - uses `--version` if provided  
  - otherwise fetches the latest release from GitHub
- Downloads the corresponding tar.gz archive for:
  - `ork`
  - `orkcc` (unless `--runtime-only` is used)
- Extracts the binary from the archive.
- Installs it to the same directory where `ork` was originally installed.
- Prints the installed version after completion.

---

## Upgrade Check

```bash
ork upgrade --check
```

Displays:

- current installed version  
- latest available version  
- whether an upgrade is available  

---

## Notes

- If a binary is not available for the current platform, it is skipped.
- Installation may require elevated permissions depending on the system.
- The upgrade process replaces the existing binaries in place.
