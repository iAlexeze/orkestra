# 02 — Katalog: WebApp operator

Build a Katalog that exposes a `WebApp` CRD backed by the `web-service` motif. This is the first step where you author a pattern and deploy it yourself.

---

## Files

| File | Purpose |
|------|---------|
| `crd.yaml` | WebApp CRD schema |
| `katalog.yaml` | Katalog that imports the web-service motif |
| `cr.yaml` | Sample CR — no Ingress |
| `cr-with-ingress.yaml` | Sample CR — Ingress created when `spec.host` is set |
| `simulate.yaml` | Gate: Deployment + Service created, Ingress absent when host empty |
| `simulate-ingress.yaml` | Gate: Ingress created when host is non-empty |
| `e2e.yaml` | Full cluster test |

---

> **Before you start:** If `ORK_REGISTRY` is not set, export it before proceeding (see [01-motifs](../01-motifs/README.md#push-to-the-registry)).

---

## Template, validate and simulate

All offline — no cluster required:

```bash
ork template
ork validate
ork simulate
ork simulate -f simulate-ingress.yaml
```

`ork template` expands the motif import and shows exactly which Kubernetes resources the katalog will produce — useful for reviewing what the web-service motif contributes before validating.

The two simulate files prove the `when:` condition on the Ingress. 

- [simulate.yaml](simulate.yaml) uses [cr.yaml](cr.yaml) — no `spec.host`, no Ingress:

```text
  Cycle 1:
    + deployments/my-webapp
    + services/my-webapp-svc        ← no ingress
    ~ status/my-webapp
  ...
  ✓ create deployments/my-webapp (cycle 1)
  ✓ create services/my-webapp-svc (cycle 1)
  PASS
```

- [simulate-ingress.yaml](simulate-ingress.yaml) uses [cr-with-ingress.yaml](cr-with-ingress.yaml) — `spec.host` set, Ingress appears:

```text
  Cycle 1:
    + deployments/my-webapp
    + services/my-webapp-svc
    + ingresses/my-webapp-ingress   ← ingress created
    ~ status/my-webapp
  ...
  ✓ create ingresses/my-webapp-ingress (cycle 1)
  PASS
```

Both pass before anything touches a cluster.

---

## Deploy

**1. Apply the CRD**

```bash
kubectl apply -f crd.yaml
```

**2. Preview permissions**

`ork validate --full` shows every RBAC rule the bundle will contain — per CRD, per component — before anything is generated. No cluster required. Review this before proceeding.

```bash
ork validate --full -f katalog.yaml
```

**3. Generate and apply the bundle**

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

**4. Install Orkestra**

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

**5. Create a WebApp**

```bash
kubectl apply -f cr.yaml
kubectl get webapp my-webapp -n default
```

---

## What to observe

The Ingress condition in the web-service motif:

```yaml
when:
  - field: "{{ .inputs.host }}"
    exists: true
```

With [cr.yaml](cr.yaml) applied (no `spec.host`) — matching what simulate.yaml asserted — the condition fails and no Ingress is created:

```bash
kubectl get ingress -n default
# No resources found in default namespace.
```

Apply [cr-with-ingress.yaml](cr-with-ingress.yaml) and the Ingress appears on the next reconcile cycle:

```bash
kubectl apply -f cr-with-ingress.yaml
kubectl get ingress -n default
```

---

## Push to the registry

The katalog currently imports the motif by local path — correct for development, but not for publishing:

```yaml
imports:
  - motif: ../01-motifs/web-service/motif.yaml
```

Before pushing, inspect the published motif to get its exact OCI reference:

```bash
ork inspect web-service:v1.0.0 --motif
```

The output includes a ready-to-copy import snippet:

```text
To import:
  imports:
    - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
```

Update the import in [katalog.yaml](katalog.yaml) to that reference:

```yaml
imports:
  - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
```

Also update the `author:` field in [katalog.yaml](katalog.yaml) to your name or org before publishing.

This is the developer workflow: local paths while building, registry references when publishing. Once this katalog is published, any future version imports directly from the registry — the local path is only ever used during development.

Now push. If you are in a new terminal or starting from this example directly, set the registry first:

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

Simulate and e2e run automatically before the artifact is published. The gate results are written as OCI annotations — visible to any consumer via `ork inspect webapp-operator:v1.0.0`.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [03-katalog-cache](../03-katalog-cache/README.md) — stateful operator with auto-rotating credentials
