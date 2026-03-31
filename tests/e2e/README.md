# End-to-End Tests

E2E tests verify Orkestra running against a **real Kubernetes cluster**. They deploy
the operator binary, apply CRDs and custom resources, and assert that the expected
Kubernetes resources appear and that the health and metrics endpoints respond correctly.

## Prerequisites

| Tool | Purpose |
|------|---------|
| `kind` | Creates a local test cluster |
| `kubectl` | Applies manifests and reads resource state |
| `go` | Builds the `ork` binary |
| `jq` | Parses JSON from the health endpoint |
| `curl` | Hits health and metrics endpoints |

Install `kind`: https://kind.sigs.k8s.io/docs/user/quick-start/

## Running

### Single scenario

```bash
make test-e2e
```

This runs all three scenarios in sequence (`website`, `activation`, `dependencies`).

To run one scenario manually:

```bash
./tests/e2e/run.sh website
```

### What `run.sh` does

1. Creates a `kind` cluster named `orkestra-test`
2. Builds `ork` into `/tmp/ork`
3. Executes `./tests/e2e/<scenario>/test.sh`
4. Deletes the cluster on exit (even on failure — via `trap`)

The cluster is **ephemeral**: a fresh cluster for every run, cleaned up automatically.

## Scenarios

### `website`

Tests the canonical Website CRD lifecycle:

1. Installs `examples/website/website-crd.yaml`
2. Starts Orkestra with `examples/website/website-katalog.yaml`
3. Applies `examples/website/website-cr.yaml`
4. Asserts: `Deployment/test-website` exists
5. Asserts: `Service/test-website-svc` exists
6. Asserts: health endpoint returns `{ "healthy": true }`
7. Asserts: `/metrics` contains `controller_reconcile_total`

### `activation`

Tests the CRD-missing activation lifecycle:

1. Starts Orkestra with a Katalog that references a CRD not yet installed
2. Verifies the operator stays running but reports unhealthy
3. Installs the missing CRD
4. Verifies the operator activates and reports healthy

### `dependencies`

Tests dependency-ordered startup:

1. Installs a multi-CRD Katalog with explicit `dependsOn` edges
2. Applies CRs in reverse dependency order
3. Verifies dependent CRs only reconcile after their dependencies are healthy

## Writing a new E2E scenario

1. Create a new directory under `tests/e2e/`:

   ```
   tests/e2e/my-scenario/
   └── test.sh
   ```

2. Write `test.sh` following this structure:

   ```bash
   #!/bin/bash
   set -e

   echo "Testing: My Scenario"

   # Install CRD
   kubectl apply -f examples/my-crd/crd.yaml

   # Start Orkestra
   /tmp/ork run --katalog examples/my-crd/katalog.yaml &
   ORK_PID=$!
   sleep 5

   # Apply CR and assert outcomes
   kubectl apply -f examples/my-crd/cr.yaml
   sleep 5
   kubectl get myresource test -n default

   # Assert health
   curl -sf localhost:8080/katalog/my-crd/health | jq -e '.healthy == true'

   kill $ORK_PID
   echo "My Scenario passed"
   ```

3. Register the scenario in `Makefile`:

   ```makefile
   test-e2e:
       ./tests/e2e/run.sh website
       ./tests/e2e/run.sh activation
       ./tests/e2e/run.sh dependencies
       ./tests/e2e/run.sh my-scenario   # add this line
   ```

## Debugging a failing E2E test

**Cluster stays up after failure**: `run.sh` uses `trap ... EXIT` so the cluster is
always deleted. To keep it alive for debugging, remove the `trap` line and run the
scenario directly via `./tests/e2e/<scenario>/test.sh`.

**Operator logs**: the operator runs in the background (`&`). Redirect its output to a
log file to capture errors:

```bash
/tmp/ork run --katalog examples/website/website-katalog.yaml > /tmp/ork.log 2>&1 &
```

**Resource not appearing**: increase the `sleep` durations in `test.sh`. The defaults
(5 s) are conservative for a fresh `kind` cluster; slow CI machines may need more.

**Timeout on cluster creation**: `kind create cluster` can be slow on first run while
it pulls container images. Subsequent runs use the local image cache and are faster.

## CI

E2E tests run in the `test-e2e` CI job, which:
- Only runs after both `test-unit` and `test-integration` pass
- Uses a `kind` cluster provisioned by the CI runner
- Has a 10-minute timeout per scenario

E2E tests are **not** run on every PR by default — they require explicit opt-in via
the `run-e2e` label on the pull request. This keeps PR feedback fast while ensuring
full coverage before merge.
