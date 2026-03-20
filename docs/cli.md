# **Orkestra CLI**

`ork` is the command-line interface for Orkestra. It covers the full
operator lifecycle — scaffold, validate, preview, generate, and run.

```bash
ork <command> [flags]
```

---

## Commands

| Command | Description |
|---|---|
| `ork init` | Scaffold a new operator project |
| `ork validate` | Validate a Katalog or Komposer |
| `ork template` | Preview the merged, validated Katalog |
| `ork generate runtime` | Generate runtime wiring for typed CRDs and Go hooks |
| `ork run` | Start the operator runtime |
| `ork version` | Print version information |

---

## `ork init`

Scaffold a new Orkestra operator project.

```bash
ork init <project-name> [flags]
```

**Flags:**

| Flag | Description |
|---|---|
| `--typed` | Scaffold a Go project for typed CRDs or custom reconcilers |
| `--module` | Go module path — only used with `--typed` (default: `github.com/myorg/<name>`) |

### Dynamic project (default)

The default. Creates a Katalog-only project — no Go, no build step. The
installed `ork` binary is your operator.

```bash
ork init my-operator
cd my-operator
```

**What gets created:**

```
my-operator/
  examples/
    website/
      website-crd.yaml        sample CRD definition
      website-cr.yaml         sample CRs to apply
      website-katalog.yaml    starter Katalog
  katalogs/                   your CRD declarations go here
  .env.example                all config options with defaults
  .gitignore
  README.md
```

**Start immediately:**

```bash
kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml
```

### Typed project

For operators that need compiled Go types, Go hooks, or custom reconcilers.
Runs `go mod tidy` automatically.

```bash
ork init my-operator --typed
ork init my-operator --typed --module github.com/myorg/my-operator
```

**Additional files created:**

```
my-operator/
  cmd/
    orkestra/
      main.go       thin entrypoint — calls cli.Execute
  pkg/
    runtime/        generated files go here
    hooks/          write your Go hooks here
  go.mod
```

**Start with:**

```bash
go run ./cmd/orkestra/ run --katalog examples/website/website-katalog.yaml
```

---

## `ork validate`

Validate a Katalog or Komposer. Resolves all sources, merges, and runs
the full validation pipeline. Exits non-zero on any error.

```bash
ork validate --katalog <path> [--katalog <path>...]
```

**Flags:**

| Flag | Description |
|---|---|
| `--katalog` | Path or URL to a Katalog or Komposer (repeatable) |

**Examples:**

```bash
# Single file
ork validate --katalog ./katalog.yaml

# Multiple entry points — merged before validation
ork validate --katalog ./infra.yaml --katalog ./apps.yaml

# Remote
ork validate --katalog https://raw.github.com/myorg/crds/main/katalog.yaml

# Environment variable
ork validate --katalog $KATALOG_PATH
```

**Success:**
```
Success: Katalog is valid
```

**Errors are specific and actionable:**

```bash
# Missing required field
CRD "website-staging": missing required fields: apiTypes.group, apiTypes.version, apiTypes.kind

# Duplicate CRD name
merger: duplicate CRD "website": defined in
  "file:./katalogs/website.yaml" and
  "url:https://remote/app.yaml"

# Dependency cycle
dependency cycle detected involving 'project'

# Duplicate GVK
duplicate GVK detected: demo.orkestra.io/v1alpha1, Kind=Website
  (CRDs: website and website-staging)

# Katalog with sources block
"./katalog.yaml": kind Katalog cannot declare sources —
  use kind: Komposer to compose multiple Katalogs

# Komposer sourcing another Komposer
"./komposer.yaml" sources.files["./other.yaml"]:
  a Komposer cannot source another Komposer —
  only Katalog files are valid sources
```

---

## `ork template`

Render and print the merged, validated Katalog. Shows the post-merge,
post-validation state with all defaults applied — exactly as Orkestra
will see it at runtime.

```bash
ork template --katalog <path> [flags]
```

**Flags:**

| Flag | Description |
|---|---|
| `--katalog` | Path or URL to a Katalog or Komposer (repeatable) |
| `--json` | Output CRDs as JSON |
| `--yaml` | Output CRDs as YAML |
| `--graph` | Show ASCII dependency graph |
| `--verbose`, `-v` | Show all fields per CRD |

### Default — summary

Shows which CRDs are declared and their dependency relationships.

```bash
ork template --katalog ./katalog.yaml
```

```
Success: Katalog is valid

Rendered CRDs:
  - website  (depends on: orkapp)
  - orkapp
  - platformnamespace
  - database
  - website-staging  (depends on: orkapp)
```

### `--json` — full post-validation state

Shows every field after defaults are applied. Useful for verifying that
modes are set correctly, finalizers are inherited, and overrides took effect.

```bash
ork template --katalog ./katalog.yaml --json
```

```json
[
  {
    "name": "website",
    "enabled": true,
    "group": "demo.orkestra.io",
    "version": "v1alpha1",
    "kind": "Website",
    "plural": "websites",
    "namespaced": true,
    "namespace": "orkestra",
    "workers": 2,
    "resync": "30s",
    "dependsOn": ["orkapp"],
    "finalizers": ["finalizer.demo.orkestra.io/website"],
    "mode": "dynamic"
  }
]
```

### `--yaml` — YAML output

Same content as `--json` in YAML format.

```bash
ork template --katalog ./katalog.yaml --yaml
```

```yaml
- name: website
  enabled: true
  group: demo.orkestra.io
  version: v1alpha1
  kind: Website
  plural: websites
  namespaced: true
  namespace: orkestra
  workers: 2
  resync: 30s
  dependsOn:
    - orkapp
  finalizers:
    - finalizer.demo.orkestra.io/website
  mode: dynamic
```

### `--graph` — dependency graph

ASCII tree showing which CRDs depend on which. Useful for understanding
startup order before running.

```bash
ork template --katalog ./katalog.yaml --graph
```

```
Dependency Graph:
website
  └─ orkapp
orkapp
platformnamespace
database
website-staging
  └─ orkapp
```

CRDs with no dependencies appear at the root. Indented entries are what
the parent depends on — `website` starts only after `orkapp` is ready.

### `--verbose` — all fields

Every field for every CRD in human-readable format.

```bash
ork template --katalog ./katalog.yaml --verbose
```

```
Verbose merged katalog output:
CRD: website
  Group: demo.orkestra.io
  Version: v1alpha1
  Kind: Website
  Plural: websites
  Enabled: true
  Namespaced: true
  Namespace: orkestra
  Workers: 2
  Resync: 30s
  DependsOn: [orkapp]
  Finalizers: [finalizer.demo.orkestra.io/website]
  Mode: dynamic
  ...
```

### Combining with a Komposer

`ork template` works with Komposers. All sources are resolved and merged
before printing. What you see is the final merged state.

```bash
ork template --katalog ./komposer.yaml --graph
```

```
Dependency Graph:
website
platformnamespace
database    ← from Helm chart, overridden inline (workers: 4)
cache       ← from Helm chart
```

---

## `ork generate runtime`

Generate `zz_generated_runtime_registry.go` from the Katalog. Only needed
when your operator uses typed CRDs, Go hooks, or custom constructors.
Dynamic template operators do not need this step.

```bash
ork generate runtime --katalog <path> [flags]
```

**Flags:**

| Flag | Description |
|---|---|
| `--katalog` | Path or URL to a Katalog or Komposer (repeatable) |
| `--dry-run` | Print generated output to stdout without writing files |

**When you need it:**

| Situation | Needed? |
|---|---|
| Dynamic CRD with `onCreate`/`onReconcile`/`onDelete` templates | **No** |
| Dynamic CRD with `reconciler.hooks` declared | **Yes** |
| Typed CRD (`apiTypes.location` set) | **Yes** |
| `reconciler.default: false` with constructor | **Yes** |

**Examples:**

```bash
# Generate from a single Katalog
ork generate runtime --katalog ./katalog.yaml

# Preview without writing — use in CI to verify registry is up to date
ork generate runtime --katalog ./katalog.yaml --dry-run

# Generate from a Komposer
ork generate runtime --katalog ./komposer.yaml

# Multiple sources
ork generate runtime --katalog ./infra.yaml --katalog ./apps.yaml
```

**What it generates:**

```go
// pkg/runtime/zz_generated_runtime_registry.go
func RegisterRuntimeObjects() {
    // ObjectRegistry + ListRegistry entries for typed CRDs
    // HookRegistry entries for Go hooks
    // ReconcilerRegistry entries for custom constructors
}

func RegisterTypedScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
    // AddToScheme calls for each typed CRD
}
```

**After generating, always run:**

```bash
go mod tidy
```

**Pure dynamic output:**

```
nothing to generate — all CRDs use declarative templates
(interpreted at runtime by GenericReconciler, no generate step needed)
```

---

## `ork run`

Start the Orkestra operator runtime.

```bash
ork run --katalog <path> [flags]
```

**Flags:**

| Flag | Description |
|---|---|
| `--katalog` | Path or URL to a Katalog or Komposer (repeatable) |

The merger runs first — if merging or validation fails, the runtime never
starts. Fail fast before any reconcilers start.

**Examples:**

```bash
# Single Katalog
ork run --katalog ./katalog.yaml

# Komposer
ork run --katalog ./komposer.yaml

# Multiple entry points
ork run --katalog ./infra.yaml --katalog ./apps.yaml

# Remote
ork run --katalog https://config.company.com/crds/production-katalog.yaml

# Environment variable
ork run --katalog $KATALOG_PATH
```

**What happens after starting:**

```
katalogs merged     total=5 enabled=5
all informer caches synced
starting 2 workers for demo.orkestra.io/v1alpha1, Kind=Website
website workers started and ready
starting 3 workers for orkestra.konduktor.io/v1alpha1, Kind=OrkApp
orkapp workers started and ready
```

**Endpoints available immediately:**

```bash
GET /health                       # liveness
GET /ready                        # readiness
GET /metrics                      # Prometheus metrics
GET /katalog                      # all CRDs with health and dependency graph
GET /katalog/{crd}                 # single CRD config and reconcile stats
GET /katalog/{crd}/health          # 200 healthy / 503 degraded
```

---

## `ork version`

Print version, commit, and build date.

```bash
ork version
```

```
   ___       _              _
  / _ \ _  _| |___ _ _  ___| |_ _ _ __ _
 | (_) | || | / -_) ' \/ -_)  _| '_/ _` |
  \___/ \_,_|_\___|_||_\___|\__|_| \__,_|
          O R K E S T R A

Orkestra version: v1.0.0
Commit:           abc1234f
Built:            2026-03-20T10:00:00Z
```

In development builds (not from a release binary): `version: dev`.

---

## Typical workflows

### Zero-code dynamic operator

```bash
# 1. Scaffold
ork init my-operator && cd my-operator

# 2. Edit examples/website/website-katalog.yaml with your CRD
# 3. Validate before running
ork validate --katalog examples/website/website-katalog.yaml

# 4. Preview what will run
ork template --katalog examples/website/website-katalog.yaml --graph

# 5. Apply CRD and run
kubectl apply -f examples/website/website-crd.yaml
ork run --katalog examples/website/website-katalog.yaml
```

### Typed operator with Go hooks

```bash
# 1. Scaffold
ork init my-operator --typed && cd my-operator

# 2. Write your Katalog with reconciler.hooks declared
# 3. Validate
ork validate --katalog katalogs/my-katalog.yaml

# 4. Generate runtime wiring
ork generate runtime --katalog katalogs/my-katalog.yaml

# 5. Install dependencies
go mod tidy

# 6. Run
go run ./cmd/orkestra/ run --katalog katalogs/my-katalog.yaml
```

### Komposer with multiple sources

```bash
# 1. Validate the composed result
ork validate --katalog ./komposer.yaml

# 2. See what sources contributed and final state
ork template --katalog ./komposer.yaml --json | jq '.[].name'

# 3. Verify the dependency graph
ork template --katalog ./komposer.yaml --graph

# 4. Run
ork run --katalog ./komposer.yaml
```

### CI pipeline

```bash
# Validate in CI — fail fast before deploying
ork validate --katalog ./katalog.yaml

# Verify generated registry is up to date (for typed operators)
ork generate runtime --katalog ./katalog.yaml --dry-run
```