# ArgoCD

## App → Application

`00-argocd` wraps ArgoCD. Your team creates an `App` CR. Orkestra creates an ArgoCD `Application`. ArgoCD syncs the repository. The developer never sees the Application spec.

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/00-argocd
```

---

## The mapping

```text
App CR (internal)          ArgoCD Application (ecosystem)
─────────────────          ──────────────────────────────
spec.repo              →   spec.source.repoURL (with https:// prefix)
spec.path              →   spec.source.path
spec.branch            →   spec.source.targetRevision
spec.targetNamespace   →   spec.destination.namespace
                       →   spec.destination.server: https://kubernetes.default.svc
                       →   spec.project: default
                       →   spec.syncPolicy.automated.prune: true
                       →   spec.syncPolicy.automated.selfHeal: true
```

The defaults (`project: default`, `syncPolicy`, `destination.server`) are enforced by the Katalog — the developer does not set them.

---

## What the Katalog enforces

**Admission** — an `App` without `spec.labels.team` is denied at apply time:

```bash
kubectl apply -f cr-denied.yaml
# Error: admission denied: All apps must declare a team label (spec.labels.team)
```

The ArgoCD Application is never created. The validator fires before the reconciler runs.

**Status propagation** — the Katalog's `status.fields` read the ArgoCD Application's live state from `.children.customs` and write it back onto the `App` CR:

```bash
kubectl get app my-webapp -o jsonpath='{.status}'
# {"syncStatus":"Synced","health":"Healthy"}
```

**Deletion protection** — the `App` CR cannot be deleted until protection is disabled. An `argocd/v1alpha1 Application` deletion does not cascade to ArgoCD — ArgoCD owns its own cleanup.

---

## Try it

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/00-argocd
# Follow steps in README
```

---

## `hasStatus` on custom resource entries

```yaml
onCreate:
  custom:
    - kind: Application
      hasStatus: false   # exclude from .children.* context
```

When `hasStatus: false`, the created resource is excluded from the `.children.customs` context that the `status.fields` templates read. Set it when the CRD does not expose a status subresource and you want to avoid empty template values or discovery overhead.

- `true` — include in children context, attempt status reads
- `false` — exclude entirely
- omitted — auto-detect via API discovery at runtime

→ [01 — cert-manager](./02-cert-manager.md) — same pattern, certificates instead of GitOps.
