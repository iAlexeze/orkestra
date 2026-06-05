# How it works

`ork e2e` runs a fixed lifecycle every time. The phases always execute in the same order, and teardown always runs — whether the test passed, failed, or was interrupted.

---

## The lifecycle

```text
1.  Cluster ready
        kind cluster created  — or existing cluster used (--use-current / --cluster)

2.  Dependencies checked
        kind, helm, kubectl available in PATH

3.  CRD applied
        spec.crd applied to the cluster

4.  OCI imports pre-pulled
        any motif or registry imports in the Katalog pulled before bundle generation

5.  Bundle generated and applied
        ork generate bundle → ConfigMap + RBAC applied

6.  Setup runs
        setup.apply  — manifests applied in order
        setup.helm   — prerequisite charts installed
        setup.wait   — blocks until each listed resource is ready

7.  Orkestra installed
        helm upgrade --install orkestra
        controlCenter disabled automatically

8.  Orkestra ready
        waits for the runtime Deployment to have available replicas

9.  Expectations run
        for each expect block in order:
          after: cr-applied  → CR applied (once), then checkpoint polled
          after: cr-deleted  → CR deleted (once), then checkpoint polled

10. Imports run (if any)
        each imported E2E runs in sequence in the same or a fresh cluster

11. Cleanup
        for borrowed clusters: CR → Orkestra → bundle → setup → CRDs deleted
        for owned kind clusters: kind cluster deleted
        kubectl context restored to the context that was active before the run
```

---

## Cluster modes

**Default — kind cluster (no flags):**
`ork e2e` creates a temporary kind cluster named `ork-e2e` (or the name in `spec.cluster.name`), runs the full lifecycle, and deletes the cluster when done. Teardown is guaranteed: the cluster is always deleted even if the test fails, unless `--keep-cluster` is set.

```bash
ork e2e               # creates and deletes a kind cluster
ork e2e --keep-cluster  # creates but does not delete — inspect the cluster after
```

**Existing cluster (`--use-current`):**
Runs against whatever kubectl context is currently active. Nothing is created or deleted at the cluster level. Orkestra and all applied resources are cleaned up in reverse order when the test finishes.

```bash
ork e2e --use-current
```

**Named context (`--cluster`):**
Switches to a specific kubeconfig context, runs the test, then restores the original context. Same cleanup behavior as `--use-current`.

```bash
ork e2e --cluster kind-staging
```

---

## Teardown order

Teardown always runs for clusters the test does not own. The order reverses the apply order to avoid dangling dependencies:

```text
CR deleted           (if not already deleted by a cr-deleted checkpoint)
Orkestra uninstalled (helm uninstall)
Bundle deleted       (RBAC, ConfigMap, Namespace)
Setup files deleted  (in reverse apply order)
CRDs deleted         (cascades to any remaining CRs of those types)
```

For owned kind clusters, the cluster deletion handles all of this in one shot.

---

## Orkestra installation flags

`ork e2e` always passes `--set controlCenter.enabled=false` to Helm. The Control Center is not needed in automated tests and adds startup time.

Additional flags are passed with `--set`:

```bash
# Each --set becomes --set key=value
ork e2e --set "runtime.image.tag=dev" --set "gateway.replicas=3"
```

When the Katalog declares a `gateway:` block, `--set gateway.enabled=true` is appended automatically.

If Orkestra is already installed in the cluster, `ork e2e` syncs the runtime instead of reinstalling — the bundle ConfigMap was updated, and the runtime needs a rollout to pick it up.

---

→ Next: [Writing a test](02-writing-a-test.md)
