# ork uninstall

Remove the Orkestra CLI, Control Center binary, local cache, and shell completions.

```bash
ork uninstall [flags]
```

---

## Flags

| Flag | Description |
|------|-------------|
| `-y, --yes` | Skip the confirmation prompt |
| `--dry-run` | Show what would be removed without deleting anything |

---

## Usage

Interactive uninstall (prompts for confirmation):

```bash
ork uninstall
```

Non-interactive (CI or scripted removal):

```bash
ork uninstall --yes
```

Preview what would be removed:

```bash
ork uninstall --dry-run
```

---

## What gets removed

| Path | Contents |
|------|----------|
| `~/.orkestra/bin/ork` | Orkestra CLI binary |
| `~/.orkestra/bin/orkcc` | Control Center binary |
| `~/.orkestra/` | Entire Orkestra directory — cache, config, plugins |
| `~/.bash_completion.d/ork` | Bash completion script |
| `~/.zsh/completions/_ork` | Zsh completion script |
| `~/.config/fish/completions/ork.fish` | Fish completion script |

The install directory can be overridden by setting `ORK_INSTALL_DIR` before running `ork uninstall`.

---

## Notes

- Only paths that exist on disk are removed — missing paths are silently skipped.
- To reinstall after uninstalling, run the install script again.
