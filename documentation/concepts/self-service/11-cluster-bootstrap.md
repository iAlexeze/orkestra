# Cluster bootstrap

Bootstrapping a remote cluster — creating a scoped ServiceAccount, extracting its
token, and storing the credential somewhere the gateway can read it — is something
every platform engineer has done many times: in Terraform, in bash, by hand through
`kubectl`. `ork clusters bootstrap` automates it from a single point.

The tool was built to support Orkestra's multi-cluster gateway, but it works for any
system that needs a least-privilege ServiceAccount and token on a remote cluster.

## What bootstrap does

```text
Target cluster                          Gateway cluster
──────────────────────────────          ───────────────────────────
ServiceAccount  (kube-system)
ClusterRole     (scoped to CRDs)
ClusterRoleBinding
Secret          (long-lived token)  →   Secret (token + CA cert)
```

Bootstrap connects to the target cluster, provisions the access objects, extracts the
token, and stores it in the gateway cluster so the gateway can authenticate to the
target at apply time.

## Orkestra path

When bootstrapping for Orkestra, the ClusterRole rules are derived automatically from
the katalog's serve-enabled CRDs — one rule per API group, exact resource names, no
wildcards:

```bash
ork clusters bootstrap --context kind-prod --name prod -f katalog.yaml
```

After the run, bootstrap prints a `gateway.clusters` block to paste directly into
the katalog:

```yaml
gateway:
  clusters:
    prod:
      endpoint: https://127.0.0.1:6443
      tokenRef:
        name: orkestra-prod
        namespace: default
        key: token
      caRef:
        name: orkestra-prod
        namespace: default
        key: ca.crt
```

## Generic path (non-Orkestra)

Bootstrap works without a katalog. Supply the ClusterRole rules in the config file,
or omit them entirely to provision only the ServiceAccount and token:

```bash
ork clusters bootstrap --config cluster-config.yaml --no-hint
```

`--no-hint` suppresses the Orkestra snippet and prints only the Secrets that were
created — useful for ArgoCD, Flux, or any other consumer.

## Bootstrapping multiple clusters at once

Use a config file to bootstrap all target clusters in one command:

```yaml
# cluster-config.yaml
clusters:
  - name: staging
    context: kind-ork-multi-2

  - name: prod
    context: kind-ork-multi-3
    sa-namespace: restricted-ns      # optional, default: kube-system
```

```bash
ork clusters bootstrap --config cluster-config.yaml
```

Bootstrap runs each entry in order and prints the `gateway.clusters` block for all of
them.

### Config file with explicit rules (generic path)

```yaml
clusters:
  - name: prod
    context: kind-prod
    rules:
      - apiGroups: ["apps"]
        resources: ["deployments", "deployments/status"]
        verbs: ["get", "list", "create", "update", "patch", "delete"]
```

When `rules` is absent, ClusterRole and ClusterRoleBinding are skipped — only the
ServiceAccount and token Secret are created. The caller is responsible for applying
RBAC separately.

## Validating a config file

Check the config file without connecting to any cluster:

```bash
ork clusters bootstrap --validate cluster-config.yaml
```

```text
✓ bootstrap config valid (2 clusters)
  staging  →  kind-ork-multi-2
  prod     →  kind-ork-multi-3
```

Invalid verbs and missing required fields are caught here before any cluster is
touched.

## SA namespace

By default the ServiceAccount and token Secret are created in `kube-system`, which
always exists. For clusters with restricted admission policies on `kube-system`, pass
a different namespace:

```bash
ork clusters bootstrap --context kind-prod --name prod --sa-namespace platform-system
```

Or set it per entry in the config file via `sa-namespace`. The namespace must already
exist — bootstrap does not create it.

## Idempotency

Bootstrap is safe to re-run. The ClusterRole is updated to reflect the current
katalog (useful after adding a new CRD to `serve:`), existing token Secrets are
reused, and the credential Secret in the gateway cluster is updated if it already
exists.

## See also

- [`ork clusters bootstrap` reference](../../reference/cli/clusters.md#ork-clusters-bootstrap)
- [Multi-cluster routing](10-multi-cluster-routing.md)
- [Bootstrap config schema](../../reference/schema/02-katalog/25-bootstrap-config.md)
