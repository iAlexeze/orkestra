# Notes

Notes are the template vocabulary of an Orkestra Katalog — pure functions available inside every `{{ }}` expression. They let you reason about CR field values, not just copy them.

```yaml
status:
  fields:
    - path: image
      value: "{{ fullImage }}"          # user-defined note
    - path: ready
      value: "{{ allReplicasReady .children.deployments.api }}"  # built-in note
```

**Contract:** notes are pure (same input → same output), safe (nil/empty input never panics), stateless (no I/O, no side effects).

---

## Two kinds of notes

| Kind | Where defined | Scope |
|------|--------------|-------|
| Built-in | Orkestra standard library | Every Katalog |
| User-defined | `notes:` block on a Katalog or Motif | This Katalog only |

Built-in notes cover the common cases: string manipulation, Kubernetes resource queries, time arithmetic, semver comparison. The [full reference](../../reference/orkestra-notes/index.md) lists every built-in by domain.

User-defined notes let teams name an expression once and reuse it everywhere. They sit alongside built-ins in the FuncMap and can call each other and any built-in.

---

## User-defined notes

### Declaring inline

```yaml
# katalog.yaml
notes:
  - name: fullImage
    description: Combine image and tag into a qualified reference
    expression: "{{ .spec.image }}:{{ .spec.tag | default \"latest\" }}"

  - name: serviceHost
    expression: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
```

Once declared, call them anywhere:

```yaml
onReconcile:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ fullImage }}"
```

### Distributing via Motifs

Large teams declare notes in a Motif and import them at the Katalog level via `spec.imports`. Every CRD in the Katalog can then call those notes:

```yaml
# motifs/team-notes/motif.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: team-notes
notes:
  - name: fullImage
    expression: "{{ .spec.image }}:{{ .spec.tag | default \"latest\" }}"
  - name: appLabel
    expression: "{{ .metadata.namespace }}-{{ .metadata.name }}"
```

```yaml
# katalog.yaml
spec:
  imports:
    - motif: ./motifs/team-notes/motif.yaml
```

This is the same distribution path that [profiles](../profiles/10-user-defined-profiles.md) use — `spec.imports` merges both into Katalog-wide registries before any CRD reconciles.

### Conflict rules

| Situation | Result |
|-----------|--------|
| Two `spec.imports` Motifs declare the same note name | Hard error at startup |
| Inline `notes:` and a Motif import declare the same name | Local wins silently |
| Note name matches a built-in | Warning logged; `shadow: true` silences it |

### Shadowing a built-in

```yaml
notes:
  - name: default
    expression: "{{ .spec.fallbackImage }}"
    shadow: true   # acknowledges the override
```

---

## Inspection

```bash
# Browse built-in notes
ork notes

# List user-defined notes from a Katalog file
ork notes -f katalog.yaml
```

---

## Try it

```bash
ork init --pack intermediate
cd intermediate/09-notes/01-built-in
ork validate && ork simulate
```

## See also

- [Reference: built-in notes](../../reference/orkestra-notes/index.md) — full function catalog by domain
- [User-defined profiles](../profiles/10-user-defined-profiles.md) — same distribution mechanism via `spec.imports`
