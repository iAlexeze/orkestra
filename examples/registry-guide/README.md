# Registry Guide

A structured, zero-to-production walkthrough of the Orkestra registry. Each directory
is a self-contained step. Work through them in order or jump to the concept you need.

## What you will build

By the end of this guide you will have:

- Consumed a published pattern in three commands
- Written your own Motifs and Katalogs from scratch
- Published a pattern with simulate and e2e gates
- Composed multiple Katalogs into a single Komposer
- Upgraded a live pattern and rolled it back
- Deprecated a pattern and migrated consumers to a replacement
- Built a typed Go operator with hooks for complex business logic
- Combined declarative and typed katalogs in one Komposer
- Automated the full pipeline with GitHub Actions

---

## Before you begin

**Registry access** — steps that push or pull patterns require a container registry. Authenticate once:

```bash
docker login ghcr.io
```

Set your registry paths (replace `myorg` with your actual org or username):

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
```

**Cluster** — steps that deploy to Kubernetes need a cluster. If you don't have one, create a local kind cluster:

```bash
ork create cluster
```

This provisions a single-node kind cluster named `ork-playground` and sets your kubeconfig context to it. Skip this if you already have a cluster pointed at by `kubectl`.

---

## Steps

| Step | What it covers |
|------|----------------|
| [00-consume](./00-consume/README.md) | Pull a registry pattern and create a CR — day one |
| [01-motifs](./01-motifs/README.md) | Write reusable building blocks: web-service, data-store, platform-admission |
| [02-katalog-api](./02-katalog-api/README.md) | Build a katalog from the web-service motif; simulate gate |
| [03-katalog-cache](./03-katalog-cache/README.md) | Build a katalog from the data-store motif; stateful + credentials |
| [04-katalog-platform](./04-katalog-platform/README.md) | Add admission policies to an existing katalog via motif |
| [05-komposer](./05-komposer/README.md) | Compose multiple katalogs into one Runtime binary |
| [06-pattern-zoo](./06-pattern-zoo/README.md) | Mix OCI registry patterns, local katalogs, and Helm sources |
| [07-upgrade](./07-upgrade/README.md) | Publish v1.1.0, upgrade a live CR, verify zero downtime |
| [08-bad-actor](./08-bad-actor/README.md) | Audit trail: trace who pushed what and when |
| [09-deprecation](./09-deprecation/README.md) | Mark a pattern deprecated, migrate consumers to the replacement |
| [10-hooks-katalog](./10-hooks-katalog/README.md) | Typed Go operator: hooks, generate registry, build, publish |
| [11-typed-komposer](./11-typed-komposer/README.md) | Combine declarative and typed katalogs in one Komposer |
| [12-ork-action](./12-ork-action/README.md) | GitHub Actions: automate validate → simulate → push |

---

## Key concepts

### Namespace

Every Katalog declares `metadata.namespace` — a logical tenant scope used by the Control Center to group CRDs into separate panels. It is not a Kubernetes namespace.

```yaml
metadata:
  name: webapp-operator
  namespace: api-team
```

Omitting it defaults to `"default"`, which means all CRDs appear in a single panel. Teams that share a runtime should always declare a namespace so the Control Center renders each team's operators independently.

See [Multi-tenancy](https://orkestra.sh/docs/concepts/operatorbox/multi-tenancy/) for the full model including `clusterName` and `crossAccess` controls.

---

### Tags

Every Motif and Katalog declares a `tags:` list under `metadata:`. Tags are indexed
by the registry and surfaced in `ork patterns --tag <tag>`. Use them to make
patterns discoverable without knowing the exact name.

```yaml
metadata:
  tags:
    - web
    - deployment
    - ingress
```

### `resources.onCreate` and `once: true` secrets

Orkestra reconciles resources on every cycle. For auto-generated secrets (passwords,
API keys, tokens) this would regenerate a new random value on every loop — breaking
every connection that holds the old credential.

Two mechanisms prevent this:

1. **`resources.onCreate`** — resources declared here are only processed at first
   reconcile. They are never updated on subsequent cycles (no drift correction).

2. **`once: true`** on the secret — the reconciler skips the secret if it already
   exists in the cluster, regardless of which reconcile phase it is in.

Together they guarantee: generate once, rotate on schedule.

```yaml
resources:
  onCreate:         # only runs at first reconcile
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true  # skip if secret already exists
        rotateAfter: "30d"
        data:
          password: "{{ randomAlphanumeric 24 }}"
```

### Simulate gates

Every pattern in this guide ships with a `simulate.yaml`. Running
`ork simulate -f simulate.yaml` executes the reconcile loop locally — no cluster
needed. Use it as the first gate before pushing to a registry.

### E2E gates

`e2e.yaml` runs the Runtime against a real cluster (kind by default). It is the
second gate: simulate proves logic, e2e proves behaviour against real Kubernetes APIs.

---

## Troubleshooting

**Push fails with 403 permission_denied**

```
push failed: pushing: failed to perform "Push" on destination:
POST "https://ghcr.io/v2/ghcr.io/katalog/webapp-operator/blobs/uploads/":
response status code 403: denied: permission_denied: create_package
```

Your token does not have `write:packages` scope, or the package does not exist yet and your account does not have permission to create it. On GitHub: go to **Settings → Developer settings → Personal access tokens** and ensure `write:packages` is checked. Then re-authenticate:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

---

**Push resolves to the wrong registry**

If `ORK_REGISTRY` or `ORK_MOTIFS_REGISTRY` is not set, `ork` falls back to the default registry. Set both before pushing or pulling:

```bash
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
export ORK_REGISTRY=ghcr.io/myorg/katalogs
```

Run `ork patterns` afterwards to confirm the command is reading from the right registry.

---

**CI: motif pull fails with 403 during ork push**

```
loading motif "oci://ghcr.io/myorg/motifs/web-service:v1.1.0": pull failed: 403 denied
```

`ork push` pulls all motif dependencies before validating. In CI (GitHub Actions), the runner must have read access to every motif the katalog imports. GHCR packages are private by default.

Two options:

1. **Make the motif packages public** — go to `https://github.com/myorg?tab=packages`, open each motif package, and set visibility to public. No token changes needed.

2. **Pass a token with `read:packages` scope** — the `orkspace/orkestra-action` forwards `registry-password` to `docker login` before any pull. Set it to a token with read access:

   ```yaml
   - uses: orkspace/orkestra-action@v1
     with:
       registry-url: ghcr.io/${{ github.repository_owner }}/katalogs
       registry-username: ${{ github.actor }}
       registry-password: ${{ secrets.GITHUB_TOKEN }}
   ```

   `GITHUB_TOKEN` has `read:packages` for packages in the same repository by default. For motifs in a separate repository, use a personal access token with `read:packages` stored as a repository secret.

---

**Typed operator e2e fails with image pull error**

`ork push` runs the e2e gate before publishing. The e2e spins up a real cluster and starts your operator pod — which means the cluster must be able to pull the runtime image. If the image is in a private registry, the kind cluster has no credentials and the pod stays in `ImagePullBackOff`.

Make the runtime image public before running `ork push`, or pre-load it into the kind cluster and reuse it:

```bash
kind load docker-image ghcr.io/myorg/database-operator:v1.0.0 --name <cluster-name>
ork push --use-current
```

Alternatively, push the image first, set the package visibility to public in GitHub (`ghcr.io` → package settings → Change visibility), then run `ork push`.

---

**Docker Hub: nested paths not supported**

Docker Hub only supports `docker.io/<username>/<image>` — you cannot use multi-level paths like `docker.io/myorg/motifs/web-service`. Set both registries to your Docker Hub username directly:

```bash
export ORK_MOTIFS_REGISTRY=docker.io/myusername
export ORK_REGISTRY=docker.io/myusername
```

The pattern name becomes the image name: `docker.io/myusername/web-service:v1.0.0`. To separate motifs from katalogs on Docker Hub, prefix the pattern name at push time or use a different username per registry type.

---

## Running the examples

All commands in each step's README are meant to be run from within that step's directory. `cd` into it first:

```bash
cd registry-guide/02-katalog-api
ork template
ork validate
```

From the registry-guide root you can also run with explicit paths:

```bash
# Validate a katalog
ork validate -f 02-katalog-api/katalog.yaml

# Run the simulate gate (no cluster needed)
ork simulate -f 02-katalog-api/simulate.yaml

# Authenticate with the registry (standard Docker credential store)
docker login ghcr.io

# Push — Orkestra reads name and version from metadata in the pattern files
ork push ./02-katalog-api/

# Speed up the E2E gate during local iteration by reusing your existing cluster
# (ork push is intended for production publishing — always use a clean cluster there)
ork push ./02-katalog-api/ --use-current
ork push ./02-katalog-api/ --cluster ork-playground

# Start the Runtime
ork run -f 02-katalog-api/komposer.yaml

# Create a CR instance
kubectl apply -f 02-katalog-api/cr.yaml
```