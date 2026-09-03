# Multi-cluster routing

One gateway instance can route intents to multiple Kubernetes clusters. The
runtime on each cluster reconciles whatever Custom Resources arrive there — it
has no knowledge of the gateway's topology decisions. The gateway is the only
component that knows where each intent goes.

## The problem it solves

A gateway handles intents from many sources: a developer using the Control
Center, a CI pipeline through the API, a Slack command, a GitHub push. These
sources do not all need the same cluster. Staging applies should not land in
prod. A regional deployment should reach the right geographic cluster.

Without multi-cluster routing, every intent lands on the single cluster the
gateway runs on. With it, the gateway can route by environment field, by token
identity, by target alias, or by any other value available in the intent at
apply time.

## How it works

Register clusters in `gateway.clusters`:

```yaml
gateway:
  clusters:
    prod:
      endpoint: https://prod.internal:6443
      secretRef:
        name: prod-credentials
        key: kubeconfig
    staging:
      endpoint: https://staging.internal:6443
      tokenRef:
        name: staging-sa-token
        key: token
      caRef:
        name: staging-ca
        key: ca.crt
```

Point CRD entries at a cluster:

```yaml
spec:
  crds:
    widget:
      serve:
        enabled: true
        cluster: staging        # default — all targets use staging
        target:
          prod-release:
            cluster: prod       # this target always goes to prod
```

At startup the gateway builds one Kubernetes client per registered cluster. At
apply time, `serve.cluster` and `target.cluster` are resolved to a cluster name,
and the CR is written there.

## Static routing vs template routing

**Static** — a fixed cluster name, validated at `ork validate` time:

```yaml
serve:
  cluster: staging
```

If `staging` is not in `gateway.clusters`, validation fails with:
```text
✗ spec.crds.widget.serve.cluster: cluster "staging" is not defined in gateway.clusters
```

**Template** — an expression resolved against the intent at apply time. The full
resolver context is available: `.request.*`, `.metadata.*`, and any user-defined
note functions:

```yaml
serve:
  cluster: '{{ if eq .request.env "prod" }}prod{{ else }}staging{{ end }}'
```

Template expressions are validated at `ork validate` time for parse correctness
and function existence, but name resolution is deferred to apply time. If a
template resolves to a name that is not registered, the intent is rejected at
apply time.

## Target alias override

`target.cluster` overrides `serve.cluster` for a specific named target. This is
the mechanism for directing specific intents to specific clusters without
changing the default:

```yaml
serve:
  cluster: staging            # default for all unspecified targets
  target:
    primary:
      primary: true
      cluster: prod           # primary target always lands on prod
    preview:
      cluster: staging        # explicit — same as default, but documented
    regional-eu:
      cluster: '{{ if eq .request.region "eu" }}eu-prod{{ else }}prod{{ end }}'
```

## Local cluster fallback

When `serve.cluster` and `target.cluster` are both absent, the intent goes to
the local cluster — the one the gateway runs on. This is unchanged behaviour.

## Read path behaviour

GET requests for resources and schema support a `?cluster=<name>` query parameter
that routes the request to the named registered cluster:

```bash
# Read a resource from a specific cluster
curl /api/v1/resources/AppRequest/default/payments-api?cluster=prod \
  -H "Authorization: Bearer $TOKEN"

# Get the schema for a target on a specific cluster
curl /api/v1/schema?target=app&cluster=staging \
  -H "Authorization: Bearer $TOKEN"
```

When `?cluster` is omitted, the gateway reads from the local cluster. When the
cluster name is not registered, the gateway returns a 404.

## Onboarding a new cluster

Use `ork clusters bootstrap` to provision the access the gateway needs on a
target cluster without manual credential steps:

```bash
ork clusters bootstrap --context kind-prod --name prod
```

This creates on the **target cluster**:
- `ServiceAccount` `kube-system/orkestra-gateway`
- `ClusterRole` `orkestra-gateway` — scoped to exactly the CRDs in your katalog
- `ClusterRoleBinding`
- Long-lived SA token `Secret`

And on the **gateway cluster** (current context):
- A credential `Secret` containing the token and CA cert

The ClusterRole covers only the resources your operators declare — no wildcards.

After running bootstrap, paste the printed `gateway.clusters` block into your
katalog and run `ork validate` to confirm.

## Declarative deployment (generate)

`ork clusters bootstrap` is the interactive path — connect, provision, done. For
GitOps and production deployments, generate the RBAC files declaratively instead:

```bash
ork generate rbac -f katalog.yaml
# or, for the full install bundle:
ork generate bundle -f katalog.yaml
```

When `gateway.clusters` is configured, these commands produce one file per cluster
in addition to the main output:

```text
rbac.yaml                  ← apply to local (gateway) cluster
gateway-prod-rbac.yaml     ← apply to prod cluster
gateway-staging-rbac.yaml  ← apply to staging cluster
```

Each per-cluster file contains a ServiceAccount, ClusterRole, and
ClusterRoleBinding in `kube-system`. The ClusterRole rules are scoped to the CRDs
that statically route to that cluster — the same minimal rule set that `ork clusters
bootstrap` creates live.

After writing each file, `ork generate rbac` prints where to apply it:

```text
  Apply to cluster "prod" (https://prod.internal:6443):
    kubectl apply -f gateway-prod-rbac.yaml

  Apply to cluster "staging" (https://staging.internal:6443):
    kubectl apply -f gateway-staging-rbac.yaml
```

**Template-routed CRDs** — when `serve.cluster` is a Go template expression, the
target cluster is unknown at generate time. Those CRDs appear in every cluster's
RBAC file and a warning is printed:

```text
warning: template-routed CRD(s) added to all cluster RBAC files: Widget
  Remove rules for clusters that should not have access.
```

Inspect the generated files and remove rules for clusters that the template cannot
actually route to.

**SA namespace** — the ServiceAccount defaults to `kube-system` (always exists on
target clusters). If your cluster policy disallows resources in `kube-system`, pass
`--sa-namespace <ns>` to `ork clusters bootstrap` to use a different namespace. For
the generated files, edit the `namespace:` field in the ServiceAccount and
ClusterRoleBinding subjects accordingly.

## Validating cluster configuration

**Offline** — checks credential forms and serve references without connecting:

```bash
ork clusters validate
ork clusters validate --full    # also shows CRD routing per cluster
```

**Online** — reads credentials from the management cluster, connects to each
remote cluster, and verifies the CRDs are installed:

```bash
ork clusters check
ork clusters check --clusters prod,staging    # subset only
ork clusters check --context my-mgmt-ctx     # specific context for reading secrets
```

## Security

The gateway requires only the permissions it was bootstrapped with — the exact
GVRs from `serve`-enabled CRD entries. It does not need cluster-admin, does
not need access to Secrets on the target cluster, and does not need access to
any namespace-scoped resources outside of what the katalog declares.

Credential Secrets live in the gateway cluster's namespace. Target clusters
hold the ServiceAccount and token Secret in `kube-system`. No credentials
cross from target cluster to gateway cluster automatically — the bootstrap
command creates them once; rotation is a future concern.

## Where to go next

- [`gateway.clusters` schema](../../reference/schema/02-katalog/24-gateway-clusters.md)
- [`serve.cluster` field](../../reference/schema/02-katalog/20-serve.md)
- [`target.cluster` field](../../reference/schema/02-katalog/21-serve-nested-spec.md)
