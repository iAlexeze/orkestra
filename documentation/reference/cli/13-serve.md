# ork serve

Inspect and validate Internal Developer Platform (Serve) configurations.

The Serve is the contract between platform teams and developers. These commands let you inspect and validate Serve configurations without needing to run the gateway or access a cluster.

## Commands

| Command | Description |
|---------|-------------|
| [`validate`](#ork-serve-validate) | Validate Serve configuration in a Katalog |
| [`schema`](#ork-serve-schema) | Show the flat schema for an Serve target |
| [`fields`](#ork-serve-fields) | List all Serve fields with their paths and types |
| [`tokens`](#ork-serve-tokens) | Show token permissions for a CRD |
| [`targets`](#ork-serve-targets) | List all Serve targets in a Katalog |
| [`can-i`](#ork-serve-can-i) | Check if a token can perform an operation |
| [`response`](#ork-serve-response) | Show the Serve response configuration |

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

✅ application
  target: smartapp  /  kind: AppRequest
  name:      "{{ repoSlug .spec.repository }}"
  namespace: "{{ teamName }}-{{ environmentName }}"
  fields:    12 total (spec: 8, labels: 2, annotations: 2)
  nested:    5 path(s)
  tokens:    4 token(s) with restrictions
  response:  default: true, payload: 4, exclude: 3
  poll:      url, field

✅ database
  target: db  /  kind: Database
  name:      "{{ repoSlug .spec.repository }}"
  namespace: "{{ teamName }}"
  fields:    6 total (spec: 4, labels: 1, annotations: 1)
  tokens:    none (all tokens allowed)
  response:  none (default CR response)

──────────────────────────────────────────────────────────────────────
✓ Serve configuration is valid
  2 serve-enabled CRD(s)
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

One of `--target`, `--kind`, or `--name` is required.

### Examples

```bash
# Show schema by target
ork serve schema --target smartapp

# Show schema by kind
ork serve schema --kind AppRequest

# Show schema by CRD name
ork serve schema --name application
```

### Output

```text
Schema for: application (target: smartapp)
──────────────────────────────────────────────────────────────────────
FIELD           LABEL            TYPE    PATH                    REQUIRED
repository      Repository       string  app.repository          ✓
image           Container Image  string  app.image               ✓
cpu             CPU Request      string  app.resources.cpu
replicas        Replicas         integer scaling.replicas
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
| `--sort-by` | | Sort fields by `"name"` (default) or `"order"` |

### Examples

```bash
# List all fields across all CRDs
ork serve fields

# List fields for a specific target
ork serve fields --target smartapp

# List fields for a specific kind
ork serve fields --kind AppRequest

# List fields for a specific CRD name
ork serve fields --name application

# Sort fields by order (as declared in the Katalog)
ork serve fields --target smartapp --sort-by order
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

Show token permissions for an serve-enabled CRD.

This displays the `serve.tokens` configuration, including which tokens have access, their permissions (`global`/`schema`/`resources`), and namespace restrictions.

```bash
ork serve tokens [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target to show tokens for |
| `--kind` | `-k` | Kind to show tokens for |
| `--name` | `-n` | CRD name to show tokens for |

One of `--target`, `--kind`, or `--name` is required.

### Examples

```bash
# Show tokens by target
ork serve tokens --target smartapp
```

### Output

```text
Token permissions for CRD: application (target: smartapp)
──────────────────────────────────────────────────────────────────────
TOKEN            GLOBAL  SCHEMA  RESOURCES           NAMESPACES
control-center   *       *       *                   *
ci-pipeline              get,list create,update,get,list staging
monitoring               get,list get,list           *
```

---

## `ork serve targets`

List all serve-enabled targets in a Katalog.

This shows each target, its CRD kind, and whether it has fields defined.

```bash
ork serve targets
```

### Examples

```bash
# List all targets
ork serve targets
```

### Output

```text
Serve Targets
──────────────────────────────────────────────────────────────────────
TARGET      KIND        FIELDS  TOKENS
smartapp    AppRequest  12      yes
database    Database    8       no

2 target(s)
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
| `--target` | `-t` | Target to check |
| `--kind` | `-k` | Kind to check |
| `--name` | `-n` | CRD name to check |
| `--operation` | `-o` | Operation to check **(required)** |
| `--namespace` | `-N` | Namespace to check (default: all namespaces) |
| `--class` | `-c` | Endpoint class to check (`resources`, `schema`) |

One of `--target`, `--kind`, or `--name` is required.

### Examples

```bash
# Check if token can create
ork serve can-i --token control-center --target smartapp --operation create

# Check if token can delete in a namespace
ork serve can-i --token ci-pipeline --target smartapp --operation delete --namespace staging

# Check if token can list
ork serve can-i --token monitoring --target smartapp --operation list
```

### Output (Allowed)

```text
✓ control-center can create on "smartapp"
```

### Output (Denied)

```text
✗ ci-pipeline cannot delete on "smartapp" in namespace "staging"
  Reason: token "ci-pipeline" does not have "delete" permission for resources class
  Available: 
    - schema:    get
    - resources: create,update,list,get
```

---

## `ork serve response`

Show the Serve response configuration.

This displays what callers will see in the Gateway API response based on `serve.config.response`. It shows `default: true/false`, payload fields (with their template expressions), excluded paths, and poll URL configuration.

No cluster access is required — this reads the Katalog directly.

```bash
ork serve response [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target to show response for |
| `--kind` | `-k` | Kind to show response for |
| `--name` | `-n` | CRD name to show response for |
| `--preview` | `-p` | Show a sample response preview |

One of `--target`, `--kind`, or `--name` is required.

### Examples

```bash
# Show response config
ork serve response --target smartapp

# Show response config with preview
ork serve response --target smartapp --preview
```

### Output

```text
Response configuration for: application (target: smartapp)
──────────────────────────────────────────────────────────────────────

default: true

Payload fields:
  FIELD         EXPRESSION
  phase         '{{ .status.phase }}'
  serviceURL    '{{ serviceURL }}'
  queueDepth    '{{ .external.queueDepth.value | default 0 }}'
  nextSteps     '{{ nextSteps }}'

Excluded paths:
  ✗ metadata.managedFields
  ✗ status.observedGeneration
  ✗ metadata.name

Poll URL:
  url:   "{{ devServerURL }}"
  field: "status.phase"
```

### With `--preview`

```text
──────────────────────────────────────────────────────────────────────

Response preview (templates shown as-is, not resolved):

{
  "phase": "{{ .status.phase }}",
  "serviceURL": "{{ serviceURL }}",
  "queueDepth": "{{ .external.queueDepth.value | default 0 }}",
  "nextSteps": "{{ nextSteps }}"
}

Excluded fields:
  ✗ metadata.managedFields
  ✗ status.observedGeneration
  ✗ metadata.name
```

---

## Related

- [`ork validate`](03-validate.md) — full Katalog validation
- [`idp` concept](../../concepts/idp/) — conceptual overview
- [`serve.fields` schema reference](../../reference/schema/02-katalog/20-serve.md#servefieldsname)
- [`serve.config.response` schema reference](../../reference/schema/02-katalog/20-serve.md#serveconfigresponse)
- [`serve.tokens` schema reference](../../reference/schema/02-katalog/20-serve.md#servetokens)
