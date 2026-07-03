## v0.7.9 — Reconciler configuration, user-defined profiles, kubectl DSL, and resilience examples

### Resilience pack — new sub-examples

Three new examples under `examples/resilience/`, each with `ork validate`, `ork simulate`, `ork run` walkthrough, and a fully passing `ork e2e`:

| Example | What it teaches |
|---------|-----------------|
| `admission-protection` | Runtime validation as a resilience layer. Bad CR → `failureThreshold` exceeded → operator degrades. Patch to valid image → automatic recovery, no restart. |
| `crd-missing-recovery` | Runtime CRD watch. Delete the CRD at runtime without deletion protection — Orkestra detects the disappearance, degrades, and retries. Re-apply the CRD and CR → full recovery. `lastError` preserved as audit trail. |
| `leader-failover` | Leader election resilience. Helm-deployed runtime with `replicaCount: 2`. Kill the konductor pod — a follower is elected within `leaseDuration` and reconciliation continues. Covers lease inspection and leader election configuration. |

### Live API documentation

New concept section: **Every CRD is a Live API** (`documentation/concepts/live-api/`). Covers the runtime HTTP API on port 8080 — `/katalog`, `/katalog/{crd}`, `/katalog/{crd}/health`, `/katalog/{crd}/cr` — with real response shapes, `isKonductor` signal, `hasUnhealthyDependencies`, and the gateway API on port 8443. Includes a complete endpoint reference.

### E2E concept: testing leader-led deployments

New concept page (`documentation/concepts/e2e/06-leader-led-deployments.md`) explaining the follower routing problem and how `leaderElection` solves it. Covers `isKonductor`, lease namespace defaulting, when to use service vs. leader port-forward, and `kubectl.logs` with `leaderElection`. General-purpose — applies to any operator using Kubernetes leader election, not just Orkestra.

### Source caching for `ork pull -f`

`ork pull -f komposer.yaml` now pre-warms all three local cache namespaces so that subsequent `ork template`, `ork validate`, and `ork generate` calls are served from disk rather than making network requests:

| Cache | Location | Key |
|-------|----------|-----|
| Git Helm sources | `~/.orkestra/helm/git/<sha256>/` | `SHA256(repo + ref + subpath)` |
| Remote Helm repository charts | `~/.orkestra/helm/repo/<sha256>/` | `SHA256(repo + chart + version)` |
| Remote HTTPS files | `~/.orkestra/files/<sha256>/` | `SHA256(url)` |

Caches have no TTL — entries persist until `ork pull -f komposer.yaml --refresh` is run, which bypasses all three and overwrites the stored copies. `--refresh` on the OCI registry pull was already supported; it now extends to Helm and file sources.

### Bug fixes

- **Merger log noise**: git clone and Helm pull progress messages in `resolveGitChart` and `resolveRemoteChart` downgraded from `Info` to `Debug`. Only shown with `--debug`.
- **charts/examples gitlink**: dangling gitlink (`mode 160000`) in the repository index with no `.gitmodules` caused `ork template` to fail on machines that cloned the repo via the merger. Re-added as regular tracked files.
- **`.gitignore` bare `ork` pattern**: the entry `ork` matched `pkg/ork/` and blocked `pkg/ork/metrics.go` from being tracked. Changed to `/ork` (root-only).
- **govulncheck in CI**: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` added to `validate-pr` workflow. `go install` + call-by-name was tried first but `GOPATH/bin` is not on `PATH` in CI runners.
- **CVE dependency updates**: `x/crypto` → `v0.52.0`, `x/net` → `v0.55.0`, `containerd` → `v1.7.33`. Three remaining containerd advisories have `Fixed in: N/A` upstream and are not actionable.

### Breaking: reconciler config moves inside `operatorBox`

`workers`, `resync`, and `queue` (`maxDepth`, `failureThreshold`) move from the CRD root into `operatorBox.reconciler`. The `operatorBox` is the complete definition of what makes a CRD an operator — the reconciler runtime config belongs there alongside `onCreate`, `onReconcile`, `status`, and `admission`.

```yaml
# before
spec:
  crds:
    service:
      workers: 4
      resync: 30s
      queue:
        maxDepth: 200
        failureThreshold: 5
      operatorBox:
        onCreate: ...

# after
spec:
  crds:
    service:
      operatorBox:
        reconciler:
          workers: 4
          resync: 30s
          queue:
            maxDepth: 200
            failureThreshold: 5
        onCreate: ...
```

`ork validate` rejects the old layout with a clear error pointing to `operatorBox.reconciler`.

### New profile class: `reconciler`

Named presets for concurrency and queue tuning. Reference with `operatorBox.reconciler.profile`:

| Profile | workers | resync | queue.maxDepth |
|---------|---------|--------|----------------|
| `high-throughput` | 10 | 5m | 1000 |
| `conservative` | 2 | 1m | 100 |
| `development` | 1 | 30s | 50 |

Inline fields override the profile when both are declared. Retry is always exponential backoff — no `maxRetries`.

### User-defined profiles for all profile classes

All profile classes now support user-defined named presets declared in the `profiles:` block at the root of a Katalog or Motif. Previously only `networkPolicies`, `resourceQuotas`, `limitRanges`, `hpa`, `pdb`, and `rollingUpdate` supported this. Four additional classes are added:

| Class | YAML key | What it tunes |
|-------|----------|---------------|
| Resources | `profiles.resources` | Container CPU and memory requests/limits |
| Probes | `profiles.probes` | Probe timing — initialDelaySeconds, periodSeconds, failureThreshold, successThreshold, timeoutSeconds |
| Container Security | `profiles.containerSecurity` | Per-container securityContext — allowPrivilegeEscalation, readOnlyRootFilesystem, runAsNonRoot, capabilities |
| Pod Security | `profiles.podSecurity` | Pod-level securityContext — runAsNonRoot, runAsUser, runAsGroup, fsGroup |

```yaml
profiles:
  resources:
    - name: api-worker
      requests: { cpu: "500m", memory: "256Mi" }
      limits: { cpu: "2", memory: "1Gi" }

  probes:
    - name: slow-boot
      initialDelaySeconds: 60
      periodSeconds: 20
      failureThreshold: 5
      timeoutSeconds: 10

  containerSecurity:
    - name: strict-readonly
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      runAsNonRoot: true
      capabilities:
        drop: [ALL]

  podSecurity:
    - name: ci-runner
      runAsNonRoot: true
      runAsUser: 2000
      runAsGroup: 2000
      fsGroup: 2000
```

User-defined profiles are resolved before Orkestra built-ins. Declaring a profile with the same name as a built-in overrides it for that Katalog. `ork validate` prints a warning when a built-in is shadowed.

`operatorBox.autoscaler` does not yet support user-defined profiles — configure autoscaler behavior inline.

### kubectl: DSL block for e2e assertions

A `kubectl:` block can now sit alongside `resources:` and `commands:` in any `expect:` entry. Eleven subcommands:

| Subcommand | Generates |
|------------|-----------|
| `get` | `kubectl get <kind> <name> -o jsonpath='{<field>}'` or `--output json/yaml` |
| `logs` | `kubectl logs -n <ns> [-l <selector> \| <name> \| leaderElection] [--since=<since>]` |
| `describe` | `kubectl describe <kind> <name/selector> -n <ns>` |
| `exec` | `kubectl exec -n <ns> <pod> -- <command>` |
| `port-forward` | Port-forward + curl as one operation; lifecycle managed by the runner |
| `apply` | `kubectl apply -f <file>` or inline manifest via stdin |
| `patch` | `kubectl patch <kind> <name> --type=<merge\|strategic\|json> -p '<patch>'` |
| `delete` | `kubectl delete -f <file>` or by resource identity; `ignoreNotFound: true` available |
| `events` | `kubectl events --for=<kind>/<name> -n <ns>` |
| `auth` | `kubectl auth can-i <verb> <resource> [--as <as>]` |
| `cp` | `kubectl cp <ns>/<pod>:<src>` — copies to temp file, asserts content, cleans up |
| `top` | `kubectl top <pod\|node>` — requires metrics-server; installed automatically via Helm |

All subcommands share six assertion fields: `equals`, `notEquals`, `outputContains`, `outputNotContains`, `greaterThan`, `lessThan`. `greaterThan` and `lessThan` parse the output as `float64` — the check fails if the output is not numeric. `jq` and `yq` extraction is supported where applicable. `apply` and `patch` run before read subcommands each iteration so mutations take effect before assertions check them.

`commands[].run` now also supports `outputNotContains`.

**`leaderElection` on `port-forward` and `logs`**

Both `port-forward` and `logs` entries accept a `leaderElection` block that resolves the target pod from a Kubernetes `coordination.k8s.io/v1` Lease. In multi-replica deployments only the elected leader runs reconcilers — without this, port-forward may land on a follower returning stale state and `kubectl logs` may target the wrong pod. `leaderElection` guarantees assertions always run against the pod with authoritative state.

```yaml
kubectl:
  port-forward:
    - leaderElection:
        lease: orkestra-konductor
      namespace: orkestra-system
      port: 8080
      path: /katalog/myapp/health
      jq: state
      equals: "healthy"
  logs:
    - leaderElection:
        lease: orkestra-konductor
        namespace: orkestra-system
      outputContains: "became konductor"
```

`service` and `pod` are optional when `leaderElection` is set. For `logs`, `name`, `labelSelector`, and `leaderElection` are mutually exclusive. `lease.namespace` defaults to the entry's `namespace`.

**`kubectl.delete`**

Supports deletion by file or by resource identity. `ignoreNotFound: true` silences errors when the resource is already gone — useful in cleanup steps or after cascade deletes.

```yaml
kubectl:
  delete:
    - file: ./crd.yaml
      ignoreNotFound: true
    - kind: Pod
      name: my-pod
      namespace: default
      ignoreNotFound: true
```

**`wait:` on checkpoints and `port-forward` entries**

Each `expect:` checkpoint accepts a `wait:` field — a duration to sleep before the polling loop starts. Individual `port-forward` entries accept their own `wait:` — a duration to sleep after the connection is established but before the curl request is sent.

**Curl fix: HTTP 4xx/5xx responses no longer silently fail**

Port-forward curl calls previously used `curl -sf`, which exits non-zero on any HTTP 4xx/5xx response. Health endpoints for degraded operators return HTTP 503, causing assertions against degraded state to time out rather than report the actual body. Changed to `curl -s` — the body is always returned and assertions check content directly.

**Port-forward progress spinner**

During the polling loop, a spinner shows the URL being curled and the resolved target (`curl http://localhost:PORT/path (→ pod/holder-name)`).

**Tool pre-flight**: `ork e2e` scans the spec and installs missing tools before assertions run — `curl` (port-forward), `jq`, `yq` (apt-get / apk / brew), and `metrics-server` (Helm, with `--kubelet-insecure-tls` on kind clusters).

**`include:` for `expect:` composition**

Large test suites can be split across files. An `include:` entry is replaced in place by the checkpoints in the referenced file — position in the list determines where the expanded checkpoints appear in the run order. The file uses `expect:` as its root key.

```yaml
expect:
  - include: ./infra-ready.yaml   # expands here — runs first
  - name: Operator-specific check
    after: cr-applied
    timeout: 30s
    resources:
      - kind: MyOperator
        name: my-resource
  - include: ./cleanup.yaml       # expands here — runs last
```

`ork validate` expands all includes before reporting — the checkpoint list and count reflect the fully-expanded run order. Paths are resolved relative to the `e2e.yaml` that contains the `include:` entry.

**Validator**: `ork validate` checks all `kubectl:` blocks — required fields, mutual exclusion, at least one assertion per entry, `jq`/`yq` format consistency, `top` kind must be `pod` or `node`.

**Fixture**: `pkg/e2e/fixture/` is a living integration test with one checkpoint per subcommand. Rule: add a checkpoint when you add a subcommand.

---

## v0.7.8 — First-class NetworkPolicy, ResourceQuota, LimitRange, ClusterRole, ClusterRoleBinding; namespace-provisioner sub-pack

### New resource types

Five resource types promoted from placeholder stubs to fully managed Orkestra resources:

| Type | Scope | Notes |
|------|-------|-------|
| `NetworkPolicy` | Namespaced | Profile or explicit ingress/egress/policyTypes |
| `ResourceQuota` | Namespaced | Profile or explicit hard map |
| `LimitRange` | Namespaced | Explicit limit items |
| `ClusterRole` | Cluster | No OwnerReferences — owned via `orkestra.io/owner` label |
| `ClusterRoleBinding` | Cluster | RoleRef immutability handled (delete + recreate on change) |

### New profiles

**NetworkPolicy profiles** — set `profile:` on any networkPolicy entry:

| Profile | Effect |
|---------|--------|
| `deny-all` | Block all ingress and egress |
| `deny-all-ingress` | Block all ingress |
| `deny-all-egress` | Block all egress |
| `allow-same-namespace` | Allow ingress from pods in the same namespace |
| `allow-dns-egress` | Allow egress on port 53 (UDP + TCP) |

**ResourceQuota profiles** — set `profile:` on any resourceQuota entry:

| Profile | Pods | CPU | Memory |
|---------|------|-----|--------|
| `small` | 10 | 2 | 4Gi |
| `medium` | 20 | 4 | 8Gi |
| `large` | 50 | 8 | 16Gi |
| `xlarge` | 100 | 16 | 32Gi |

### User-defined profiles

All six profile classes now support user-defined named presets declared in the `profiles:` block at the root of a Katalog or Motif:

```yaml
profiles:
  networkPolicies:
    - name: org-deny-all
      policyTypes: [Ingress, Egress]
  resourceQuotas:
    - name: org-medium
      hard: { pods: "25", cpu: "4", memory: "8Gi" }
  limitRanges:
    - name: org-container-defaults
      limits:
        - type: Container
          default: { cpu: 500m, memory: 512Mi }
          defaultRequest: { cpu: 100m, memory: 128Mi }
  hpa:
    - name: org-conservative
      targetCPUUtilizationPercentage: "70"
  pdb:
    - name: org-at-least-one
      minAvailable: "1"
  rollingUpdate:
    - name: org-safe
      maxSurge: "1"
      maxUnavailable: "0"
```

User profiles resolve before built-ins. Motif-declared profiles merge into the importing Katalog's registry; duplicate names in the same class across Katalog and Motif are a hard error at load time. `ork validate` shows the profile count per class in Motif output.

LimitRange profiles are always user-defined — there are no built-in LimitRange presets.

### Katalog validation

`ork validate` now rejects unknown profile names for all six profile families (HPA, PDB, RollingUpdate, NetworkPolicy, ResourceQuota, LimitRange) and `fromNamespace` set without `toNamespaces` or vice versa.

### Simulate: cross-namespace copy auto-skip

Resources declaring `fromNamespace` / `toNamespaces` are automatically skipped before the fake reconciler runs. A note is printed for each skipped resource. Use `ork e2e` to verify cross-namespace copies against a real cluster.

### E2E: per-entry waits and setup-complete lifecycle

- `setup.apply` and `setup.helm` entries support an inline `wait:` list that blocks after each step
- New lifecycle event `after: setup-complete` for infrastructure assertions before the CR is applied. This is now the default when `after:` is omitted
- `spec.crd` and `spec.cr` are optional when `spec.custom.target: kubernetes` is set

### ork validate output consistency

Simulate and Motif output now use the same `●` header + structured fields format as Katalog and E2E.

### Chart: gateway PDB + self-test

Gateway PodDisruptionBudget added (`enabled: true`, `minAvailable: 1`). The Orkestra Helm chart now ships with `charts/orkestra/e2e.yaml` — a `custom.target: kubernetes` spec that installs and validates the chart itself using `ork e2e`.

### New sub-pack: `use-cases/namespace-provisioner`

```bash
ork init --pack use-cases/namespace-provisioner
```

Four progressive examples building a production namespace provisioner — a single CRD that provisions a namespace with quota, RBAC, and network policy in one apply:

| Example | What you get |
|---------|-------------|
| `01-explicit` | Hard-coded quota, ClusterRole, ClusterRoleBinding, NetworkPolicy — every resource the operator emits, fully visible |
| `02-profiles` | Same eight resources; sizes via built-in profiles (ResourceQuota: small/medium/large/xlarge, NetworkPolicy: allow-same-namespace / deny-all-egress) |
| `03-user-defined` | User-defined profiles declared in a shared Motif; each team tier references a named preset — one Motif entry = one new tier |
| `04-motif-profiles` | Org-wide policy in a single Motif file consumed by any Katalog that imports it |

The step-by-step design narrative lives in `documentation/guides/namespace-provisioner/`.

Because the operator creates ClusterRoles and ClusterRoleBindings, Orkestra automatically adds `escalate` and `bind` to the generated bundle — required by Kubernetes privilege escalation prevention. These verbs are absent from operators that manage no RBAC resources.

### ork create pattern

`ork create pattern` scaffolds a complete operator suite in one command:

```bash
ork create pattern
ork create pattern -o ./my-operator/
ork create pattern --typed
```

**Always generated:** `katalog.yaml`, `simulate.yaml`, `e2e.yaml`, `README.md`.

**Typed mode** (`--typed`, `--add-hook`, `--add-constructor`): also generates `values.yaml`, `Makefile`, and `Dockerfile` — ready to build, push, and release a runtime binary.

---

## v0.7.7 — New packs, typed group overrides, universal e2e, breaking: operatorBox.reconciler

### New pack: `from-controller-runtime`

```bash
ork init my-operator --pack from-controller-runtime
```

Eight examples tracing the migration path from an existing controller-runtime operator to Orkestra:

| Example | What you get |
|---------|-------------|
| `00-baseline` | The controller-runtime starting point |
| `01-declarative` | Zero Go. Same behaviour. Two CRDs including a Worker with `rotateAfter: 30d` token rotation |
| `02-hybrid` | Declarative + one Go hook for what templates can't express |
| `03-hooks-only` | All resources in Go, typed access to your CRD spec |
| `04-constructor-migration` | Lift the existing reconcile loop into Orkestra's constructor |
| `05-constructor-orkestra-resources` | Same constructor, Orkestra resource helpers replace manual Get/Create/Patch |
| `06-ork-migrate` | `ork migrate` rewrites controller-runtime reconcilers automatically |
| `07-all-options` | All five options in one binary via Komposer — `komposer-local.yaml` for local dev, `komposer.yaml` for OCI distribution |

The step-by-step narrative lives in `documentation/guides/migration/`.

### New pack: `ecosystem-composition`

```bash
ork init my-operator --pack ecosystem-composition
```

Seven examples building an internal developer platform on top of the tools you already run:

| Example | What you get |
|---------|-------------|
| `00-argocd` | `App` CRD → ArgoCD Application. Admission. Status propagation |
| `01-cert-manager` | `SecurityConfig` CRD → Certificate |
| `02-prometheus` | `MonitoringConfig` CRD → ServiceMonitor + PrometheusRule |
| `03-crossplane` | `Infra` CRD → Crossplane Composite Claim |
| `04-platform-stack` | All four, composed with Komposer |
| `05-policy-layer` | Shared admission motif across all CRDs. Deletion protection |
| `06-all-in-one` | Single `PlatformResource` CRD with `workloadType` discriminator routing to the right tool |

The `06-all-in-one` guide (`documentation/guides/ecosystem/07-all-in-one.md`) includes a full trade-off comparison between focused CRDs and a unified CRD.

### New pack: `resilience`

```bash
ork init my-operator --pack resilience
```

Operators that stay running — through panics, cascading failures, and degraded dependencies:

| Example | What you get |
|---------|-------------|
| `safe-reconcile` | Panic isolation in the worker pool. Nil pointer in a typed hook, caught by `safeReconcile`. Two declarative CRDs keep reconciling while the typed one degrades. |

The deep-dive lives in `documentation/concepts/operatorbox/01-reconcile-pipeline/03-panic-recovery.md`.

### Typed apiTypes group override (marketplace-ready patterns)

Typed katalogs can now be published under one API group and consumed under another. Set `apiTypes.group` in your komposer override — the generated registry uses `AddKnownTypeWithName` to register the Go type under the override group rather than the package's compiled-in `GroupVersion` constant. `apiTypes.location` is now purely the import path for the Go structs, not an implicit API identity contract.

This enables the marketplace model: pull a typed pattern from OCI, set your org's group, build, run. No source fork required.

### Patch API simplified

`PatchStatus`, `PatchFinalizers`, `PatchLabels`, `PatchAnnotations`, and `PatchSpec` no longer take an explicit `gvr schema.GroupVersionResource`. GVR is resolved internally from the object's Go type via the scheme and REST mapper — the same mechanism used by `Get`, `Create`, and `Patch`. Constructor reconcilers no longer need to reference `GroupVersionResource` constants at all.

```go
// before
r.kube.PatchStatus(ctx, obj, apiv1.GroupVersionResource, fields)

// after
r.kube.PatchStatus(ctx, obj, fields)
```

### Breaking: `operatorBox.reconciler` sub-block

`default`, `hooks`, and `constructor` have moved under an `operatorBox.reconciler:` key.

**Before:**
```yaml
operatorBox:
  default: true
  hooks:
    location: ...
    function: ...
```

**After:**
```yaml
operatorBox:
  reconciler:
    hooks:
      location: ...
      function: ...
```

Migration rules:
- **Declarative-only katalogs** (no `hooks:` or `constructor:`): delete `default: true` entirely. A nil `reconciler:` block means GenericReconciler — `default: true` is now implicit and redundant.
- **Hook katalogs**: wrap `hooks:` (and `default: true` if present) under `reconciler:`. Move `default:` and `hooks:` in by two spaces; everything else (`onCreate:`, `status:`, etc.) stays at the `operatorBox:` level.
- **Constructor katalogs**: wrap `default: false` and `constructor:` under `reconciler:`. Move both in by two spaces.

`utils.StrictUnmarshal` rejects unknown fields, so an old katalog with top-level `default:` or `hooks:` under `operatorBox` will fail at parse time with a clear error before any validation runs.

### `ork e2e` is now a universal Kubernetes test harness

`spec.customOperator: true` is replaced by `spec.custom.target: kubernetes`. The new
field names the thing being tested rather than describing what to skip.

**Supported targets:**

| Value | Status |
|-------|--------|
| `kubernetes` | Supported |
| `container` | Coming soon |

**Migration** — find and replace in all `e2e.yaml` files:

```yaml
# before
spec:
  customOperator: true

# after
spec:
  custom:
    target: kubernetes
```

`ork validate e2e` now fast-fails with a clear error on unknown target values. If
`container` is specified, it exits with "coming soon" rather than silently doing nothing.

The docs reframe `custom.target: kubernetes` as what it is: `ork e2e` with any
workload that runs on Kubernetes — operators, Helm charts, raw manifests, third-party
tools. The cluster lifecycle, assertion polling, and cleanup are Orkestra's. The
workload is yours.

New guide: `documentation/guides/e2e-universal.md` — "Test Anything That Runs in Kubernetes"

---

## v0.7.6 — Registry Guide pack, CLI UX polish, bug fixes

### Registry Guide pack

A structured, zero-to-production walkthrough of the Orkestra registry — 13 self-contained steps from consuming a published pattern on day one to automated CI publishing with GitHub Actions. Covers declarative operators, typed Go operators with hooks, multi-katalog Komposers, CRD API evolution, deprecation workflows, and the full `ork push` gate pipeline.

```bash
ork init --pack registry-guide
```

### CLI UX polish

- **`ork push`** — `--use-current` and `--cluster` flags let you reuse an existing cluster for the e2e gate, skipping cluster provisioning for faster local iteration.
- **`ork push`** on typed operators — `RuntimeVersion` is read from `go.mod` and recorded as an OCI annotation. Visible in `ork inspect` as `Runtime: vX.Y.Z`.
- **`ork pull`** — warns when the pulled typed operator was built against a different Orkestra version than the current binary.
- **`ork inspect`** — shows `Typed: ✓ hooks` and `Runtime: vX.Y.Z` for typed patterns.
- **`ork validate` / `ork simulate`** — surface a clear `TypedOperatorError` with an actionable `ork inspect <ref>` hint when a Komposer sources a registry-based typed operator.

### orkestra-action

- `template: "true"` now auto-detects `katalog.yaml` in the working directory, consistent with `validate: "true"` and `simulate: "true"`.
- Install step downloads directly from GitHub releases — no CDN redirect, resilient to network issues.
- `/ready` and `/started` health endpoints added to the dev server image.

### Bug fixes

- Probe path templates (`{{ .spec.probePath }}`) now resolve correctly in Deployment, ReplicaSet, StatefulSet, and Pod specs.
- Deployment no longer triggers a no-op update every reconcile when an HTTP probe is configured (missing `Scheme` field on `HTTPGetAction`).
- Helm bundle deploy correctly routes CRDs to their declared `metadata.namespace` — each team sees their own namespace panel in the Control Center.
- `ork inspect --versions` table columns align correctly when version strings are colorized.
- `ork inspect --versions` works without a version tag in the argument.
- Deprecated pattern row aligns correctly in `ork patterns` — `⚠` marker moved to its own column.
- Deprecation message is shown even when no migration target is set.
- Control Center version labels no longer double the `v` prefix.
- Control Center katalog panel shows the namespace-level description when a namespace filter is active.

---

## v0.7.4 — pkg/resources rename

`pkg/orkestra-registry` is renamed to `pkg/resources`.

**Who is affected:** typed-mode operators only — Go code that writes custom hooks or constructors and imports from `pkg/orkestra-registry`. Dynamic-mode operators (pure YAML) are unaffected.

**What to change:** update import paths from `pkg/orkestra-registry/<kind>` to `pkg/resources/<kind>`:

```go
// before
"github.com/orkspace/orkestra/pkg/orkestra-registry/deployments"

// after
"github.com/orkspace/orkestra/pkg/resources/deployments"
```

**Why:** `pkg/orkestra-registry` and `pkg/registry` (the OCI distribution layer) both carried the word "registry" with no relation to each other. The resource library — idempotent Create/Update/Delete/Resolve implementations for every Kubernetes resource type — is not a registry. `pkg/resources` is accurate and removes the collision. No logic changes, no API changes, no behavioral differences.

---

## v0.7.3 Simulate: standalone operation

`ork simulate` is now a self-contained command. Key changes:

**Flat imports syntax** — simulate aggregators now use a plain list, matching e2e:

```yaml
imports:
  - ./09-hooks/simulate.yaml
  - ./10-constructor/simulate.yaml
```

**`--dev-server` flag** — start the mock server before running simulate, no cluster needed:

```bash
ork simulate --dev-server
ork simulate -f simulate.yaml --dev-server
```

**Decoupled from e2e** — passing an `e2e.yaml` to `ork simulate` now returns a helpful error instead of silently falling back to op-print mode.

**Example suites** — root `simulate.yaml` aggregators added to the beginner, intermediate, and external example packs. The three external examples with `serviceUrl`/`healthCheckUrl` now have full simulate.yaml files with `expect:` assertions, all runnable with `--dev-server`.

**health-gate CRs** — `cr-healthy.yaml` and `cr-degraded.yaml` now point to `localhost:9999` instead of httpbin.org, so they work with `--dev-server` without any external network dependency.

**`ork simulate init` improvements** — the generated `simulate.yaml` now includes a commented `absent:` block (seeded with the first observed resource type) as a hint for failure-path coverage. A new `--dry-run` flag prints to stdout instead of writing the file — useful for previewing before committing.

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
- **Typed e2e documented** — READMEs for typed examples now explain `--set runtime.image.repository` / `--set runtime.image.tag` so `ork e2e` deploys the user's custom image (which contains the generated type registry) instead of the default Orkestra image.

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

- **Custom resource registry package (`pkg/resources/customresources/`)**
  New package exposing `Create`, `Update`, `DeleteIfOwned`, `Resolve`, and `ResolvedCustomResourceSpec`.
  Owner references are set automatically — deleting the parent cascades to all child custom resources without requiring any `onDelete` hook.

- **Template resolution (`pkg/resources/template/resolve_customresources.go`)**
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

- **Unified replica parsing (`pkg/resources/common/parse.go`)**
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
