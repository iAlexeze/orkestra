# 02 — User-Defined Notes

Built-in notes cover the common cases. When you need an expression that is specific to your CRD — a computed hostname, a qualified name — declare it once in a `notes:` block and call it like a built-in.

**What you learn:** declaring an inline user-defined note and writing its result to a status field.

**Builds on:** [01 — Built-in Notes](../01-built-in/README.md)

---

## Declaring a user-defined note

Add a `notes:` block to the Katalog:

```yaml
notes:
  - name: serviceHost
    expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

Then call it anywhere built-in notes are available:

```yaml
status:
  fields:
    - path: host
      value: "{{ serviceHost }}"
```

The expression is a Go template evaluated against the CR. It can call built-in notes and other user-defined notes.

---

## This example

`katalog.yaml` declares one inline note — `serviceHost` — and writes it to `status.host` so you can read the computed value directly on the CR.

The expression uses two built-in fields: `.metadata.name` and `.metadata.namespace`. When the CR `my-service` is created in the `default` namespace, `serviceHost` evaluates to `my-service.default.svc.cluster.local`.

---

## Step 1 — Validate

```bash
ork validate
```

Inspect the custom note:

```bash
ork notes -f katalog.yaml
```

## Step 2 — Simulate (no cluster needed)

```bash
ork simulate
```

## Step 3 — Run

```bash
ork run
```

Watch `HOST` (from `serviceHost`) populate and `PHASE` flip to `Ready`:

```bash
kubectl get workloads -w
```

Now apply the default CR — `tag` is omitted so the expression supplies `latest`:

```bash
kubectl apply -f cr-default.yaml
kubectl get deployments
```

`my-service-default` will show `1/1`. Check the image:

```bash
kubectl get deployment my-service-default -o wide
```

The `IMAGE` column will show `nginx:latest` — the `default` note filled the missing tag.

## Step 4 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the Deployment and the status fields — `image` and `host` populated by the user-defined notes on every reconcile.

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

**Next:** [03 — Motifs](../03-motifs/README.md) — distribute notes across Katalogs by publishing them as a Motif.

**Full concept guide:** [https://orkestra.sh/docs/concepts/notes](https://orkestra.sh/docs/concepts/notes)
