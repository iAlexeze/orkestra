# 03 — Notes via Motifs

Inline notes are great for single-Katalog expressions. When your team owns many Katalogs — or when you want to publish notes for external consumers — put them in a Motif instead.

**What you learn:** packaging notes into a Motif, importing them via `spec.imports`, and the OCI publishing path that lets any team consume them.

**Builds on:** [02 — User-Defined Notes](../02-user-defined/README.md)

---

## Why a Motif?

A Motif is a reusable Orkestra bundle. When you declare notes inside a Motif, you get:

- **A single source of truth** — one change updates every Katalog that imports it
- **OCI distribution** — publish the Motif as an artifact so any team can consume it:
  ```bash
  export ORK_MOTIFS_REGISTRY=registry.example.com/platform/notes
  ork push ./motifs
  ```
- **Versioned contracts** — consumers pin a tag; you iterate without breaking them

See the [Motif publishing guide](https://orkestra.sh/docs/orkestra-registry/motifs/#publishing) for the full OCI workflow.

---

## This example

`motifs/motif.yaml` declares two notes shared across the team:

```yaml
notes:
  - name: fullImage
    expression: "{{ .spec.image }}:{{ .spec.tag | default \"latest\" }}"

  - name: appLabel
    expression: "{{ .metadata.namespace }}-{{ .metadata.name }}"
```

`katalog.yaml` imports the Motif via `spec.imports` and adds one inline note:

```yaml
spec:
  imports:
    - motif: ./motifs/motif.yaml

notes:
  - name: serviceHost
    expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

All three notes are now available in every template expression in the Katalog:

```yaml
onReconcile:
  deployments:
    - name: "{{ appLabel }}"      # → default-my-service
      image: "{{ fullImage }}"    # → nginx:1.25
```

```yaml
status:
  fields:
    - path: host
      value: "{{ serviceHost }}"  # → my-service.default.svc.cluster.local
    - path: image
      value: "{{ fullImage }}"    # → nginx:1.25
```

---

## Step 1 — Validate

Validate and inspect the merged note set — built-ins plus the Motif and inline notes — in one step:

```bash
ork validate --notes
```

## Step 2 — Simulate (no cluster needed)

```bash
ork simulate
```

## Step 3 — Run

```bash
ork run
```

Watch `IMAGE`, `HOST`, and `PHASE` populate:

```bash
kubectl get workloads -w
```

Note the `IMAGE` column shows `nginx:1.25` (from `fullImage`). Now apply the default CR — `tag` is omitted so `fullImage` supplies `latest`:

```bash
kubectl apply -f cr-default.yaml
kubectl get deployments
```

Both `default-my-service` and `default-my-service-default` appear. The second has image `nginx:latest`.

## Step 4 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the Deployment and status fields — `image`, `host`, and `label` all computed by notes whether they were declared in the Motif or the Katalog.

## Step 5 — E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

**Next:** [04 — Komposer Override](../04-komposer/README.md) — override a Motif note at the Komposer level without touching the Katalog.

**OCI publishing:** [https://orkestra.sh/docs/orkestra-registry/motifs/#publishing](https://orkestra.sh/docs/orkestra-registry/motifs/#publishing)
