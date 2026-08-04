# ork idp

Inspect and validate Internal Developer Platform (IDP) configurations.

The IDP is the contract between platform teams and developers. These commands let you inspect and validate IDP configurations without needing to run the gateway or access a cluster.

## Commands

| Command | Description |
|---------|-------------|
| [`validate`](#ork-idp-validate) | Validate IDP configuration in a Katalog |
| [`schema`](#ork-idp-schema) | Show the flat schema for an IDP target |
| [`fields`](#ork-idp-fields) | List all IDP fields with their paths and types |
| [`tokens`](#ork-idp-tokens) | Show token permissions for a CRD |
| [`targets`](#ork-idp-targets) | List all IDP targets in a Katalog |
| [`can-i`](#ork-idp-can-i) | Check if a token can perform an operation |
| [`response`](#ork-idp-response) | Show the IDP response configuration |

---

## `ork idp validate`

Validate IDP configuration in a Katalog.

This runs the same IDP-specific validations as `ork validate`, but only for IDP concerns: fields, paths, tokens, response config, and namespace rules. It does not check the full Katalog schema — only the IDP portions.

```bash
ork idp validate
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--full` | | Show a detailed breakdown of the IDP configuration for each IDP-enabled CRD. |

### Examples

```bash
# Validate IDP configuration
ork idp validate
```

### Output

```text
✓ IDP configuration is valid
```

### With `--full`

```bash
# Validate IDP configuration with detailed breakdown
ork idp validate --full
```

### Output

```text
IDP Configuration Summary
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
✓ IDP configuration is valid
  2 IDP-enabled CRD(s)
```
---

## `ork idp schema`

Show the flat schema for an IDP target.

This displays the fields that callers can submit for a target, including their labels, types, enums, and paths. The output is the same flat field structure returned by the gateway's `GET /api/v1/schema?target=<t>` endpoint.

```bash
ork idp schema [flags]
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
ork idp schema --target smartapp

# Show schema by kind
ork idp schema --kind AppRequest

# Show schema by CRD name
ork idp schema --name application
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

## `ork idp fields`

List IDP fields with their paths and types.

This shows fields declared in `idp.fields` and `idp.additionalFields` across all IDP-enabled CRDs. With `--target`, `--kind`, or `--name`, shows fields for a specific CRD.

```bash
ork idp fields
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target to show fields for |
| `--kind` | `-k` | Kind to show fields for |
| `--name` | `-n` | CRD name to show fields for |

### Examples

```bash
# List all fields across all CRDs
ork idp fields

# List fields for a specific target
ork idp fields --target smartapp

# List fields for a specific kind
ork idp fields --kind AppRequest

# List fields for a specific CRD name
ork idp fields --name application
```

### Output (All fields)

```text
IDP Fields
──────────────────────────────────────────────────────────────────────

CRD: application (target: smartapp)
  FIELD           TYPE    PATH                    SOURCE
  repository      string  app.repository          spec
  image           string  app.image               spec
  cpu             string  app.resources.cpu       spec
  team            string                          label
  environment     string                          label

Total: 5 fields across 1 CRD
```

### Output (Specific CRD)

```text
Fields for: application (target: smartapp)
──────────────────────────────────────────────────────────────────────
FIELD           TYPE    PATH                    SOURCE      REQUIRED
repository      string  app.repository          spec        ✓
image           string  app.image               spec        ✓
cpu             string  app.resources.cpu       spec
team            string                          label       ✓
environment     string                          label
```

---

## `ork idp tokens`

Show token permissions for an IDP-enabled CRD.

This displays the `allowedTokens` configuration, including which tokens have access, their permissions (`global`/`schema`/`resources`), and namespace restrictions.

```bash
ork idp tokens [flags]
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
ork idp tokens --target smartapp
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

## `ork idp targets`

List all IDP-enabled targets in a Katalog.

This shows each target, its CRD kind, and whether it has fields defined.

```bash
ork idp targets
```

### Examples

```bash
# List all targets
ork idp targets
```

### Output

```text
IDP Targets
──────────────────────────────────────────────────────────────────────
TARGET      KIND        FIELDS  TOKENS
smartapp    AppRequest  12      yes
database    Database    8       no

2 target(s)
```

---

## `ork idp can-i`

Check if a token can perform an operation.

This evaluates the same permission checks that the gateway applies to incoming Apply API requests. It considers token existence, permissions (`global`/`schema`/`resources` scopes), namespace restrictions, and target existence.

```bash
ork idp can-i [flags]
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
ork idp can-i --token control-center --target smartapp --operation create

# Check if token can delete in a namespace
ork idp can-i --token ci-pipeline --target smartapp --operation delete --namespace staging

# Check if token can list
ork idp can-i --token monitoring --target smartapp --operation list
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

## `ork idp response`

Show the IDP response configuration.

This displays what callers will see in the Apply API response based on `idp.config.response`. It shows `default: true/false`, payload fields (with their template expressions), excluded paths, and poll URL configuration.

No cluster access is required — this reads the Katalog directly.

```bash
ork idp response [flags]
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
ork idp response --target smartapp

# Show response config with preview
ork idp response --target smartapp --preview
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
- [`idp.fields` schema reference](../../reference/schema/02-katalog/20-idp.md#idpfieldsname)
- [`idp.config.response` schema reference](../../reference/schema/02-katalog/20-idp.md#idpconfigresponse)
- [`idp.allowedTokens` schema reference](../../reference/schema/02-katalog/20-idp.md#idpallowedtokens)
```text
