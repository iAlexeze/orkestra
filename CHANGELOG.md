# Changelog

---

### Note Library Expansion & Template Safety

#### Note library — new modules

- **Quantity notes** (`parseQuantity`, `formatQuantity`, `sumQuantity`) — resource budget arithmetic using Kubernetes SI and binary suffixes; enables "give each tenant 1/N of available capacity" expressions without Go
- **Replica notes** (`readyReplicas`, `availableReplicas`, `updatedReplicas`, `desiredReplicas`, `replicasReady`) — nil-safe Deployment/ReplicaSet/StatefulSet status fields
- **Job notes** (`jobSucceeded`, `jobFailed`, `jobActive`) — batch Job lifecycle gates
- **Service notes** (`serviceClusterIP`, `serviceNodePort`, `serviceLoadBalancerIP`, `serviceLoadBalancerHost`, `endpointsReady`) — Service networking fields and endpoint readiness
- **Field notes** (`resourceName`, `resourceNamespace`, `resourceUID`, `resourceVersion`, `creationTimestamp`) — direct accessors for `metadata.*` fields present on every Kubernetes resource
- All registered in `note.Map()` and available in every template expression

#### Note library — missing notes added to `kubernetes.go`

- `phase` — `status.phase` safe accessor
- `conditionReason`, `conditionMessage` — condition detail fields by type
- `resourceExists` — nil-safe existence check
- `isTerminating` — `metadata.deletionTimestamp` check
- `generation`, `observedGeneration`, `isSynced` — controller sync tracking

#### Template safety — `.children.*` and `.cross.*` navigations replaced

Raw dot-path navigation (`{{ .children.deployment.status.readyReplicas }}`) panics when intermediate keys are absent. All occurrences across YAML examples, docs, and narrative markdown replaced with note-based equivalents:
- `{{ readyReplicas .children.deployment }}`, `{{ phase .cross.db }}`, `{{ serviceLoadBalancerHost .children.service }}`, etc.
- `field:` conditions in `when:` blocks updated to use template syntax (`field: "{{ note .children.X }}"`) — notes are the single navigation model for both `value:` and `field:` positions

#### Documentation

- `pkg/note/docs/` — complete doc index now covers all 15 note modules (previously stopped at 10)
- New docs: `11-quantity.md`, `12-replica.md`, `13-job.md`, `14-service.md`, `15-fields.md`
- Updated `09-kubernetes.md` with all previously undocumented notes
- `README.md` index updated with all new rows and `field:` usage guidance

#### Bug fixes

- **TLS secrets not deleted on CR delete** — `createTLSSecret` was setting only labels, not `OwnerReferences`. Fixed: TLS secrets now carry full owner references so Kubernetes GC deletes them with the CR
- **Namespaces not deleted on CR delete** — Kubernetes GC does not honour owner references from namespace-scoped resources (CRs) to cluster-scoped resources (Namespaces). Fixed: `runTemplateOnDelete` now calls `deleteOwnedNamespaces`, which explicitly deletes all declared Namespaces owned by the CR

#### Namespace filter — informer factory

- Three-tier namespace scoping wired into the informer factory: Tier 1 (ListerWatcher scoped), Tier 2 (pre-enqueue drop in `handleEvent`), Tier 3 (reconciler safety net)
- `NamespaceFilter` struct with `AllowedNamespaces`/`RestrictedNamespaces`, `Allows()`, `IsSingleNamespace()`, `extractNamespace` with tombstone unwrap
- `GenericClient` extended with `ListInNamespace`/`WatchInNamespace` for typed informer Tier 1 scoping
- `konstructor.go` wires filter registration and `opts.Namespace` for both dynamic and typed ListerWatchers

---

### CI/CD & Helm Chart Improvements

#### Helm Chart
- Removed unnecessary configurations after testing
- Chart now uses stripped image tag (without `v` prefix) for `appVersion` and container image tags
- Updated `values.yaml` to reference the correct image tag format

#### CI/CD Workflows – Made Fully Reusable
- All workflows now accept configurable inputs:
  - `image_tag` (stripped version) passed from a central `prepare` job
  - Repository, image names, Helm repo URL, Homebrew tap, etc.
- Removed hardcoded project names (`orkspace`, `ialexeze`, `orkestra`) – workflows are portable
- Added `prepare` job to strip `v` from Git tags and propagate `image_tag` to downstream jobs
- Standardized tag handling: Git tags keep `v`, container images and Helm charts use plain semver

#### Final Release Workflow
- Orchestrates all jobs with proper dependencies and conditionals
- Uses `prepare` to compute metadata once
- All reusable workflows called with explicit inputs (defaults applied in the called workflows)
- Added release summary job that aggregates status from all components

#### Affected Workflows
- `build-matrix.yml` (unchanged but now receives metadata via prepare)
- `build-push-images.yml` – configurable image names & registry
- `package-examples.yml` – configurable repo and project name
- `release-helm.yml` – configurable chart name, repo URL, namespace
- `sign-and-release.yml` – fully configurable (GitHub repo, container registry, Helm repo, Homebrew tap)
- `publish-homebrew.yml` – accepts tap repository and main repo
- `release-summary.yml` – uses same configurable inputs for success instructions