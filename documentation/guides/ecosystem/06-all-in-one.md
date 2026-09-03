# All-in-One — Single CRD, All Ecosystem Tools

The earlier examples define one CRD per ecosystem tool: `App` for ArgoCD, `SecurityConfig` for cert-manager, `MonitoringConfig` for Prometheus, `Infra` for Crossplane. Four CRDs, four schemas, four admission rule sets.

The alternative: one `PlatformResource` CRD with a `spec.workloadType` discriminator. The same CRD handles all four integrations. Conditions in the Katalog route each CR to the right ecosystem resource.

---

## The CRD

```yaml
spec:
  workloadType: app | cert | monitoring | infra
  team: platform

  # app fields
  repo: github.com/myorg/services
  path: apps/webapp
  branch: main
  targetNamespace: production

  # cert fields
  domain: api.myorg.io
  issuer: letsencrypt-prod
```

One resource. One schema. One place to look up what fields are available.

---

## The Katalog

```yaml
onCreate:
  custom:
    - apiVersion: argoproj.io/v1alpha1
      kind: Application
      when:
        - field: spec.workloadType
          equals: app
      # ... ArgoCD spec ...

    - apiVersion: cert-manager.io/v1
      kind: Certificate
      when:
        - field: spec.workloadType
          equals: cert
      # ... cert-manager spec ...
```

Each item in `custom:` has a `when:` block. Orkestra evaluates the condition on each reconcile. Only the matching resource is created.

---

## The motif

The `platform-admission` motif enforces organisation-wide admission rules across all workload types from one import:

```yaml
imports:
  - motif: ../motifs/platform-admission/motif.yaml
    with:
      allowedRegistry: "ghcr.io/myorg/"
      maxReplicas: "10"
```

The motif's validation and mutation rules apply to `PlatformResource` regardless of workload type. Motifs centralise the rules so they do not need to be repeated across four separate katalogs.

---

## Try it

```bash
cd ecosystem-composition/05-all-in-one
ork validate
kubectl apply -f crd.yaml
kubectl apply -f cr-app.yaml
kubectl apply -f cr-cert.yaml
```

---

## Trade-offs

The choice between focused CRDs and a single unified CRD depends on your organisation's priorities. Both approaches are supported. Orkestra delivers either way.

### Focused CRDs (one per ecosystem tool)

**Advantages:**
- Clear domain boundary — `App` can only create ArgoCD resources; no accidental cross-routing
- Separate admission rules per domain — `SecurityConfig` rules do not interfere with `App` rules
- Independent versioning — the `App` CRD schema can evolve independently of `MonitoringConfig`
- Clearer RBAC — you can grant a team access to `App` without granting access to `Infra`
- Smaller cognitive surface per CRD — a developer creating a cert does not see infra fields

**Trade-offs:**
- Four schemas to maintain, four Katalogs, four sets of tests
- Developers need to know which CRD to use for which purpose
- Status surfaces differently on each CRD — `App.status.syncStatus`, `SecurityConfig.status.certReady`, etc.

**Best fit:** Platform teams that want strict domain separation, independent lifecycle per tool, or per-domain RBAC.

---

### Single unified CRD

**Advantages:**
- One schema, one place — `kubectl get platformresources` shows everything
- Shared admission rules via one motif import — organisation-wide policies enforced without duplication
- Single status shape — all resources surface `workloadType`, `team`, `phase` consistently
- Developers learn one CRD regardless of what they are deploying

**Trade-offs:**
- All workload types must fit the same CRD schema — field collision risk for deeply different workloads
- The `when:` conditions must be maintained as workload types grow
- A single error in the Katalog can affect all workload types at once
- Harder to grant fine-grained access — you cannot give a team `App` access without also giving them `PlatformResource` access for all types

**Best fit:** Smaller platforms, early-stage IDP builds, or organisations that prioritise a single interface over domain separation.

---

Both patterns compose with Komposer, both support `ork e2e`, both publish to an OCI registry via `ork push`. The decision is about your organisation's boundaries — not Orkestra's capabilities.
