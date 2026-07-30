# kubectl block

The `kubectl:` block provides a structured alternative to `commands:` for the most common assertion patterns. Each subcommand maps directly to the `kubectl` command people already know — `kubectl.get`, `kubectl.logs`, `kubectl.describe`, `kubectl.exec`, `kubectl.port-forward`.

Raw `commands:` stays for anything that doesn't fit a subcommand — though `kubectl.apply`, `kubectl.delete`, and `kubectl.patch` now cover the most common mutation patterns.

---

## Structure

`kubectl:` sits alongside `resources:` and `commands:` in each `expect:` entry:

```yaml
expect:
  - name: Deployment has correct resource profile
    after: cr-applied
    timeout: 60s
    resources:
      - kind: Deployment
        name: my-service
        namespace: default
    kubectl:
      get:
        - kind: Deployment
          name: my-service
          field: .spec.template.spec.containers[0].resources.requests.cpu
          equals: 200m
```

All subcommands in a `kubectl:` block are checked in the same polling loop as `resources:` and `commands:`. All must pass for the checkpoint to pass.

### Execution order

Within a single `kubectl:` block, subcommands always execute in this fixed order regardless of how they appear in the YAML:

1. **`apply`** — create or update resources
2. **`patch`** — modify fields on existing resources
3. **`restart`** — trigger a rollout restart
4. **`scale`** — change replica count
5. **`delete`** — remove resources
6. **`get`**, **`logs`**, **`describe`**, **`exec`**, **`port-forward`**, **`events`**, **`auth`**, **`cp`**, **`top`** — assertions (read-only)

Mutations always run before assertions so that state changes take effect before they are evaluated. Within mutations, the order is create → modify → destroy — so you can `scale` a resource in the same block that later `delete`s it.

---

## Assertion fields

Every subcommand supports the same assertion fields:

| Field | Description |
|-------|-------------|
| `equals` | Output (trimmed) must exactly match this string |
| `notEquals` | Output must not exactly match this string |
| `oneOf` | Output (trimmed) must match one of the listed strings |
| `notOneOf` | Output (trimmed) must not match any of the listed strings |
| `outputContains` | Output must contain this substring |
| `outputNotContains` | Output must not contain this substring |
| `regex` | Output (trimmed) must match this RE2 regular expression (Go's `regexp` syntax) |
| `greaterThan` | Output (trimmed, parsed as a number) must be greater than this value — **strict** |
| `lessThan` | Output (trimmed, parsed as a number) must be less than this value — **strict** |
| `greaterThanOrEqual` | Output must be greater than or equal to this value |
| `lessThanOrEqual` | Output must be less than or equal to this value |
| `between` | Output must be numerically within an inclusive range. Value is `"min,max"` |
| `notBetween` | Output must be numerically outside an inclusive range. Value is `"min,max"` |
| `exists` | Output (trimmed) must be non-empty — field is present and has a value |
| `notExists` | Output (trimmed) must be empty — field is absent or unset |

Multiple assertions on the same entry all apply. Empty fields are ignored. These are evaluated with the same `Condition` operators as `when:`/`anyOf:` (see [when/anyOf conditions & Operators](../02-katalog/06-when-conditions.md#operators)), against a single synthetic `output` field holding the trimmed command output — the numeric comparisons fail if the output is not parseable as a number.

`oneOf` is useful when the expected value is one of several valid strings — for example, a status field that reflects current runtime state:

```yaml
kubectl:
  get:
    - kind: APIServer
      name: my-api
      namespace: default
      field: .status.phase
      oneOf: [Peak, Steady]
```

---

## `kubectl.get`

Generates: `kubectl get <kind> <name> -n <namespace> -o jsonpath='{<field>}'`

```yaml
kubectl:
  get:
    # jsonpath field extraction
    - kind: Deployment
      name: my-service
      namespace: default
      field: .spec.template.spec.containers[0].resources.requests.cpu
      equals: 200m

    # full JSON output with jq extraction
    - kind: ResourceQuota
      name: my-service-quota
      namespace: default
      format: json
      jq: .status.hard.pods
      equals: "10"

    # full YAML output with yq extraction
    - kind: ConfigMap
      name: my-config
      namespace: default
      format: yaml
      yq: .data.maxConnections
      outputContains: "100"
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind |
| `name` | yes | Resource name |
| `namespace` | no | Namespace. Default: `default` |
| `field` | no | jsonpath expression to extract. e.g. `.spec.replicas` |
| `format` | no | `json` or `yaml` — returns the full resource. Ignored when `field` is set |
| `jq` | no | jq expression applied to output before asserting. Requires `format: json` |
| `yq` | no | yq expression applied to output before asserting. Requires `format: yaml` |

---

## `kubectl.logs`

Generates: `kubectl logs -n <ns> [-l <selector> | <name>] [-c <container>] [--since=<since>]`

```yaml
kubectl:
  logs:
    # assert a log line exists
    - labelSelector: app=my-service
      namespace: default
      since: 30s
      outputContains: "server started on port 8080"

    # assert no error logs (JSON structured logging)
    - labelSelector: app=my-service
      namespace: default
      jq: .level
      outputNotContains: error

    # assert exact log message in a named pod
    - name: my-service-abc123
      container: sidecar
      outputContains: "config reloaded"

    # assert a log line emitted by the current leader pod
    - leaderElection:
        lease: my-operator-leader
        namespace: my-operator-system
      outputContains: "acquired leader lock"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | no | Pod name. Mutually exclusive with `leaderElection` |
| `labelSelector` | no | Label selector (e.g. `app=my-service`). Mutually exclusive with `leaderElection` |
| `leaderElection` | no | Resolve the log target from a Kubernetes Lease holder. Mutually exclusive with `name` and `labelSelector`. See [leaderElection](#leaderlection) |
| `namespace` | no | Namespace. Default: `default` |
| `container` | no | Container name. Defaults to the first container |
| `since` | no | Limit output to logs from the last duration (e.g. `30s`, `2m`) |
| `jq` | no | jq expression applied to each log line. Useful for JSON-structured logs |

> **Note** — `name`, `labelSelector`, and `leaderElection` are mutually exclusive. Exactly one must be provided.

---

## `kubectl.describe`

Generates: `kubectl describe <kind> [-n <ns>] [<name> | -l <selector>]`

Useful for asserting Kubernetes events, conditions, and resource details that don't appear in structured fields.

```yaml
kubectl:
  describe:
    # assert image was pulled successfully
    - kind: Pod
      labelSelector: app=my-service
      namespace: default
      outputContains: "Successfully pulled image"

    # assert no crash events
    - kind: Pod
      labelSelector: app=my-service
      namespace: default
      outputNotContains: "Back-off restarting failed container"
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind |
| `name` | no | Resource name. Use `labelSelector` to match by label instead |
| `labelSelector` | no | Label selector |
| `namespace` | no | Namespace. Default: `default` |

---

## `kubectl.exec`

Generates: `kubectl exec -n <ns> <pod> [-c <container>] -- <command>`

```yaml
kubectl:
  exec:
    # verify a config file was mounted correctly
    - labelSelector: app=my-service
      namespace: default
      command: [cat, /etc/config/app.conf]
      outputContains: "maxConnections=100"

    # verify a secret is accessible inside the container
    - labelSelector: app=my-service
      namespace: default
      container: app
      command: [sh, -c, "echo $DB_PASSWORD"]
      outputNotContains: ""
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | no | Pod name. Use `labelSelector` to match by label instead |
| `labelSelector` | no | Label selector. One of `name` or `labelSelector` required |
| `namespace` | no | Namespace. Default: `default` |
| `container` | no | Container name. Defaults to the first container |
| `command` | yes | Command to run as a list (no shell interpolation) |
| `jq` | no | jq expression applied to the output before asserting |
| `yq` | no | yq expression applied to the output before asserting |

---

## `kubectl.port-forward`

Opens a port-forward to a service, pod, or the elected leader of a Kubernetes Lease, makes an HTTP request via curl, and asserts the response. The runner manages the port-forward lifecycle — background start, port-open polling, curl, cleanup. No shell scripting required.

`curl`, `jq`, and `yq` are installed automatically if not present when detected in the spec.

```yaml
kubectl:
  port-forward:
    # assert via service
    - service: my-api
      namespace: default
      port: 9090
      path: /healthz
      outputContains: "ok"

    # assert a YAML API endpoint
    - service: my-api
      namespace: default
      port: 9090
      path: /config
      method: GET
      yq: .maxConnections
      outputContains: "100"

    # assert via leader election — port-forward directly to the leader pod
    - namespace: my-operator-system
      port: 8080
      path: /healthz
      leaderElection:
        lease: my-operator-leader
      outputContains: "ok"

    # authenticated POST — e.g. a gateway Apply API behind a bearer token
    - service: orkestra-gateway
      namespace: orkestra-system
      port: 8443
      path: /api/v1/apply
      method: POST
      headers:
        Authorization: "Bearer ${ORK_CI_TOKEN}"
        Content-Type: application/json
      body: '{"apiVersion":"platform.myorg.io/v1","kind":"AppRequest","metadata":{"name":"bad"},"spec":{"replicas":1}}'
      outputContains: '"accepted":false'
```

| Field | Required | Description |
|-------|----------|-------------|
| `service` | no | Service name to port-forward to. Required when `leaderElection` is not set and `pod` is not set |
| `pod` | no | Pod name to port-forward to. Alternative to `service` |
| `leaderElection` | no | Resolve the port-forward target from a Kubernetes Lease. When set, `service` and `pod` are not required. See [leaderElection](#leaderlection) |
| `namespace` | no | Namespace. Default: `default` |
| `port` | yes | Port to forward (used as both local and remote) |
| `path` | no | HTTP path to request via curl after port-forward is ready |
| `method` | no | HTTP method. Default: `GET` |
| `headers` | no | Map of request headers, e.g. `Authorization` for a token-gated endpoint. Values go through `os.ExpandEnv` (`${VAR}` syntax, same as `gateway.applyAPI.auth.tokens.token`) so a CI secret never needs to be written into the e2e file |
| `body` | no | Request body for `POST`/`PUT`/`PATCH`. Also goes through `os.ExpandEnv` |
| `wait` | no | Duration to sleep after the port-forward is ready but before sending the curl request (Go duration: `5s`, `10s`). Useful when the endpoint needs time to stabilize |
| `jq` | no | jq expression applied to the response before asserting |
| `yq` | no | yq expression applied to the response before asserting |

### `leaderElection`

Resolves the port-forward target by reading a Kubernetes Lease object and port-forwarding directly to the holder pod. This guarantees that assertions run against the process with authoritative state — not a follower that may return stale data.

```yaml
kubectl:
  port-forward:
    - namespace: my-operator-system
      port: 8080
      path: /metrics
      leaderElection:
        lease: my-operator-leader
        namespace: my-operator-system   # optional; defaults to the port-forward namespace
      outputContains: "process_start_time"
```

| Field | Required | Description |
|-------|----------|-------------|
| `lease` | yes | Name of the `coordination.k8s.io/v1` Lease object |
| `namespace` | no | Namespace of the Lease. Defaults to the port-forward `namespace` |

At runtime, the harness runs `kubectl get lease <name> -n <namespace> -o jsonpath='{.spec.holderIdentity}'` to find the current holder, then opens a port-forward to `pod/<holder>`. If the Lease has no holder yet, the step retries until the checkpoint times out.

See [Testing Leader-Led Deployments](../../concepts/e2e/06-leader-led-deployments.md) for the full picture on why this matters and when to use it.

---

## `kubectl.apply`

Applies manifests during an expect checkpoint. Use `file` to reference a path on disk or `inline` to embed the manifest directly. `kubectl apply` is idempotent so re-running inside the poll loop is safe.

Generates: `kubectl apply -f <file>` or `echo '<inline>' | kubectl apply -f -`

```yaml
kubectl:
  apply:
    # apply a file relative to the e2e.yaml directory
    - file: ./fixtures/v2-cr.yaml

    # apply an inline manifest
    - inline: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: feature-flags
          namespace: default
        data:
          v2: enabled

    # apply with a namespace override
    - file: ./fixtures/tenant-quota.yaml
      namespace: team-alpha

    # assert a rejection — e.g. an admission webhook denial — instead of
    # treating any apply failure as a broken test
    - file: ./fixtures/duplicate-domain.yaml
      exitCode: 1
      outputContains: "spec.domain must be unique"
```

| Field | Required | Description |
|-------|----------|-------------|
| `file` | no | Path to a manifest file. Relative paths resolve from the `e2e.yaml` directory. Mutually exclusive with `inline` |
| `inline` | no | Raw YAML or JSON manifest applied via stdin. Mutually exclusive with `file` |
| `namespace` | no | Namespace override for resources that don't declare one |
| `exitCode` | no | Expected exit code. Default `0` (success) — set non-zero to assert the apply must be *rejected* (e.g. an admission webhook denial) rather than treating any failure as a broken test |
| `equals`, `notEquals`, `outputContains`, `outputNotContains`, `regex`, `greaterThan`, `lessThan`, `greaterThanOrEqual`, `lessThanOrEqual`, `between`, `notBetween`, `exists`, `notExists`, `oneOf`, `notOneOf` | no | Assertions on the combined stdout+stderr — same fields and semantics as [`commands:`](03-expect.md#commands) |

Combined stdout+stderr is captured the same way regardless of `exitCode` — a webhook's denial message lands in `kubectl`'s stderr, so `outputContains` can assert on the actual rejection reason, not just that the apply failed.

---

## `kubectl.delete`

Deletes resources during an expect checkpoint. Use `file` to delete all resources in a manifest, or `kind` + `name` for a single resource. `file` and `kind`/`name` are mutually exclusive.

Generates: `kubectl delete -f <file>` or `kubectl delete <kind> <name> -n <namespace>`

```yaml
kubectl:
  delete:
    # delete all resources in a manifest
    - file: ./crd.yaml

    # delete a single resource by identity
    - kind: Pod
      name: my-pod
      namespace: default
      ignoreNotFound: true
```

| Field | Required | Description |
|-------|----------|-------------|
| `file` | one of file or kind+name | Path to a manifest file. Relative paths resolve from the e2e.yaml directory |
| `kind` | one of file or kind+name | Kubernetes resource kind |
| `name` | one of file or kind+name | Resource name |
| `namespace` | no | Namespace to target. Defaults to `default` |
| `ignoreNotFound` | no | Silences errors when the resource does not exist |

---

## `kubectl.patch`

Patches a Kubernetes resource in-place. Useful for triggering state transitions — driving a state machine forward, updating a field to test a reconciler's reaction, etc.

Generates: `kubectl patch <kind> <name> -n <namespace> --type=<type> -p '<patch>'`

```yaml
kubectl:
  patch:
    # merge patch (default) — scale up replicas
    - kind: Deployment
      name: my-service
      namespace: default
      patch: '{"spec":{"replicas":3}}'

    # strategic merge patch — update a container image
    - kind: Deployment
      name: my-service
      namespace: default
      type: strategic
      patch: |
        spec:
          template:
            spec:
              containers:
              - name: app
                image: my-service:v2

    # json patch — set a specific field by path
    - kind: MyResource
      name: my-resource
      namespace: default
      type: json
      patch: '[{"op":"replace","path":"/spec/phase","value":"active"}]'
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind |
| `name` | yes | Resource name |
| `namespace` | no | Namespace. Default: `default` |
| `type` | no | Patch strategy: `merge` (default), `strategic`, or `json` |
| `patch` | yes | Patch content as a YAML or JSON string |

---

## `kubectl.events`

Lists Kubernetes events for a specific resource and asserts the output. Useful for verifying that the operator emitted expected events or that no error events occurred.

Generates: `kubectl events --for=<kind>/<name> -n <namespace>`

```yaml
kubectl:
  events:
    # assert the operator emitted a Reconciled event
    - kind: Deployment
      name: my-service
      namespace: default
      outputContains: Reconciled

    # assert no BackOff events occurred
    - kind: Pod
      name: my-service-abc123
      namespace: default
      outputNotContains: BackOff
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Kubernetes resource kind |
| `name` | yes | Resource name |
| `namespace` | no | Namespace. Default: `default` |

---

## `kubectl.auth`

Checks permissions via `kubectl auth can-i` and asserts the result (`yes` or `no`). Useful for verifying that the operator created the correct RBAC resources — ServiceAccounts, ClusterRoles, ClusterRoleBindings.

Generates: `kubectl auth can-i <verb> <resource> [-n <namespace>] [--as <as>]`

```yaml
kubectl:
  auth:
    # assert the operator's service account can list pods
    - verb: list
      resource: pods
      namespace: default
      as: system:serviceaccount:default:my-operator
      equals: "yes"

    # assert it cannot delete secrets (principle of least privilege)
    - verb: delete
      resource: secrets
      namespace: default
      as: system:serviceaccount:default:my-operator
      equals: "no"
```

| Field | Required | Description |
|-------|----------|-------------|
| `verb` | yes | Action to check: `get`, `list`, `create`, `delete`, `patch`, etc. |
| `resource` | yes | Kubernetes resource type: `pods`, `deployments`, `secrets`, etc. |
| `namespace` | no | Namespace scope. Omit for cluster-scoped checks |
| `as` | no | User or service account to impersonate. Use `system:serviceaccount:<ns>:<name>` form |

---

## `kubectl.cp`

Copies a file out of a running container and asserts its content. Resolves the pod by name or label selector, copies to a temporary path, applies assertions, and cleans up. Supports `jq` and `yq` extraction for structured file content.

Generates: `kubectl cp <ns>/<pod>:<src> <tempfile>`

```yaml
kubectl:
  cp:
    # assert a generated config file contains the expected value
    - labelSelector: app=my-service
      namespace: default
      src: /etc/config/app.conf
      outputContains: "maxConnections=100"

    # assert a JSON file field via jq
    - labelSelector: app=my-service
      namespace: default
      src: /etc/config/settings.json
      jq: .database.host
      equals: "postgres.default.svc"

    # assert from a named pod with a specific container
    - name: my-service-abc123
      container: app
      namespace: default
      src: /tmp/generated-cert.pem
      outputContains: "BEGIN CERTIFICATE"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | no | Pod name. Use `labelSelector` to match by label instead |
| `labelSelector` | no | Label selector. One of `name` or `labelSelector` required |
| `namespace` | no | Namespace. Default: `default` |
| `container` | no | Container name. Defaults to the first container |
| `src` | yes | Path inside the container to copy from |
| `jq` | no | jq expression applied to the file content before asserting |
| `yq` | no | yq expression applied to the file content before asserting |

---

## `kubectl.top`

Queries live CPU and memory usage via `kubectl top` and asserts the output. Requires `metrics-server`; the runner installs it automatically via Helm when any `top` entry is present. On kind clusters, `--kubelet-insecure-tls` is set automatically.

Generates: `kubectl top <kind> [-n <namespace>] [<name> | -l <selector>] [--containers]`

```yaml
kubectl:
  top:
    # assert both probe pods appear in metrics output
    - kind: pod
      namespace: default
      labelSelector: app=my-service
      outputContains: my-service

    # assert a specific pod's metrics row is present
    - kind: pod
      name: my-service-abc123
      namespace: default
      outputContains: my-service-abc123

    # per-container breakdown
    - kind: pod
      namespace: default
      labelSelector: app=my-service
      containers: true
      outputContains: app

    # assert node metrics are available
    - kind: node
      outputContains: cpu
```

| Field | Required | Description |
|-------|----------|-------------|
| `kind` | yes | Resource type: `pod` (or `pods`) or `node` (or `nodes`) |
| `name` | no | Pod or node name. Omit to list all |
| `labelSelector` | no | Filter pods by label. Applies to pods only |
| `namespace` | no | Namespace. Applies to pods only. Default: `default` |
| `containers` | no | Show per-container metrics (`--containers`). Pods only |

---

## Tool pre-flight

When `ork e2e` loads the spec, it scans for tool requirements and installs missing ones before assertions run:

| Tool | Required when | Installed via |
|------|--------------|---------------|
| `curl` | Any `port-forward` entry has a `path` | apt-get / apk / brew |
| `jq` | Any entry has a `jq:` field | apt-get / apk / brew |
| `yq` | Any entry has a `yq:` field | apt-get / apk / brew |
| `metrics-server` | Any `top` entry is present | Helm (`metrics-server/metrics-server`) |

Installation is automatic. A spinner shows progress. On kind clusters, metrics-server is installed with `--kubelet-insecure-tls` automatically.

---

## Combining with `resources:` and `commands:`

All three blocks work together in the same checkpoint:

```yaml
expect:
  - name: Service is healthy and correctly configured
    after: cr-applied
    timeout: 90s

    resources:
      - kind: Deployment
        name: my-service
        namespace: default
        ready: true

    kubectl:
      get:
        - kind: Deployment
          name: my-service
          field: .spec.template.spec.containers[0].resources.requests.cpu
          equals: 200m
      logs:
        - labelSelector: app=my-service
          outputContains: "ready to serve"
          outputNotContains: FATAL

    commands:
      - run: "curl -sf http://my-service:8080/healthz"
        outputContains: ok
```

---

## kubectl.restart

Trigger a rollout restart of a Deployment, StatefulSet, or DaemonSet. By default waits for the rollout to complete — the expect step's `timeout` governs how long.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | yes | Resource kind: `Deployment`, `StatefulSet`, or `DaemonSet` |
| `name` | string | yes | Resource name |
| `namespace` | string | no | Namespace. Defaults to `default` |
| `ready` | bool | no | Wait for rollout to complete. Defaults to `true` |

```yaml
kubectl:
  restart:
    - kind: Deployment
      name: orkestra-gateway
      namespace: orkestra-system
```

---

## kubectl.scale

Set the replica count on a Deployment, StatefulSet, or ReplicaSet. By default waits for the rollout to complete — the expect step's `timeout` governs how long.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `kind` | string | yes | Resource kind: `Deployment`, `StatefulSet`, or `ReplicaSet` |
| `name` | string | yes | Resource name |
| `namespace` | string | no | Namespace. Defaults to `default` |
| `replicas` | int | yes | Desired replica count |
| `ready` | bool | no | Wait for rollout to complete. Defaults to `true` |

```yaml
kubectl:
  scale:
    - kind: Deployment
      name: my-app
      namespace: default
      replicas: 3
```

---

→ Back: [06-discovery.md](06-discovery.md) | [Schema index](index.md)
