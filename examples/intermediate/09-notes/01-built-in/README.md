# 01 — Built-in Notes

Every `{{ }}` expression in a Katalog is evaluated by Orkestra's note engine before resources are applied. Notes are pure functions — the same input always produces the same output, and nil or missing fields never panic.

**What you learn:** what notes are, the three most common patterns — reading live cluster state, supplying a fallback, computing names.

**Builds on:** [08 — State Machine](../../08-state-machine/README.md)

---

## What is a note?

A note is a named function you call inside a template expression. Orkestra ships with a library of built-in notes covering Kubernetes resource queries, string manipulation, time arithmetic, and more.

```yaml
value: "{{ allReplicasReady .children.deployments }}"
```

`allReplicasReady` reads the live Deployment object and returns `"true"` when all pods are ready, `"false"` otherwise. Without notes, computing this would require a custom controller.

## Browsing notes

List every available note:

```
$ ork notes
DOMAIN        NAME                  DESCRIPTION
──────        ────                  ───────────
collections   asList                Convert input to []interface{}. Accepts native slice, YAML list str...
collections   asMap                 Convert input to map[string]interface{}. Accepts native map, YAML m...
conditional   default               Return val if non-empty, otherwise return def. "Empty" means nil,...
hpa           hpaAtMax              Returns true when currentReplicas >= maxReplicas — the HPA has ...
replica       allReplicasReady      Return true when status.readyReplicas == spec.replicas...
...
```

Filter by domain:

```
$ ork notes --domain strings
DOMAIN    NAME           DESCRIPTION
──────    ────           ───────────
strings   camelToKebab   Convert CamelCase or PascalCase to kebab-case...
strings   concat         Join any number of strings together with no separator...
strings   contains       Return true if the string contains the substring...
strings   join           Join a slice of strings into a single string with a separator...
strings   replace        Replace all occurrences of old with new in s...
```

Search by keyword:

```
$ ork notes search replicas
Found 37 note(s) matching "replicas":

DOMAIN        NAME                   DESCRIPTION
──────        ────                   ───────────
conditional   default                Return val if non-empty, otherwise return def...
hpa           hpaCurrentReplicas     Returns status.currentReplicas as int64...
hpa           hpaDesiredReplicas     Returns status.desiredReplicas as int64...
replica       allReplicasReady       Return true when status.readyReplicas == spec.replicas...
```

Inspect a specific note:

```
$ ork notes show allReplicasReady
────────────────────────────────────────────────────────────────
  allReplicasReady
────────────────────────────────────────────────────────────────
  Domain:      replica
  Description: Return true when status.readyReplicas == spec.replicas. The canonical
               rollout-complete gate. Returns true when scaled to zero (desired=0 and ready=0).
  Keywords:    replica, rollout, ready, deployment, statefulset, gate, complete

  Example:
    when:
      - field: "{{ allReplicasReady .children.deployment }}"
        equals: "true"
────────────────────────────────────────────────────────────────
```

Read the full concept guide: [Concepts → Notes](https://orkestra.sh/docs/concepts/notes)

---

## This example

The `website` CRD creates a Deployment and writes two status fields:

```
ready  → "true" / "false" (live query via allReplicasReady)
phase  → "Pending" / "Ready" (gated on the ready value)
```

### Three patterns shown

**1. Live cluster state**

```yaml
- path: ready
  value: "{{ allReplicasReady .children.deployment }}"
```

Runs after every reconcile. The result changes as pods start.

**2. Fallback value**

```yaml
replicas: "{{ default .spec.replicas 1 }}"
```

Returns `spec.replicas` if set; returns `1` if the field is absent or zero.

**3. Computed name**

```yaml
name: "{{ .metadata.name }}"
```

The template context is the full CR object. `.metadata.name`, `.spec.*`, `.status.*` are all available.

---

## Step 1 — Validate

```bash
ork validate
```

## Step 2 — Simulate (no cluster needed)

```bash
ork simulate
```

## Step 3 — Run

```bash
ork run
```

Watch `READY` flip from `false` to `true` as the Deployment comes up.

```bash
kubectl get websites -w
```

Now apply the default CR — `replicas` is omitted so the `default` note supplies `1`:

```bash
kubectl apply -f cr-default.yaml
kubectl get deployments
```

`my-website-default` will show `1/1` — one replica, supplied by the `default` note, not the CR.

## Step 4 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the Deployment and status fields — including `replicas` and `image` computed by notes on every reconcile.

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

## Next

[02 — User-Defined Notes](../02-user-defined/README.md) — name your own expressions and reuse them across the Katalog.
