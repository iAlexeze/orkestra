## Changelog – Orkestra v0.2.9

### ✨ New `ork generate katalog` – scaffold a Katalog in seconds

Scaffold a production‑ready `katalog.yaml` with sensible defaults, optional typed‑mode placeholders, and built‑in security, notification, and provider blocks. No more memorising the schema.

**Flags:**
- `--add-hook` – typed mode with a `hooks` declaration (comment)
- `--add-constructor` – typed mode with a `constructor` declaration (`default: false`)
- `--typed` – both hook and constructor sections commented; you choose one
- `--add-security` – add namespace & deletion protection stubs
- `--add-notification` – add Slack/email notification example
- `--add-provider <aws|azure|gcp>` – add cloud provider configuration

**Example:**
```bash
ork generate katalog --add-hook --add-security --add-provider aws -o my-katalog.yaml
```

[Read the full command reference](https://docs.orkestra.io/reference/cli/generate-katalog)

---

### 🚀 Complete CI/CD for typed operators (hooks & constructors)

Two new E2E workflows now run in GitHub Actions for the **advanced pack**:

- **09-hooks** – typed hooks for a `Database` CRD (StatefulSet + Service + optional CronJob)
- **10-constructors** – custom constructor for a `Pipeline` CRD (state machine with Jobs)

Both workflows:
- Generate the typed registry (`ork generate registry`)
- Show the expected validation failure with the standard `ork` binary
- Build a custom `ork` binary that includes the user’s Go code
- Build, tag (with `hooks-` or `constructor-` prefix), and push a container image to `ghcr.io/orkspace/orkestra-typed-extensions`
- Deploy the image via Helm, apply the CR, and verify resource creation
- Test cleanup via owner reference garbage collection

These workflows prove that typed operators are **fully automatable** – from `git push` to a running cluster – using the same Orkestra GitHub Action that works for dynamic operators.

---

### 🔧 Action improvements

- New input `generate-registry` – runs `ork generate registry` after `init`
- New output `registry_file` – path to the generated registry (for inspection)
- `namespace` input now defaults to `orkestra-system` and is passed to `generate configmap` and `generate bundle`
- Support for custom `image_repo` and `image_tag` in typed E2E workflows

---

### 🐛 Fixes

- `mode:` is now automatically inferred when `apiTypes.location` is set (no need to write `mode: typed` manually)
- Registry generation no longer requires `init=true` – works with any existing Katalog
- The stub `pkg/runtime/zz_generated_runtime_registry.go` now includes structured debug logging (`logger.Debug()`) to help diagnose registration issues

---

### 📖 Documentation

- New command reference for `ork generate katalog`
- Updated typed extensions guide (`09-hooks` and `10-constructors`) with step‑by‑step instructions and the full E2E workflow

---

### Upgrading

No upgrade required if you’re using `ork generate bundle` or `ork run`. For typed operators, simply regenerate your registry file with the new `ork generate registry` (the output format has not changed).
