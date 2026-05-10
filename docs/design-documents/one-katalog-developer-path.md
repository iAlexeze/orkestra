# One Katalog — The Developer Path

*Orkestra Project — May 2026*

---

## The model

One Katalog. One CRD entry (ConfigMap). One operatorBox.

Each application contributes its resources — Deployment, Service, HPA, PDB,
ServiceAccount, Role, RoleBinding — as concrete entries inside the single
`onReconcile` block. Each app has its own ConfigMap CR in its own namespace.
The one CRD entry watches all registered app namespaces.

```
~/.orkestra/deploy/
  state.json            ← deployed apps, images, namespaces, katalog hash
  katalog.yaml          ← one Katalog, one CRD, all app resources inline

app/.orkestra/
  motif.yaml            ← app's resource template (has {{ .metadata.* }} etc.)
  app.yaml              ← app's ConfigMap CR in app-ns

frontend/.orkestra/
  motif.yaml            ← frontend's template
  app.yaml              ← frontend's ConfigMap CR in frontend-ns
```

There is no `motifs/` directory. The central Katalog carries all resolved
resources directly in `onReconcile`. No imports. No file references.

---

## Why inline — not imports

Orkestra bundles the Katalog as a ConfigMap in the cluster. File `imports:`
are resolved at parse time on the host, not at runtime inside the cluster.
A Katalog that imports local paths cannot survive a restart because the
cluster-side runtime has no access to `~/.orkestra/`. Resources are therefore
resolved and embedded verbatim when `ork doctor deploy` writes the Katalog.

---

## The central Katalog

```yaml
# ~/.orkestra/deploy/katalog.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: orkestra-developer
  createdBy: orkdoctor
  projects:
    app:
      language: python
      port: "8080"
      namespace: app-ns
      currentImage: docker.io/myorg/app:abc123
    frontend:
      language: node
      port: "3000"
      namespace: frontend-ns
      currentImage: docker.io/myorg/frontend:def456

security:
  deletionProtection:
    enabled: true

spec:
  crds:
    platform:
      apiTypes:
        kind: ConfigMap
      labelSelector:
        ork.io/platform: developer
      allowedNamespaces: ["app-ns", "frontend-ns"]

      operatorBox:
        onReconcile:
          serviceAccounts:
            - name: "app"
              namespace: "app-ns"
              reconcile: true
            - name: "frontend"
              namespace: "frontend-ns"
              reconcile: true

          roles:
            - name: "app"
              namespace: "app-ns"
              rules:
                - apiGroups: [""]
                  resources: ["configmaps"]
                  resourceNames: ["app"]
                  verbs: ["get", "watch"]
              reconcile: true
            - name: "frontend"
              namespace: "frontend-ns"
              rules:
                - apiGroups: [""]
                  resources: ["configmaps"]
                  resourceNames: ["frontend"]
                  verbs: ["get", "watch"]
              reconcile: true

          roleBindings:
            - name: "app"
              namespace: "app-ns"
              roleRef: "app"
              subjects:
                - kind: ServiceAccount
                  name: "app"
                  namespace: "app-ns"
              reconcile: true
            - name: "frontend"
              namespace: "frontend-ns"
              roleRef: "frontend"
              subjects:
                - kind: ServiceAccount
                  name: "frontend"
                  namespace: "frontend-ns"
              reconcile: true

          deployments:
            - name: "app"
              namespace: "app-ns"
              image: "docker.io/myorg/app:abc123"
              resourceProfile: "burst"
              port: "8080"
              serviceAccountName: "app"
              envFrom:
                - secretRef: "app-secrets"
                - configMapRef: "app-config"
              reconcile: true
            - name: "frontend"
              namespace: "frontend-ns"
              image: "docker.io/myorg/frontend:def456"
              resourceProfile: "small"
              port: "3000"
              serviceAccountName: "frontend"
              reconcile: true

          services:
            - name: "app-svc"
              namespace: "app-ns"
              port: "8080"
              reconcile: true
            - name: "frontend-svc"
              namespace: "frontend-ns"
              port: "3000"
              reconcile: true

          hpas:
            - name: "app-hpa"
              namespace: "app-ns"
              scaleTargetRef:
                name: "app"
              minReplicas: "2"
              maxReplicas: "10"
              reconcile: true
            - name: "frontend-hpa"
              namespace: "frontend-ns"
              scaleTargetRef:
                name: "frontend"
              minReplicas: "1"
              maxReplicas: "5"
              reconcile: true

          pdbs:
            - name: "app-pdb"
              namespace: "app-ns"
              minAvailable: "1"
              reconcile: true
            - name: "frontend-pdb"
              namespace: "frontend-ns"
              minAvailable: "1"
              reconcile: true
```

One CRD. One `onReconcile` block. Each app's resources are adjacent entries.
`allowedNamespaces` grows as apps are added. `metadata.projects` carries
per-app display data for the Control Center.

---

## Each app has its own ConfigMap CR

```yaml
# app/.orkestra/app.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app
  namespace: app-ns
  labels:
    ork.io/platform: developer   # watched by the CRD entry
```

```yaml
# frontend/.orkestra/app.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: frontend
  namespace: frontend-ns
  labels:
    ork.io/platform: developer
```

The ConfigMap is the trigger. When `ork doctor deploy` applies it, Orkestra
reconciles every resource in `onReconcile` whose name matches the ConfigMap's
`metadata.name`. Because `allowedNamespaces` scopes reconciliation, the `app`
ConfigMap only creates resources in `app-ns` and the `frontend` ConfigMap only
creates resources in `frontend-ns`.

---

## How the reconciler works

The one CRD entry watches all ConfigMaps labelled `ork.io/platform: developer`
across all `allowedNamespaces`. When the `app` ConfigMap is applied, the
reconciler fires for `app`. When the `frontend` ConfigMap is applied, the
reconciler fires for `frontend`. Each fires independently.

Because all resources in `onReconcile` carry concrete values (no `{{ }}`), the
reconciler applies them directly. No template evaluation at runtime. The Katalog
is plain YAML.

---

## Namespace management — outside the Katalog

Namespaces are cluster-scoped. They cannot be owned by a namespaced
ConfigMap, so Kubernetes garbage collection never deletes them. They are
not inside the `onReconcile` block.

`ork doctor deploy` creates namespaces directly with `kubectl`:

```go
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: app-ns
  labels:
    ork.io/platform: developer
EOF
```

Each namespace is applied before the ConfigMap CR and before the Katalog.
Idempotent. Infrastructure concern, separate from operator-managed resources.

---

## Resolution at deploy time

When `ork doctor deploy` runs for an app:

1. Reads the app's `motif.yaml` template (has `{{ .metadata.name }}` etc.)
2. Substitutes `{{ .metadata.name }}` → `app`, `{{ .metadata.namespace }}` →
   `app-ns` at generation time using the app name from `app.yaml`
3. Merges the resolved resources into the `onReconcile` block of the central
   Katalog alongside any previously deployed apps
4. Writes the updated `metadata.projects` block with per-app display data
5. Hashes the Katalog and compares with `state.json`
6. If the hash changed (new app, new image): applies the Katalog and restarts
   Orkestra so the new CRD entry and `allowedNamespaces` take effect
7. Applies the app's ConfigMap CR to trigger reconciliation
8. Waits for the Deployment to become ready (two-phase: poll for existence up
   to 2 min, then rollout status up to 5 min)

On a re-deploy (same app, new image): only the `image:` field changes in
`onReconcile`, and the Deployment drift-corrects on the next reconcile cycle.

---

## State file

`~/.orkestra/deploy/state.json` is the local source of truth between deploys.
It survives reboots and is the only persistent record of what's deployed.

```json
{
  "clusterContext": "kind-orkestra",
  "katalogHash": "a3f9b2...",
  "projects": {
    "app": {
      "name": "app",
      "namespace": "app-ns",
      "currentImage": "docker.io/myorg/app:abc123",
      "previousImage": "docker.io/myorg/app:abc000",
      "katalogPath": "/home/user/.orkestra/deploy/katalog.yaml",
      "deployedAt": "2026-05-10T14:32:00Z",
      "port": "8080",
      "language": "python"
    },
    "frontend": { ... }
  }
}
```

`katalogHash` is the SHA-256 of `katalog.yaml`. A hash mismatch (new app, any
structural change) triggers an Orkestra restart. Image-only re-deploys that
produce no hash change skip the restart.

---

## ork doctor commands

```bash
# Initialise an app directory (creates .orkestra/motif.yaml and app.yaml)
ork doctor init

# Deploy one or more apps to the cluster
ork doctor deploy
ork doctor deploy --no-ha        # no HPA and PDB
ork doctor deploy --no-secure    # no deletionProtection
ork doctor deploy --clean        # rebuild katalog from state, remove stale apps

# Show live status: cluster, deployed apps, Orkestra runtime health
ork doctor status
```

`ork doctor deploy` is the primary command. It reads the current app directory,
resolves its motif, merges into the central Katalog, and drives the full
deploy-and-wait flow.

`ork doctor status` reads `state.json` and the live cluster to show:

```
Cluster:   kind-orkestra
Orkestra:  healthy  (runtime v0.9.1)

  app        app-ns      docker.io/myorg/app:abc123       deployed 2h ago
  frontend   frontend-ns docker.io/myorg/frontend:def456  deployed 5m ago
```

---

## --no-ha and --no-secure

`--no-ha` omits the HPA and PDB entries for that app's resources in
`onReconcile`. The Katalog simply has no `hpas:` or `pdbs:` entries for
that app.

`--no-secure` omits the `security:` block from the top-level Katalog:

```yaml
# with --no-secure: no security block in katalog.yaml
spec:
  crds:
    platform:
      ...
```

Both flags are evaluated at generation time. They affect what is written to
`katalog.yaml`, not what the runtime enforces.

---

## sleep and resourceProfile

Any resource in `onReconcile` can carry:

```yaml
deployments:
  - name: "app"
    resourceProfile: "burst"   # expands to CPU/memory requests+limits
    sleep: "2s"                # artificial delay before Create/Update/Delete
```

`resourceProfile` expands at katalog load time into a full
`resources.requests / limits` block. Profiles: `tiny`, `small`, `medium`,
`large`, `burst`, `steady`, `compute-heavy`, `memory-heavy`.

`sleep` fires at runtime before each Create, Update, or Delete call for that
resource. Works on all 18 resource types. Useful for:

- Autoscaler stress testing (fill the queue, verify scale-out)
- Chaos engineering (simulate slow cloud APIs per resource type)
- Queue depth and P95 latency validation

---

## Control Center — developer view

When `metadata.createdBy: orkdoctor` is set, the Control Center switches to a
developer view for that Katalog. No operator terms. No CRD. No reconciler. Just
applications.

**Home — "Your Applications":**
One card per entry in `metadata.projects`. Each card shows the app name,
language badge, image tag, and internal cluster URL. Nothing else.

**App card:**
```
app                                   [python]
docker.io/myorg/app:abc123
http://app-svc.app-ns.svc.cluster.local:8080
```

The URL is computed from `projects.<name>.port` and `projects.<name>.namespace`.
No API call needed — the data is in the Katalog response.

**Route:** `/katalog/<katalog-name>` → branches on `createdBy == "orkdoctor"`
→ renders `dev_apps.html` instead of the standard `katalog.html`.

---

## --notify-me

```bash
ork doctor init --notify-me
```

Adds a `notification:` block to the central Katalog using SMTP or Slack
credentials detected in `.env`. The developer's git `user.email` is the
default recipient. Notifications are a Katalog-level concern — not per-app.

---

## What is not here

- No Komposer. Deployment tracking uses `state.json` directly.
- No `motifs/` directory. Resolved resources are embedded in the Katalog.
- No status fields on the ConfigMap CR. Orkestra writes no status.
- No `{{ }}` in the written Katalog. All expressions are resolved before write.
- No custom CRD. The platform CRD is a standard Kubernetes ConfigMap.
- No imports in the Katalog. Import paths do not survive cluster-side restart.
