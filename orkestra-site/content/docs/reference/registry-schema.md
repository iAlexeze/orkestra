---
title: "Registry Schema"
weight: 97
---

# Registry Source Schema

Complete schema reference for `sources.registry` entries in a Komposer.

---

## RegistrySource

One entry in `sources.registry`. Pulls a single pattern from a registry,
validates its structure, and loads either `katalog.yaml` or `komposer.yaml`.

```yaml
sources:
  registry:
    - url: <string>              # required
      version: <string>          # optional — ignored when @ present in url
      oci: <bool>                # optional, default: false
      useKomposer: <bool>        # optional, default: false
      auth:                      # optional
        type: <string>
        fromEnv: <string>
        usernameFromEnv: <string>
        passwordFromEnv: <string>
```

---

### `url`

**Type:** `string`  
**Required:** yes

The URL of the registry source. Supports two forms:

**Shorthand — version embedded with `@`:**

```yaml
url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
url: https://github.com/myorg/registry@main
url: https://github.com/myorg/registry@abc123def456
```

The string is split on the last `@`. Everything before is the clean URL.
Everything after is the version. When `@` is present, the `version` field
is ignored.

**Explicit — version as a separate field:**

```yaml
url: ghcr.io/orkestra-sh/orkestra-registry/postgres
version: v14.2.0
```

{{< callout type="note" >}}
Do not include scheme prefixes (`oci://`, `https://` for OCI refs).
Orkestra constructs the correct protocol from the `oci` field and the
URL format.
{{< /callout >}}

    HTTP/HTTPS prefixes are valid for Git sources:
    `https://github.com/myorg/registry` is a Git source.

    For OCI sources, use the registry hostname directly:
    `ghcr.io/myorg/postgres` — not `oci://ghcr.io/myorg/postgres`.

---

### `version`

**Type:** `string`  
**Required:** no  
**Default:** `main` (Git) or `latest` (OCI)

The version, tag, branch, or SHA to pull.

| Source type | Meaning |
|---|---|
| OCI | Artifact tag: `v14`, `v14.2.0`, `latest` |
| Git (GitHub/GitLab) | Branch, tag, or SHA: `main`, `v1.0.0`, `abc123` |
| Git (generic) | Branch, tag, or SHA passed to `git clone --branch` |

{{< callout type="note" title="Precedence" >}}
Ignored when `@` is present in the `url` field. The `@` shorthand always
takes priority.
{{< /callout >}}

---

### `oci`

**Type:** `bool`  
**Required:** no  
**Default:** `false`

Controls the pull mechanism.

| Value | Behaviour |
|---|---|
| `false` | Pull via Git or raw HTTP. GitHub and GitLab use raw file HTTP (one request per file). Other URLs use `git clone`. |
| `true` | Pull as an OCI artifact using the ORAS protocol. Artifact ref is `url:version`. |

{{< callout type="warning" title="ORAS dependency for `oci: true`" >}}
OCI pulls require the `oras` CLI to be installed and available on `PATH`.
Install from [oras.land](https://oras.land/docs/installation). Native
Go library integration will remove this requirement in a future release.
{{< /callout >}}

---

### `useKomposer`

**Type:** `bool`  
**Required:** no  
**Default:** `false`

Controls which source file is loaded from the pattern.

| Value | File loaded |
|---|---|
| `false` | `katalog.yaml` — CRD definitions and reconcile templates |
| `true` | `komposer.yaml` — full upstream operator source tree |

{{< callout type="warning" title="Exactly one file is loaded" >}}
You cannot load both `katalog.yaml` and `komposer.yaml` from a single
registry entry. The field makes the choice explicit and enforced.
{{< /callout >}}

    If the loaded file's kind does not match the expected kind
    (`useKomposer: false` expects `kind: Katalog`, `useKomposer: true`
    expects `kind: Komposer`), Orkestra will error with a kind mismatch
    message.

---

### `auth`

**Type:** `object`  
**Required:** no  
**Default:** unauthenticated

Authentication for the registry. Credentials are resolved from environment
variables at pull time — never from literal values in YAML.

#### `auth.type`

**Type:** `string`  
**Required:** yes (when `auth` is set)

| Value | Description |
|---|---|
| `github` | GitHub personal access token or GitHub App token. Sent as `Authorization: Bearer <token>`. Requires `fromEnv`. |
| `bearer` | Generic bearer token. Sent as `Authorization: Bearer <token>`. Requires `fromEnv`. |
| `basic` | HTTP Basic Authentication. Requires `usernameFromEnv` and `passwordFromEnv`. |

For OCI registries with `oci: true`, `basic` auth is passed to `oras` as
`--username` and `--password`. The `bearer` and `github` types are passed
as `--password` (ORAS accepts tokens via the password flag).

#### `auth.fromEnv`

**Type:** `string`  
**Required:** when `auth.type` is `github` or `bearer`

Name of the environment variable containing the token.

```yaml
auth:
  type: github
  fromEnv: GITHUB_TOKEN    # reads process.env["GITHUB_TOKEN"]
```

#### `auth.usernameFromEnv`

**Type:** `string`  
**Required:** when `auth.type` is `basic`

Name of the environment variable containing the username.

#### `auth.passwordFromEnv`

**Type:** `string`  
**Required:** when `auth.type` is `basic`

Name of the environment variable containing the password.

---

## RequiredFiles

The five files every valid registry pattern must contain, each non-empty:

| File | Purpose |
|---|---|
| `crd.yaml` | The CRD definition(s) to install in the cluster before running the operator |
| `katalog.yaml` | Operator behavior — reconcile templates, conversion rules, workers, resync |
| `komposer.yaml` | Example showing how to import and override this pattern |
| `cr.yaml` | Example custom resource — the minimal CR to test the operator |
| `README.md` | Documentation — field reference, recommended overrides, examples |

{{< callout type="note" >}}
These files are validated after every pull, including during `ork validate`.
Missing or empty files cause an immediate error with a message listing all
violations.
{{< /callout >}}

---

## Full example

```yaml
sources:
  registry:

    # ── Minimal — OCI shorthand, all defaults ─────────────────────────────────
    - url: ghcr.io/orkestra-sh/orkestra-registry/postgres@v14
      oci: true

    # ── Explicit OCI — url and version separate ───────────────────────────────
    - url: ghcr.io/orkestra-sh/orkestra-registry/redis
      version: v7.2.0
      oci: true

    # ── Private OCI registry with basic auth ──────────────────────────────────
    - url: registry.myorg.com/operators/internal-crd@v2.1.0
      oci: true
      auth:
        type: basic
        usernameFromEnv: REGISTRY_USER
        passwordFromEnv: REGISTRY_PASSWORD

    # ── GitHub — raw HTTP, no clone needed ────────────────────────────────────
    - url: https://github.com/myorg/orkestra-registry@v1.0.0

    # ── GitHub private repo ───────────────────────────────────────────────────
    - url: https://github.com/myorg/private-patterns@main
      auth:
        type: github
        fromEnv: GITHUB_TOKEN

    # ── GitLab with bearer token ──────────────────────────────────────────────
    - url: https://gitlab.company.com/platform/operators@production
      auth:
        type: bearer
        fromEnv: GITLAB_TOKEN

    # ── Internal team — load their Komposer exactly as-is ────────────────────
    - url: https://github.com/myorg/canonical-registry@main
      useKomposer: true

    # ── Generic Git server ────────────────────────────────────────────────────
    - url: https://git.myorg.com/platform/operators@main
      auth:
        type: basic
        usernameFromEnv: GIT_USER
        passwordFromEnv: GIT_PASSWORD
```

---

## Error messages reference

| Error | Cause | Resolution |
|---|---|---|
| `failed structure validation: missing: <file>` | Required file absent from pattern | Add the file to the upstream pattern |
| `failed structure validation: empty: <file>` | Required file present but zero bytes | Populate the file in the upstream pattern |
| `invalid URL` | Malformed URL, whitespace, or `oci://` prefix | Remove scheme prefix or fix the URL |
| `reference not found` | Version tag or branch does not exist | Check available versions in the registry |
| `authentication required (401)` | Missing or expired credentials | Set the environment variable and check token expiry |
| `access denied (403)` | Token lacks read permission | Grant read access to the repository or registry |
| `executable file not found: oras` | `oras` not installed, `oci: true` used | Install `oras`: `brew install oras` |
| `useKomposer is false but katalog.yaml contains kind "Komposer"` | Kind mismatch | Set `useKomposer: true` or check pattern structure |
| `useKomposer is true but komposer.yaml contains kind "Katalog"` | Kind mismatch | Remove `useKomposer: true` or check pattern structure |
