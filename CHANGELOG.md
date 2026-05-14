# **CHANGELOG — `onCreate.custom` / `onUpdate.custom`: Operator Composition via Custom Resources**

### **Added — Custom Resource lifecycle hooks (`onCreate.custom` / `onUpdate.custom`)**
Introduced first-class support for composing operators by creating and managing arbitrary Kubernetes Custom Resources from within Orkestra hook declarations. An operator can now declare a `custom` block under `onCreate` or `onUpdate` to create, update, and conditionally clean up any CRD-backed resource — enabling true operator-to-operator composition without bespoke integrations.

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
