## v0.7.1 — Simulate + E2E fixes

### `ork simulate`

- **`e2e.yaml` input** — `ork simulate -f e2e.yaml` reads `spec.katalog` and `spec.cr`; skips `customOperator: true`; expands aggregator imports recursively.
- **`./...` discovery** — discovers all `*e2e.yaml` files, prints per-file results and aggregate summary; exit code 1 on cycle errors; `--skip` patterns follow the same convention as `ork e2e ./... --skip`.
- **`--skip-external`** — stubs `external:` HTTP calls with empty 200 responses. Default: calls hit the real network.
- **`--debug-ops`** — prints every recorded op with cycle number (diagnostic).
- **GVK-aware CR matching** — multi-document CR files (separated by `---`) supported; each CRD matched to the CR whose `kind` matches `apiTypes.kind`; CRDs with no matching CR skipped with a note.
- **`cross:` observation** — when sibling CRDs' CRs are present in the CR file, they are seeded into per-CRD fake informers and wired as `katalogRegistry`; `cross.*` fields populate correctly.
- **Hook wiring** — `HookRegistry[gvk]` looked up at runtime; custom binary runs real hooks against the fake cluster.
- **Constructor wiring** — `ReconcilerRegistry[gvk]` looked up; constructor called with `fakeKube`, `informer`, and `event.Discard()`.
- **Typed indexer** — typed CRDs have their CR converted via JSON round-trip to the registered Go type before being seeded; fixes constructor type-assertions and hook `BindToObjectHooks` panics.
- **`event.Discard()`** — constructors receive a silent no-op recorder; `noop.go` removed; `Recorder` interface moved to `event.go`.
- **Motif path fix** — `isFileMotif()` implemented; motif import file paths absolutized relative to the katalog dir (same fix as `crdFile`).

### E2E

- **`crdFile` path in bundle** — `CRDFile` is now cleared from every `CRDEntry` after `populateAPITypesFromCRDFile` runs; `apiTypes` are fully populated before the field is erased. Prevents the runtime pod from trying to read a local filesystem path that does not exist inside the container.
- **`10-motif-composition` e2e** — `crd-worker.yaml` added to `setup.apply`; both CRDs are installed before the CR is applied.

### Internals

- **Logger default level** — `pkg/logger` sets `InfoLevel` in its own `init()`; typeregistry debug logs no longer appear before CLI flag parsing.
- **`NewReconcilerFunc`** — changed from `*event.Event` to `event.Recorder` interface; registry template closure updated to `kubeclient.KubeClient` + `event.Recorder`.

---

## v0.7.0 — Early Access

This release ships the first public early-access build of Orkestra. It focuses on the E2E framework, the typed operator toolchain, and making every example runnable end-to-end without manual pre-steps.

### E2E Framework

- **`wait:` on import declarations** — sleep a duration before an import starts; validated at load time, shown in `ork validate`.
- **`spec.customOperator: true`** — skip the Orkestra bundle and Helm install; use `ork e2e` as a pure test harness for any operator (cert-manager, custom controllers, etc.).
- **Shared Orkestra across imports** — one Helm install and one uninstall per suite; `sharedOrkestra` prevents namespace deletion from cascading across imports. All sync output suppressed.
- **`ork e2e ./...` discovery** — recursive discovery of e2e files, skips pure aggregators, supports `--wait` and `--skip`.
- **`--dry-run` flag** — single file calls `ork validate`; `./...` lists files with count and first ten.
- **Helm output suppression** — install/uninstall/setup output captured silently; included in error message on failure.
- **Bug fixes** — `isPureAggregator` inline check, `WaitForResource` for Deployments, `checkAll` command-before-resource ordering.

### CLI — `ork generate registry`

- **Registry generation fixed** — `generateKatalog` was setting `kat.Spec` but never calling `m.Enabled()`, so `kat.Enabled()` always returned nil. `generate.TypeRegistry` iterated over an empty map and silently exited on every invocation — no registry file was ever written.
- **`TypeRegistry` returns `(bool, error)`** — `ensureMainGo` is now only written when the registry actually produced content.
- **`CustomHooksEnabled` and `ConstructorEnabled` corrected** — both methods returned `== nil` instead of `!= nil`. Safe for declarative CRDs (all call sites are guarded by `DefaultReconcile()`) but would have caused panics and wrong behaviour for typed operators.
- **Output style** — `→ generating registry for <module>`, `✓ pkg/typeregistry/zz_generated_typeregistry.go`, `○ nothing to generate` — matches the rest of the CLI.

### CLI — `ork init --pack advanced`

- **Typed examples embedded** — `go:embed` silently skips subdirectories containing a `go.mod` (nested module rule). `09-hooks`, `10-constructor`, and `11-mixed-operator-pattern` were missing from every `ork init --pack advanced` invocation. Fixed by renaming `go.mod`/`go.sum` → `.txt` at embed time.
- **Transparent extraction** — `ork init` now restores `go.mod.txt` → `go.mod`, `go.sum.txt` → `go.sum`, and strips `//go:build ignore` from all extracted `.go` files. Users get a fully compilable project with no manual steps.
- **Makefile safety net** — `make registry` and `make build` in typed examples perform the same restoration for users who clone the repo directly.

### CLI — path-relative resolution

- `crdFile`, `crFiles`, `setup.apply`, and Komposer `imports.files` now resolve relative to the declaring file. `ork run -f /any/path/katalog.yaml` works from any directory.
- CLI-provided `-f` paths are converted to absolute on intake so downstream resolution is always anchored correctly.

### Examples

- **`examples/use-cases/full-stack-app`** — all six sub-katalogs switched from inline `apiTypes:` to `crdFile:`; manual `kubectl apply -f crd.yaml` pre-step eliminated from every walkthrough. `03-cross-crd/crd.yaml` split into `crd-managed-database.yaml` + `crd-database-backed-app.yaml`. `06-full-stack` uses a `setup:` block to pull in the managed-database dependency.
- **Advanced typed examples** — `09-hooks` (typed Go hooks), `10-constructor` (custom reconciler), and `11-mixed-operator-pattern` (dynamic + hooks + constructor together) fully implemented, documented, and e2e-ready.
- **Typed e2e documented** — READMEs for typed examples now explain `--helm-arg runtime.image.repository` / `--helm-arg runtime.image.tag` so `ork e2e` deploys the user's custom image (which contains the generated type registry) instead of the default Orkestra image.

### Documentation

- **`pkg/e2e` developer docs** — all four existing reference pages rewritten; three new pages: `05-imports.md`, `06-discovery.md`, `07-custom-operator.md`.
- **Schema docs** — `wait:` imports field and `spec.customOperator` documented.
- **Getting-started** — CLI reference table updated; `ork run -f` note on optional `-f` when `katalog.yaml` is in the current directory.
- **Blog** — `04-why-i-built-this.md`.

---

## pre-v1-alpha — Public Deployment, CRD Health Stability, Multi-Runtime

### Public Deployment (`deployments/public/`)

- Six-cluster public demo: 6 runtimes, 75 CRDs, 1 Control Center aggregating all of them.
- Local motifs extracted for reuse across Katalogs (`replicaset-service`, `rotating-api-key`, `rotating-credentials`, `worker-pool`, `tls-cert`). Each CRD entry imports what it needs; no repeated boilerplate.
- All CRDs carry `subresources: status: {}` so the runtime can write status back to the API server.
- `make reload` — dev-only target that builds runtime + CC images to `/tmp/`, loads them into the `orkestra-playground` kind cluster, rewrites `values.yaml` with a `dev-<timestamp>` tag, and re-deploys all six clusters. Never touches the developer's local `ork` CLI binary.
- Runtime resource limits reduced to 50m/64Mi requests, 200m/256Mi limits — right-sized for kind clusters.
- Control Center set to 2 replicas.

### Bug Fixes — CRD Health State Machine

**CRD oscillating between healthy and pending on resync**
- `LastReconcile()` was calling `h.pending.Store(true)` as a side effect inside a getter. Every call to `HealthAsMap()` (on every health query) could silently flip a healthy CRD back to pending.
- `SetStarted()` was unconditionally setting `pending=true` on every worker start, including resync-triggered restarts — overwriting the `pending=false` set by `RecordSuccess()`. Now only sets pending if the CRD has not yet successfully reconciled.

**CRD showing "not started" or "degraded" under network lag**
- `postStartRetryInterval` was left at 3 seconds (a debug value) instead of the intended 30 seconds. This caused the retry loop to hit the API server every 3 seconds across all CRDs continuously.
- `crdExists()` collapsed all errors — including network timeouts and dial failures — into `false`, treating any transient API server hiccup as "CRD disappeared." Phase 1 (runtime disappearance check) and Phase 2 (missing CRD activation) would then call `SetMissingAtRuntime()` + `SetDegraded()`, flipping healthy CRDs to degraded and pending CRDs to "not started."
- Fixed: `crdExists()` now returns a tri-state — `(true, nil)` exists, `(false, nil)` definitively absent, `(false, err)` transient. All three callers skip state changes when `err != nil`.
- `postStartRetryInterval` restored to 90 seconds with exponential backoff capped at 5 minutes.

### RBAC — Namespace-Scoped ClusterRole Names

- `ClusterRole` and `ClusterRoleBinding` generated by `ork generate bundle` are now named `orkestra-<namespace>` and `orkestra-gateway-<namespace>`. Previously all runtimes shared the same names, so the last `kubectl apply` won — only that runtime's RBAC rules survived.

### Build — Docker Isolation

- `make docker`, `make docker-cc`, `make docker-gateway` now build Linux binaries to `/tmp/ork-docker-build/` instead of `~/.orkestra/bin/`. Docker builds no longer overwrite the developer's local `ork` CLI with a runtime binary (which has no `generate` or `validate` commands).

### Numbers

Memory baseline updated from estimate to measured: **79 MB RSS** for a live 10-CRD runtime (`process_resident_memory_bytes` from `/metrics`). README and website updated accordingly.

---

## Deletion Protection with Per‑CRD Overrides

- Added cluster‑wide deletion protection for any resource (custom or built‑in) labelled `orkestra.io/deletion-protection=true`.
- New `ValidatingWebhookConfiguration` (`protect.resources.orkestra.orkspace.io`) intercepts DELETE on all configured GVRs and denies if the label is present.
- Webhook rules are automatically built from:
  - All custom CRDs enabled in the Katalog (respecting per‑CRD `protectCRs` flag).
  - All built‑in resources marked `OrkestraInternal` (deployments, services, namespaces, RBAC, etc.).
- When `security.deletionProtection.enabled` is true, the reconciler automatically adds the protection label to every resource it creates or updates.
- Per‑CRD overrides allow fine‑grained control:
  ```yaml
  deletionProtection:
    protectCRD: false   # allow deletion of the CRD definition itself
    protectCRs: true    # protect instances (only effective if protectCRD=true)
    strictMode: true    # block removal of the protection label (per‑CRD override of global strictMode)
  ```
- Validation warnings (`ork validate`) appear for inconsistent combinations (e.g., `protectCRD=false` with `protectCRs=true`), guiding users toward correct configuration.
- The `ork validate` output now shows protection status with icons (🛡️ full, 🔓 CRD only, ⚠️ warning, ⛔ none) and lists warnings per CRD.
- Fixed a bug where `customResourceGVRs()` only added the first custom CRD’s GVR, causing incomplete webhook rules.
- Added `cleanupOnShutdown: true` to automatically remove the webhook configuration when the Gateway stops gracefully.

## Deletion Protection (Security)

- Added cluster‑wide deletion protection for any resource (custom or built‑in) labelled `orkestra.io/deletion-protection=true`.
- A new `ValidatingWebhookConfiguration` (`protect.resources.orkestra.orkspace.io`) intercepts DELETE requests on all configured GVRs and denies them if the target has the protection label.
- The webhook rules are automatically built from:
  - All custom CRDs enabled in the Katalog.
  - All built‑in resources marked `OrkestraInternal` (e.g., deployments, services, namespaces, RBAC objects, etc.).
- When `security.deletionProtection.enabled` is true, the reconciler automatically adds the protection label to every resource it creates or updates.
- In standalone mode (no reconciler), users can manually label existing resources; the webhook still honours the label.
- Fixed a bug in `customResourceGVRs()` that previously added only the first custom CRD GVR, causing incomplete webhook rules.
- Webhook `failurePolicy: Fail` ensures deletions are blocked when Orkestra Gateway is unreachable.
- `cleanupOnShutdown: true` removes the webhook configuration when the Gateway stops gracefully.

**Example Katalog snippet:**
```yaml
security:
  deletionProtection:
    enabled: true          # global switch
    cleanupOnShutdown: true
    failurePolicy: Fail
```

**Usage:**
- Orkestra‑managed resources are protected automatically.
- For external resources: `kubectl label <resource> orkestra.io/deletion-protection=true`
- Protected deletions are denied with a clear error message and remediation hint.

---
## CHANGELOG — simulate, validate, e2e, ork-action, CRD-driven inference

### **Added — `ork simulate` (in-memory operator simulation)**

New command that runs the full operator reconcile loop against a fake in-memory cluster — no Kubernetes required. Give it a Katalog and a CR and it shows exactly which resources are created, updated, or deleted each cycle.

- `pkg/simulate` — new package: `FakeKubeclient`, `Run`, `Result`, `CycleResult`, `Op`
- Fake clientset uses `PrependReactor` so every k8s operation is recorded before the default object tracker handles it
- CR is pre-seeded with managed labels and annotations so the reconciler's idempotency guards skip those patches every cycle
- Deployment status is advanced to `Available` after cycle 1 to unblock state machines waiting on `AvailableReplicas`
- Steady state detected when two consecutive cycles produce identical verb+resource sequences; `Result.SteadyAt` records the cycle number
- `--cycles N` always runs exactly N cycles; identical consecutive cycles are collapsed in output as `(cycles X–Y: identical)`
- `+` shown for creates, `~` for updates, `-` for deletes; if a resource is both created and patched in one cycle (e.g. `reconcile: true`), only `+` is shown
- Zerolog global logger is redirected to `io.Discard` during simulation so reconciler JSON logs don't pollute output
- CR YAML is converted via `sigsyaml.YAMLToJSON` before unmarshalling to ensure numeric fields are `float64` (k8s `DeepCopyJSON` requirement)
- Progressive documentation in `pkg/simulate/docs/` (output, steady state, limitations, internals)

### **Added — `ork e2e` command**

New command for declarative end-to-end tests against a real kind cluster. Provisions the cluster, installs operator dependencies, applies the CRD and bundle, starts the Orkestra runtime, applies the CR, and polls expectations.

- `docs/reference/cli/e2e.md` — full CLI reference page
- Added to `docs/reference/cli/index.md` operator commands table

### **Improved — `ork validate` output**

- Header now reads "Validating Katalog..." or "Validating Komposer..." based on the detected document kind
- E2E and Motif validation now use the same colour style as Katalog validation: `HealthIcon`, `ColorGray`, `Bold`, `ColorReset`
- Errors use `ColorRed` consistently across all document kinds

### **Added — CRD-driven API inference in `ork run` (dev mode)**

- `crdFile:` in Katalog spec populates `apiTypes` (group, version, kind, plural) directly from the CRD YAML — no need to declare `apiTypes:` separately
- Dev mode auto-applies CRDs before starting the runtime
- Dev mode auto-provisions a kind cluster when no cluster is available

### **Improved — `ork-action` (GitHub Action)**

Replaced the complex Docker-based action with a minimal composite wrapper:
- Installs `ork` via curl, runs `ork e2e` — no Dockerfile, no entrypoint script
- Inputs: `e2e-file`, `ork-version`, `keep-cluster`, `cluster`

---

# **CHANGELOG — `onCreate.custom` / `onReconcile.custom`: Operator Composition via Custom Resources**

### **Added — Custom Resource lifecycle hooks (`onCreate.custom` / `onReconcile.custom`)**
Introduced first-class support for composing operators by creating and managing arbitrary Kubernetes Custom Resources from within Orkestra hook declarations. An operator can now declare a `custom` block under `onCreate` or `onReconcile` to create, update, and conditionally clean up any CRD-backed resource — enabling true operator-to-operator composition without bespoke integrations.

Key components:

- **New types (`pkg/types/custom_resource.go`)**
  Added `CustomResourceTemplateSource`, `CustomResourceMetadata`, and `CustomResourceCondition`.
  `HasStatus *bool` controls whether child status is read back into the parent resolver context.
  `BuildGVK()` and `ResolveGVR()` methods provide GVK/GVR resolution from the declarative spec.

- **Custom resource registry package (`pkg/orkestra-registry/customresources/`)**
  New package exposing `Create`, `Update`, `DeleteIfOwned`, `Resolve`, and `ResolvedCustomResourceSpec`.
  Owner references are set automatically — deleting the parent cascades to all child custom resources without requiring any `onDelete` hook.

- **Template resolution (`pkg/orkestra-registry/template/resolve_customresources.go`)**
  Added `ResolveCustomResourceTemplate` on the Resolver to expand motif templates and inject resolved values into the custom resource spec before apply.

- **Katalog validation (`pkg/katalog/validate_custom_resource.go`)**
  Validation rules for custom resource declarations are enforced at startup in the Katalog layer, matching the validation model used for deployments, statefulsets, and jobs.

- **Reconciler — run (`pkg/reconciler/run_customresource.go`)**
  `runCustomResources` evaluates conditions, checks GVK existence at runtime, and applies or cleans up child resources.
  If the target CRD is not installed at startup, `runCustomResources` skips gracefully and logs a warning; the kordinator's `retryMissingCRDs` loop refreshes the REST mapper when the CRD later appears.

- **Reconciler — forEach fan-out (`pkg/reconciler/expand_customresources.go`)**
  `forEach` support for custom resources mirrors the fan-out behaviour already available for deployments and statefulsets.

- **Reconciler — child status (`pkg/reconciler/children.go`)**
  `readCustomResourceGroup` reads child CR status into the parent resolver context.
  `hasStatus: false` skips the API call entirely (useful for fire-and-forget resources); `true` or `nil` (auto-detect) reads status back.

- **Drift correction**
  `Update` always corrects spec and metadata drift. `spec.Reconcile` no longer gates drift detection — it only controls whether `Update` is called on every reconcile cycle or only on `onCreate`.

- **`hasStatus` pointer semantics**
  `hasStatus` is a `*bool` pointer: `nil` = auto-detect, `true` = read child status into parent resolver context, `false` = skip.
  Orkestra does NOT write status to child CRs — Layer 1 (Ready) is only written to the owner CRD by the generic reconciler.

- **Unified replica parsing (`pkg/orkestra-registry/common/parse.go`)**
  Added `ParseReplicas(s string) int32` to unify replica string-to-int32 parsing across deployments, statefulsets, replicasets, pods, jobs, and cronjobs.

- **Motif input quoting fix (`pkg/motif/expander.go`)**
  `renderInputs` now strips YAML quotes for inputs declared as `type: integer` or `type: bool`, preventing the `Invalid value: "string"` class of errors when Orkestra-rendered values are applied to strictly-typed CRD fields.

### **Impact**
Custom resource hooks enable Orkestra operators to compose other operators declaratively.
Any CRD-backed resource can be created, updated, and garbage-collected as a first-class child of an Orkestra-managed CR.
Missing CRDs are handled gracefully at runtime, and owner-reference-based cascading deletion removes the need for explicit cleanup hooks.

---

# **CHANGELOG — ONCOP Integration (Orkestra Native Cross‑Operator Protocol)**

### **Added — ONCOP v1 (Orkestra Native Cross‑Operator Protocol)**  
Introduced ONCOP as the unified, typed, cross‑operator observation protocol for Orkestra. ONCOP replaces ad‑hoc HTTP integrations and hard‑coded URLs with a declarative, URL‑inferable, cache‑aware protocol used across autoscaling, status fields, and template resolution.

Key components:

- **Typed observation surfaces**  
  Added first‑class ONCOP types:  
  `metrics`, `health`, `cr`, `info`, `events`  
  Each type maps to a deterministic URL shape under `/katalog/<crd>`.

- **URL inference engine**  
  Implemented `BuildONCOPURL` to construct ONCOP URLs from `CrossCRDDeclaration` using:  
  `source.host`, `source.type`, `crd`, `selector.namespace`, `selector.name`.

- **Cross‑operator resolver integration**  
  Updated `readCross()` to support ONCOP host‑based reads as Path 2, after informer registry and before raw endpoint fallback.  
  Responses injected into `.cross.<as>` for templates, autoscale conditions, and status fields.

- **New ONCOP type: `cr`**  
  Added `type: cr` for CR‑specific detail (`status`, `spec`, `children`, `metrics`).  
  Distinguishes CR detail from CRD‑level `info`.

- **Autoscaler ONCOP support**  
  Autoscale conditions now resolve `cross.<crd>.metrics.*` via ONCOP metrics endpoint with optional caching (`cacheFor:`).

- **Resolver enhancements**  
  Added `ParseCrossField` and extraction helpers (`ExtractCrossCRD`, `ExtractCrossCategory`, `ExtractCrossFieldName`, `ExtractCrossNamespace`) to unify cross‑field parsing.

- **Fallback semantics**  
  Resolution priority formalised as:  
  `informer registry → ONCOP host → raw endpoint → empty result`.

- **Cross‑binary caching**  
  Added per‑source caching for ONCOP responses to avoid repeated remote calls.

### **Impact**  
ONCOP enables consistent, declarative, cross‑operator observation across Orkestra.  
Autoscalers, status fields, and templates now consume cross‑operator data without bespoke integrations or hard‑coded URLs.  
Operators implementing ONCOP become first‑class participants in the Orkestra ecosystem.
