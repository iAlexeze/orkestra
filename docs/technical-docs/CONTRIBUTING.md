# Contributing to Orkestra

Orkestra is built on a simple idea: operators should be declarations, not software projects. Contributing to Orkestra means either extending the runtime that makes that idea work, or growing the ecosystem of patterns that proves it at scale.

Both matter. This document covers both.

---

## Before you start

Read the following before making any code changes:

- [Technical Documentation Index](./index.md) — architecture overview and component map
- [Your CRD Is Enough](../blog/your-crd-is-enough.md) — the design philosophy
- [Trust and Failure Model](../publications/trust-and-failure-model.md) — the failure guarantees the codebase must uphold

Understanding these will prevent contributions that are technically correct but architecturally wrong. The most common mistakes from new contributors:
- Breaking per-CRD isolation by sharing state between CRD reconcilers
- Adding synchronous API calls where the informer cache should be used
- Adding a configuration option that violates "simple in design"

When in doubt, open an issue before writing code. A five-minute conversation prevents a week of wasted work.

---

## Development environment

### Requirements

- Go 1.22 or later
- `kubectl` configured against a local cluster (kind or minikube)
- `make` (for the build targets)
- `openssl` (for TLS certificate generation in tests)

Recommended:
- `oras` CLI (for testing OCI registry features)
- `helm` 3.x (for testing Helm source resolution)

### Clone and build

```bash
git clone https://github.com/iAlexeze/orkestra
cd orkestra
go build ./...
go test ./...
```

### Run locally against a cluster

```bash
# Apply a test CRD
kubectl apply -f examples/website/website-crd.yaml

# Run Orkestra with the example Katalog
go run ./cmd/ork/main.go run --katalog examples/website/website-katalog.yaml

# Apply a CR in another terminal
kubectl apply -f examples/website/website-cr.yaml

# Verify
kubectl get deployments
curl localhost:8080/katalog/website
```

### Run tests

```bash
# Unit tests only
go test ./pkg/... ./health/...

# All tests (requires cluster access)
go test ./...

# Specific package
go test ./pkg/merger/...

# With race detector
go test -race ./...
```

### Code generation

After modifying type definitions in `pkg/types/types.go`, re-run code generation:

```bash
make generate
```

After modifying a Katalog with `apiTypes.location` set, regenerate the runtime registry:

```bash
go run ./cmd/ork/main.go generate runtime --katalog katalog.yaml
```

---

## Codebase structure

```
cmd/
  ork/                   main binary — CLI entry points
    main.go
    run.go               ork run
    validate.go          ork validate
    generate.go          ork generate runtime
    status.go            ork status
    describe.go          ork describe
    events.go            ork events
    top.go               ork top

domain/
  object.go              Object interface (wraps runtime.Object + meta)
  reconciler.go          Reconciler interface
  hooks.go               ReconcileHooks[T], AnyReconcileHooks

health/
  health.go              HealthServer struct, NewHealthServer, Start, Shutdown
  handlers.go            /katalog, /katalog/{crd}, /katalog/{crd}/health
  conversion_handler.go  /convert endpoint
  conversion_stats.go    ConversionStats rolling window
  admission_review.go    AdmissionReview types
  admission_handlers.go  /validate and /mutate endpoints
  admission_evaluation.go validation and mutation rule evaluation
  admission_stats.go     AdmissionStats rolling window
  admission_metrics.go   Prometheus metrics for admission
  webhook_registration.go ValidatingWebhookConfiguration + MutatingWebhookConfiguration

pkg/
  katalog/
    katalog.go           KomposeKatalogFromYaml, Katalog struct
    builtins.go          Built-in kind registry (Deployment, Pod, etc.)
    enrichment.go        EnrichCRDEntry — discovery API + built-in lookup
    conversion_registry.go InMemoryConversionRegistry
    admission_registry.go  InMemoryAdmissionRegistry
    validation.go        ConversionRegistry interface

  merger/
    merger.go            Merger struct, Merge(), loadKatalogFile()
    file.go              loadFileSource(), loadFileWithAuth()
    helm.go              loadHelmSource()
    registry.go          loadRegistrySource(), pullPattern(), validatePatternStructure()
    registry_git.go      pullGitHubPattern(), pullGitLabPattern(), pullGenericGitPattern()
    registry_oci.go      pullOCIPattern(), orasPull()

  reconciler/
    generic.go           GenericReconciler[T], safeReconcile
    run_deployments.go   runDeployments() — template runner for Deployments
    run_services.go      runServices()
    run_secrets.go       runSecrets()
    run_configmaps.go    runConfigMaps()
    run_jobs.go          runJobs()
    run_cronjobs.go      runCronJobs()
    run_pods.go          runPods()
    run_serviceaccounts.go runServiceAccounts()
    conditions.go        evaluateConditions() for when: blocks
    validation.go        runValidation() — reconcile-time validation
    mutation.go          runMutation() — reconcile-time mutation

  kontroller/
    informer_factory.go  InformerFactory, per-CRD informer creation
    queue_registry.go    QueueRegistry, per-CRD queue creation
    health_tracker.go    CRDHealth, per-CRD health state tracking
    worker.go            runWorker(), worker pool management

  types/
    types.go             CRDEntry, APITypes, ReconcilerConfig, HookTemplates, etc.
    admission.go         ValidationRule, MutationRule, AdmissionWebhookConfig

  orkestra-registry/
    deployments/         orkdeploy.Create, Update, Delete, Resolve
    services/            orksvc.*
    secrets/             orksecret.*
    configmaps/          orkcm.*
    jobs/                orkjobs.*
    cronjobs/            orkcron.*
    pods/                orkpods.*
    serviceaccounts/     orksa.*
    template/            orktmpl.Resolver, NewResolver, NewResolverFromMap

  konfig/               Configuration constants, env var names, defaults
  kubeclient/           Kubeclient wrapper, FromContext
  logger/               zerolog wrapper
  utils/                LoadFile, LoadFileWithAuth, FileAuth
```

---

## Contribution types

### Bug fixes

1. Open an issue first if the bug is not obviously a typo or test failure
2. Include a minimal reproduction case (Katalog + CR + expected vs actual behaviour)
3. Fix with a test that would have caught the bug
4. Reference the issue in the PR

### Adding a resource type to OrkestraRegistry

OrkestraRegistry is designed to grow. Each new resource type follows the same four-function pattern: `Create`, `Update`, `Delete`, `Resolve`.

**Step 1: Define the template source type in `pkg/types/types.go`**

Add `XxxTemplateSource` to `HookTemplates` and define its fields. Follow the pattern of existing types. Include a `Version` field, a `Conditions []Condition` field, and `Reconcile bool`.

**Step 2: Add a resolver method to `pkg/orkestra-registry/template/resolver.go`**

```go
func (r *Resolver) ResolveXxxTemplate(src orktypes.XxxTemplateSource) (orktypes.XxxTemplateSource, error)
```

Resolve all string fields that might contain template expressions. Apply the namespace default.

**Step 3: Create `pkg/orkestra-registry/xxxs/xxx.go`**

```go
package orkxxx

type ResolvedXxxSpec struct {
    Name      string
    Namespace string
    // ... other fields
}

func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedXxxSpec) error
func Resolve(src orktypes.XxxTemplateSource, ownerName string) ResolvedXxxSpec
```

**Step 4: Add a runner in `pkg/reconciler/run_xxxs.go`**

```go
func runXxxs(ctx context.Context, kube *kubeclient.Kubeclient,
    resolver *orktmpl.Resolver, owner domain.Object,
    srcs []orktypes.XxxTemplateSource, update bool) error
```

**Step 5: Call the runner from `runTemplateReconcile` in `pkg/reconciler/generic.go`**

**Step 6: Write tests** — at minimum, test Create idempotency (calling Create twice should not error), Update when resource does not exist (should create it), and Delete when resource does not exist (should not error).

**Step 7: Update documentation** — add to the resource type table in the OrkestraRegistry technical docs.

### Adding a new CLI command

CLI commands live in `cmd/ork/`. Each command follows the same pattern:

```go
// cmd/ork/mycommand.go
package main

import (
    "github.com/spf13/cobra"
)

func newMyCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand [args]",
        Short: "Short description for ork --help",
        Long:  `Longer description shown when running ork mycommand --help.`,
        RunE:  runMyCommand,
    }
    // flags
    return cmd
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    // implementation
}
```

Register in `cmd/ork/main.go`:

```go
rootCmd.AddCommand(newMyCommand())
```

### Publishing a registry pattern

Registry patterns belong in the `konduktor-io/orkestra-registry` repository, not in this repository.

The pattern directory must contain exactly five files:

```
my-pattern/
  v1.0.0/
    crd.yaml        the CRD definition
    katalog.yaml    operator behavior
    komposer.yaml   example import
    cr.yaml         example CR
    README.md       documentation
```

Run `ork validate --katalog katalog.yaml` inside the directory before opening a PR. The CI pipeline runs this check on every PR.

See [Publishing a Pattern](../runtime-manual/concepts/registry-sources/publishing-a-pattern.md) for the complete guide.

---

## Code standards

### Error messages

Error messages must be lowercase, actionable, and specific enough to diagnose without reading source:

```go
// Good
return fmt.Errorf("enrichment failed for %q: kind %q not found in cluster discovery — ensure the CRD is installed", entry.Name, entry.APITypes.Kind)

// Bad
return fmt.Errorf("enrichment failed")
return fmt.Errorf("Error: enrichment failed for entry")
```

Always wrap errors with context using `%w`:

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("doing something for %q: %w", name, err)
}
```

### Logging

Use the `logger` package (zerolog wrapper). Log levels:

- `Debug` — internal state, per-reconcile details, template expression values
- `Info` — startup, shutdown, significant state transitions
- `Warn` — recoverable issues that the operator is handling
- `Error` — failures that require attention

Include structured fields, not interpolated strings:

```go
// Good
logger.Info().
    Str("crd", entry.Name).
    Str("kind", entry.APITypes.Kind).
    Msg("enrichment complete")

// Bad
logger.Info().Msgf("enrichment complete for %s (%s)", entry.Name, entry.APITypes.Kind)
```

### Tests

Every new function needs a test. The project uses standard `testing` package. Table-driven tests are preferred for functions with multiple input variants:

```go
func TestResolvedURL(t *testing.T) {
    tests := []struct {
        name    string
        src     RegistrySource
        wantURL string
        wantVer string
    }{
        {
            name:    "@ shorthand",
            src:     RegistrySource{URL: "ghcr.io/myorg/postgres@v14"},
            wantURL: "ghcr.io/myorg/postgres",
            wantVer: "v14",
        },
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotURL, gotVer := tt.src.ResolvedURL()
            if gotURL != tt.wantURL { t.Errorf(...) }
            if gotVer != tt.wantVer { t.Errorf(...) }
        })
    }
}
```

### Invariants that must not be broken

The following invariants are enforced by the test suite. Any PR that breaks them will not be merged:

1. **Per-CRD isolation** — workers, queues, and reconcilers must never be shared between CRD entries
2. **Idempotent registry functions** — Create, Update, Delete must be safe to call multiple times
3. **safeReconcile coverage** — every reconcile path must be wrapped
4. **No API calls in informer event handlers** — event handlers enqueue keys, they do not reconcile

---

## Pull request process

### Before opening a PR

```bash
# Format
gofmt -w ./...

# Lint (requires golangci-lint)
golangci-lint run ./...

# Test
go test ./...

# Build
go build ./...
```

### PR description template

```markdown
## What does this PR do?

<!-- One paragraph describing the change -->

## Why?

<!-- What problem does this solve? Link to issue if applicable -->

## Testing

<!-- How did you test this? Include commands to reproduce -->

## Checklist

- [ ] Tests added or updated
- [ ] Documentation updated (if user-facing change)
- [ ] `go test ./...` passes
- [ ] No new invariants broken
```

### Review expectations

- Reviews happen on a best-effort basis — there is no SLA
- Feedback may ask for architecture changes, not just code changes
- A review that requests significant changes is not a rejection
- Merged PRs may be reverted if production issues are traced to them

---

## Where to ask questions

- **GitHub Issues** — bug reports, feature requests, architecture questions
- **GitHub Discussions** — general questions, patterns, use cases
- **PR comments** — specific feedback on a change in flight

There is no Slack, Discord, or mailing list currently. GitHub is the single channel.

When opening an issue:
- Include the Orkestra version (`ork version`)
- Include the relevant Katalog YAML (redact sensitive values)
- Include the error message and relevant logs
- Include what you expected vs what happened

---

## Releasing

Releases are cut by the maintainer. The release process:

```bash
# Tag the release
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions handles:
#   - Go binary builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)
#   - GPG signing
#   - GitHub Release creation with checksums
#   - Docker image push to GHCR
#   - Helm chart packaging
#   - Homebrew tap update
```

Maintainers do not accept requests to cut releases on behalf of contributors. Patch releases for critical bugs may be cut on short notice.
