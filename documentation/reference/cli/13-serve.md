# ork serve

Inspect, validate, and deliver via Internal Developer Platform (Serve) configurations.

## Commands

| Command | Description |
|---------|-------------|
| [`validate`](#ork-serve-validate) | Validate Serve configuration in a Katalog |
| [`targets`](#ork-serve-targets) | List all Serve targets in a Katalog |
| [`aliases`](#ork-serve-aliases) | List serve aliases for a target or across all CRDs |
| [`schema`](#ork-serve-schema) | Show the flat schema for a Serve target or alias |
| [`fields`](#ork-serve-fields) | List all Serve fields with their paths and types |
| [`tokens`](#ork-serve-tokens) | Show token permissions for a CRD or alias |
| [`response`](#ork-serve-response) | Show the Serve response configuration for a target or alias |
| [`can-i`](#ork-serve-can-i) | Check if a token can perform an operation |
| [`play`](#ork-serve-play) | Run a serve intent locally through the full gateway chain |
| [`apply`](#ork-serve-apply) | Apply an intent or CR to a live gateway |

---

## `ork serve validate`

Validate Serve configuration in a Katalog.

This runs the same serve-specific validations as `ork validate`, but only for Serve concerns: fields, paths, tokens, response config, and namespace rules. It does not check the full Katalog schema — only the Serve portions.

```bash
ork serve validate
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--full` | | Show a detailed breakdown of the Serve configuration for each serve-enabled CRD. |

### Examples

```bash
# Validate Serve configuration
ork serve validate
```

### Output

```text
✓ Serve configuration is valid
```

### With `--full`

```bash
# Validate Serve configuration with detailed breakdown
ork serve validate --full
```

### Output

```text
Serve Configuration Summary
──────────────────────────────────────────────────────────────────────

● application
  target: smartapp  /  kind: AppRequest
  name:      "{{ repoSlug .spec.repository }}"
  namespace: "{{ teamName }}-{{ environmentName }}"
  fields:    12 total (spec: 8, labels: 2, annotations: 2)
  tokens:    4 tokens with restrictions
  aliases:   2
    · preview   (1 token, custom response)
    · internal  (1 token, custom response)
  response:  default: true, payload: 4, exclude: 0
  poll:      field

● database
  target: db  /  kind: Database
  name:      (caller must supply)
  namespace: "{{ teamName }}"
  fields:    none
  tokens:    none (all tokens allowed)
  response:  none (default CR response)

──────────────────────────────────────────────────────────────────────
✓ Serve configuration is valid
  2 Serve-enabled CRD(s)
```
---

## `ork serve schema`

Show the flat schema for an Serve target.

This displays the fields that callers can submit for a target, including their labels, types, enums, and paths. The output is the same flat field structure returned by the gateway's `GET /api/v1/schema?target=<t>` endpoint.

```bash
ork serve schema [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target to show schema for |
| `--kind` | `-k` | Kind to show schema for |
| `--name` | `-n` | CRD name to show schema for |
| `--alias` | `-a` | Alias to show schema for |

One of `--target`, `--kind`, `--name`, or `--alias` is required.

### Examples

```bash
# Show schema by target
ork serve schema --target apifixture

# Show schema by alias
ork serve schema --alias preview

# Show schema by kind
ork serve schema --kind PlatformResource

# Show schema by CRD name
ork serve schema --name platRsc
```

### Output (primary target)

```text
Schema for: platrsc (target: apifixture)
──────────────────────────────────────────────────────────────────────
FIELD           LABEL            TYPE    PATH            REQUIRED
environment     Environment      string  environment
repoURL         Repository URL   string  repoURL
targetNamespace Target Namespace string  targetNamespace
```

### Output (alias)

```text
Schema for: platrsc (alias: preview → target: apifixture)
──────────────────────────────────────────────────────────────────────
FIELD           LABEL            TYPE    PATH            REQUIRED
environment     Environment      string  environment
repoURL         Repository URL   string  repoURL
targetNamespace Target Namespace string  targetNamespace
```

---

## `ork serve fields`

List Serve fields with their paths and types.

This shows fields declared in `serve.fields` and `serve labels/annotations` across all serve-enabled CRDs. With `--target`, `--kind`, or `--name`, shows fields for a specific CRD.

```bash
ork serve fields
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target to show fields for |
| `--kind` | `-k` | Kind to show fields for |
| `--name` | `-n` | CRD name to show fields for |
| `--alias` | `-a` | Alias to show fields for |
| `--sort-by` | | Sort fields by `"name"` (default) or `"order"` |

### Examples

```bash
# List all fields across all CRDs
ork serve fields

# List fields for a specific target
ork serve fields --target apifixture

# List fields for a specific alias
ork serve fields --alias preview

# List fields for a specific kind
ork serve fields --kind PlatformResource

# Sort fields by order (as declared in the Katalog)
ork serve fields --target apifixture --sort-by order
```

### Output (All fields)

```text
Serve Fields
──────────────────────────────────────────────────────────────────────

CRD: application (target: smartapp)
  FIELD           TYPE    PATH                    SOURCE
  cpu             string  app.resources.cpu       spec
  environment     string                          label
  image           string  app.image               spec
  repository      string  app.repository          spec
  team            string                          label

Total: 5 fields across 1 CRD
```

### Output (Specific CRD)

```text
Fields for: application (target: smartapp)
──────────────────────────────────────────────────────────────────────
FIELD           TYPE    PATH                    SOURCE      REQUIRED
cpu             string  app.resources.cpu       spec
environment     string                          label
image           string  app.image               spec        ✓
repository      string  app.repository          spec        ✓
team            string                          label       ✓
```

### Output (Sorted by `order`)

```text
Fields for: application (target: smartapp)
──────────────────────────────────────────────────────────────────────
FIELD           TYPE    PATH                    SOURCE      REQUIRED  ORDER
repository      string  app.repository          spec        ✓         1
image           string  app.image               spec        ✓         2
team            string                          label       ✓         3
cpu             string  app.resources.cpu       spec                  4
environment     string                          label                 5
```
---

## `ork serve tokens`

Show effective token permissions for a serve-enabled CRD or alias.

This displays the `serve.tokens` configuration including which tokens have access, their permissions (`global`/`schema`/`resources`), and namespace restrictions. With `--alias`, shows the effective token map for that alias — its own tokens when declared, or the inherited CRD-level tokens with a source note.

```bash
ork serve tokens [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Primary target to show tokens for |
| `--kind` | `-k` | Kind to show tokens for |
| `--name` | `-n` | CRD name to show tokens for |
| `--alias` | `-a` | Alias to show effective tokens for (can be used instead of `--target`) |

One of `--target`, `--kind`, `--name`, or `--alias` is required.

### Examples

```bash
# Show CRD-level tokens
ork serve tokens --target apifixture

# Show effective tokens for an alias (alias-own tokens or inherited)
ork serve tokens --alias preview

# Show by kind
ork serve tokens --kind PlatformResource
```

### Output (primary target)

```text
Token permissions for CRD: platrsc (target: apifixture)
──────────────────────────────────────────────────────────────────────
TOKEN           GLOBAL  SCHEMA  RESOURCES  NAMESPACES
control-center  *                          *
ci-pipeline             get     get,list   *
```

### Output (alias with own tokens)

```text
Token permissions for alias "preview" (CRD: platrsc)
Source: alias "preview" (own tokens)
──────────────────────────────────────────────────────────────────────
TOKEN           GLOBAL  SCHEMA  RESOURCES  NAMESPACES
control-center  *                          *
```

### Output (alias inheriting CRD tokens)

```text
Token permissions for alias "internal" (CRD: platrsc)
Source: alias "internal" (inherits CRD-level tokens)
──────────────────────────────────────────────────────────────────────
TOKEN           GLOBAL  SCHEMA  RESOURCES  NAMESPACES
control-center  *                          *
ci-pipeline             get     get,list   *
```

---

## `ork serve targets`

List all serve-enabled primary targets in a Katalog.

This shows each primary target, its CRD kind, field count, token restrictions, and alias count. Full alias details are in `ork serve aliases` or `ork serve validate --full`.

```bash
ork serve targets
```

### Examples

```bash
# List all primary targets
ork serve targets
```

### Output

```text
Serve Targets
──────────────────────────────────────────────────────────────────────
TARGET      KIND              FIELDS  TOKENS  ALIASES
apifixture  PlatformResource  12      yes     2
appgroup    AppGroup          0       no      —

2 target(s)
```

---

## `ork serve aliases`

List serve aliases across all serve-enabled CRDs, or for a specific CRD.

Aliases are additional named entry points for a CRD. Each alias can independently override token permissions and response configuration, falling back to CRD-level defaults when not set.

```bash
ork serve aliases [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target to show aliases for |
| `--kind` | `-k` | Kind to show aliases for |
| `--name` | `-n` | CRD name to show aliases for |
| `--alias` | `-a` | Alias name — shows all aliases for the same CRD |

All flags are optional. Without any flag, lists aliases across all serve-enabled CRDs.

### Examples

```bash
# List all aliases across all CRDs
ork serve aliases

# Show aliases for a specific target
ork serve aliases --target apifixture

# Show aliases for a CRD using a known alias name
ork serve aliases --alias preview
```

### Output

```text
Aliases for: platrsc (target: apifixture)
──────────────────────────────────────────────────────────────────────
ALIAS     TOKENS  RESPONSE
preview   yes     yes
internal  no      no
```

---

## `ork serve can-i`

Check if a token can perform an operation.

This evaluates the same permission checks that the gateway applies to incoming Gateway API requests. It considers token existence, permissions (`global`/`schema`/`resources` scopes), namespace restrictions, and target existence.

```bash
ork serve can-i [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--token` | `-T` | Token name to check **(required)** |
| `--target` | `-t` | Target or alias name to check |
| `--alias` | `-a` | Alias to check permissions for (overrides alias resolved from `--target`) |
| `--kind` | `-k` | Kind to check |
| `--name` | `-n` | CRD name to check |
| `--operation` | `-o` | Operation to check **(required)** |
| `--namespace` | `-N` | Namespace to check (default: all namespaces) |
| `--class` | `-c` | Endpoint class to check (`resources`, `schema`) |

One of `--target`, `--kind`, or `--name` is required.

### Examples

```bash
# Check primary target
ork serve can-i --token control-center --target apifixture --operation create

# Check alias — resolves alias-specific token map
ork serve can-i --token control-center --target preview --operation get

# Check alias explicitly (same as passing alias name to --target)
ork serve can-i --token ci-pipeline --target apifixture --alias preview --operation get
```

### Output (Allowed — primary target)

```text
✓ control-center can create on "apifixture"
```

### Output (Allowed — alias)

```text
✓ control-center can get on "apifixture" (alias: preview)
```

### Output (Denied — alias token map excludes this token)

```text
✗ ci-pipeline cannot get on "apifixture" (alias: preview)
  Reason: token "ci-pipeline" is not allowed to access "PlatformResource" — not listed in serve.tokens
```

---

## `ork serve response`

Show the effective Serve response configuration for a target or alias.

This displays what callers will see in the Gateway API response based on `serve.config.response` (or `serve.target[alias].config.response` for aliases). It shows `default: true/false`, payload fields (with their template expressions), excluded paths, and poll URL configuration.

No cluster access is required — this reads the Katalog directly.

```bash
ork serve response [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Primary target to show response for |
| `--kind` | `-k` | Kind to show response for |
| `--name` | `-n` | CRD name to show response for |
| `--alias` | `-a` | Alias to show effective response config for (can be used instead of `--target`) |
| `--preview` | `-p` | Show a sample response preview |

One of `--target`, `--kind`, `--name`, or `--alias` is required.

### Examples

```bash
# Show CRD-level response config
ork serve response --target apifixture

# Show alias-specific response config (falls back to CRD-level when none declared)
ork serve response --alias preview

# Show with preview
ork serve response --target apifixture --preview
```

### Output (primary target)

```text
Response configuration for: platrsc (target: apifixture)
──────────────────────────────────────────────────────────────────────

default: true

Payload fields:
  FIELD   EXPRESSION
  phase   {{ .status.phase }}
  target  {{ getServeTarget . }}
  alias   {{ getServeAlias . }}
  source  {{ getServeSource . }}

Excluded paths: none

Poll URL:
  field: "status.phase"
```

### Output (alias with own response config)

```text
Response configuration for alias "preview" (CRD: platrsc)
──────────────────────────────────────────────────────────────────────

default: false

Payload fields:
  FIELD         EXPRESSION
  phase         {{ .status.phase }}
  workloadType  {{ .spec.workloadType }}
  environment   {{ .spec.environment }}
  target        {{ getServeTarget . }}
  alias         {{ getServeAlias . }}

Excluded paths: none
```

---

## `ork serve play`

Run a serve intent locally through the full gateway chain.

Reads a flat YAML or JSON intent file with the same shape as a target-mode `POST /api/v1/apply` body, runs every stage of the apply chain in-process, and prints a coloured trace of each step.

No cluster connection is required. No CR is applied.

```bash
ork serve play [flags]
```

### Stages

| # | Stage | What happens |
|---|-------|-------------|
| 1 | Target resolution | Resolve `target` to a CRD and alias via the Katalog |
| 2 | Token check | Verify the named token can perform the operation |
| 3 | CR construction | Build the full CR from serve field declarations |
| 4 | Provenance | Stamp `serve-target` and `serve-alias` annotations |
| 5 | Response payload | Evaluate `serve.config.response.payload` expressions |

If any stage fails, the trace stops with a clear error at that stage.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--intent` | `-i` | `intent.yaml` or `intent.json` | Intent file (YAML or JSON) |
| `--token` | `-t` | | Token name to authenticate with **(required)** |
| `--target` | | | Override the `target` field in the intent file |
| `--operation` | `-o` | `create` | Operation to simulate (`create`, `update`, `delete`, `get`, `list`) |
| `--namespace` | `-N` | | Namespace (for `get`/`list`/`delete`) |
| `--name` | `-n` | | Resource name (for `get`/`delete`) |
| `--simulate` | | | Hand the built CR to `ork simulate`; pass a path to run in assert mode |

When `--intent` is not given, `ork serve play` looks for `intent.yaml` first, then `intent.json`, in the current directory.

`--simulate` is only valid for `create` and `update` operations.

### Intent file format

The intent file is a flat key-value document — the same payload you would send as a `POST /api/v1/apply` body. The `target` field is required unless overridden via `--target`.

```yaml
# intent.yaml
target: apifixture
name: my-payment-service
workloadType: app
team: platform
environment: staging
repoURL: https://github.com/myorg/payments
productionApproval: JIRA-1234
```

JSON is also accepted:

```json
{
  "target": "apifixture",
  "name": "my-payment-service",
  "workloadType": "app",
  "team": "platform",
  "environment": "staging",
  "repoURL": "https://github.com/myorg/payments",
  "productionApproval": "JIRA-1234"
}
```

### Examples

```bash
# Play from the default intent.yaml with a token
ork serve play --token control-center

# Play a specific file
ork serve play --intent ./deploy/intent.yaml --token ci-pipeline

# Override the target — useful for testing against a different alias
ork serve play --token control-center --target preview

# Simulate an update instead of create
ork serve play --token control-center --operation update

# Play a JSON intent file
ork serve play --intent intent.json --token control-center

# Play and hand the built CR to simulate (op-print mode)
ork serve play --token control-center --simulate

# Play and assert against a simulate.yaml spec (assert mode)
ork serve play --token control-center --simulate simulate.yaml

# Check what a read-only surface sees (no intent file needed)
ork serve play --token ci-pipeline --target apifixture --operation get --name my-service --namespace team-payments
```

### Output

```text
▶  ork serve play
  intent: intent.yaml
  target: apifixture
  token:  control-center
  op:     create

→  stage 1 · Target resolution
   ✓ kind=PlatformResource  target=apifixture  alias=(none)

→  stage 2 · Token check
   ✓ token control-center can create on PlatformResource

→  stage 3 · CR construction
   ✓ name=my-payment-service  namespace=team-payments
      {
        "apiVersion": "gateway.fixture.orkestra.io/v1alpha1",
        "kind": "PlatformResource",
        "metadata": { ... },
        "spec": {
          "environment": "staging",
          "productionApproval": "JIRA-1234",
          "repoURL": "https://github.com/myorg/payments",
          "team": "platform",
          "workloadType": "app"
        }
      }

→  stage 4 · Provenance annotations
      orkestra.orkspace.io/serve-target: apifixture
   ✓ provenance stamped

→  stage 5 · Response payload
      {
        "alias": "",
        "phase": "",
        "source": "",
        "target": "apifixture"
      }
   ✓ payload evaluated

✓ Intent would be accepted
  POST /api/v1/apply  →  PlatformResource/my-payment-service in team-payments
```

### Why play?

The gateway is an intent runner. The runtime is a CR runner. The gateway collects intentions — flat fields from a caller — and translates them into a valid Kubernetes object. The runtime takes that object and reconciles it into cluster resources. Neither needs a cluster to do its job locally. `ork serve play` runs the gateway's half; `ork simulate` runs the runtime's half.

`ork serve play` is the local equivalent of running a full GitOps webhook delivery cycle. Play lets you verify:

- The intent file builds the CR you expect.
- The token you plan to use is allowed for the target and operation.
- The field routing (spec, labels, annotations) resolves correctly.
- The `serve.name` and `serve.namespace` expressions produce the right values.
- The response payload evaluates as expected.

### Simulate handoff

With `--simulate`, play hands the built CR (already stamped with provenance annotations) directly to `ork simulate`. This covers the full local delivery loop without a cluster:

```text
intent file  →  ork serve play  →  CR  →  ork simulate  →  child resources
```

`--simulate` without a path runs simulate in op-print mode. `--simulate simulate.yaml` runs simulate in assert mode — katalog, cycles, `skipExternal`, and `expect:` all come from the spec, but play's CR is substituted for `spec.cr`. This lets you write intent-level tests that assert both that the intent produces the right CR and that the CR produces the right child resources.

```yaml
# simulate.yaml alongside your intent file
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: payment-service-intent
spec:
  katalog: ../../katalog.yaml
  cr: ""            # cr is ignored — play supplies the CR
  cycles: 5
  skipExternal: true
  expect:
    steady: true
    steadyAt: 2
    noErrors: true
    ops:
      - cycle: 1
        verb: create
        resource: deployments
```

None of this requires a running gateway, cluster access, or a token secret — just a Katalog, an intent file, and optionally a simulate spec.

---

## `ork serve apply`

Apply an intent or CR to a live gateway via `POST /api/v1/apply`.

The file may be a flat intent (target mode) or a full CR (`apiVersion` + `kind`). Both YAML and JSON are accepted. If `--file` is not given, `ork serve apply` looks for `intent.yaml` first, then `intent.json`, in the current directory.

The gateway handles everything on the other side: target resolution, token validation, admission, provenance stamping, and CR delivery. This command sends the body and prints the structured response.

```bash
ork serve apply [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | _(auto-detect)_ | Intent or CR file (YAML or JSON) |
| `--api` | | `http://localhost:8080` | Gateway base URL |
| `--token` | | _(required)_ | Bearer token for the gateway |
| `--dry-run` | | `false` | Preview without applying — the CR is not written to the cluster |

### Examples

```bash
# Apply from the default intent.yaml in the current directory
ork serve apply --api https://gateway.myorg.io --token "$ORK_TOKEN"

# Apply an explicit file
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"

# Dry run — validate at the gateway without applying
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN" --dry-run

# Apply a full CR directly (not target mode)
ork serve apply -f cr.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"

# Local gateway (default port)
ork serve apply --token "$ORK_TOKEN"
```

### Output

Accepted:

```text
  serve apply  intent.yaml

  ✓ PlatformResource  team-payments/my-payment-service
  poll: https://gateway.myorg.io/api/v1/resources/platformresource/team-payments/my-payment-service

accepted
```

Rejected:

```text
  serve apply  cr-bad.yaml

  ✗ images must come from the internal registry (registry.internal/)
    ↳ spec.image  images must come from the internal registry (registry.internal/)

apply rejected
```

Dry run:

```text
  serve dry-run  intent.yaml

  ✓ PlatformResource  team-payments/my-payment-service

dry-run accepted
```

### How it works

`ork serve apply` reads the file, marshals it to JSON, and posts it to `{api}/api/v1/apply`. The gateway detects whether the body is target mode (flat fields with a `target` key) or full CR mode (`apiVersion` + `kind`) and handles both paths identically. The command exits with code 0 on acceptance and code 1 on rejection.

`--dry-run` adds `?dryRun=true` to the request. The gateway runs the full admission pipeline — token check, field validation, admission webhooks — but does not write the CR to etcd. Use it to validate an intent against a live gateway before committing.

### Intent file vs CR file

| Shape | Detection | When to use |
|-------|-----------|-------------|
| Flat fields + `target:` | `"target"` key present | Normal developer workflow — same shape as the Control Center form |
| `apiVersion` + `kind` + `metadata` + `spec` | No `target` key | Applying a pre-built CR directly, CI pipelines that assemble full objects |

Both shapes are valid input. The gateway's field declaration determines where fields land in the CR (`spec.*`, `metadata.labels.*`, etc.). A full CR bypasses the gateway's field routing and is applied as-is, subject only to admission.

### GitOps pattern

The gateway validates on every apply — token permissions, field constraints, admission rules. A rejected apply exits non-zero and the pipeline stops.

Use `--dry-run` to verify an intent against a live gateway before committing:

```bash
# Verify — gateway validates, nothing is applied
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN" --dry-run

# Deliver — gateway applies
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"
```

`ork serve play` requires the Katalog — it is a platform team tool for testing Katalog changes before deploying. A consumer CI pipeline has only the intent file and a token; `--dry-run` is the right validation path for that case.

Rollback: revert the intent file in Git and re-run `ork serve apply` against the reverted commit.

---

## Related

- [`ork validate`](03-validate.md) — full Katalog validation
- [`idp` concept](../../concepts/idp/) — conceptual overview
- [`serve.fields` schema reference](../../reference/schema/02-katalog/20-serve.md#servefieldsname)
- [`serve.config.response` schema reference](../../reference/schema/02-katalog/20-serve.md#serveconfigresponse)
- [`serve.tokens` schema reference](../../reference/schema/02-katalog/20-serve.md#servetokens)
