# Motifs

A Motif is a reusable resource primitive. It declares named inputs and contributes resources, status fields, and admission rules to any CRD that imports it. A Motif has no CRD binding of its own — it is a pattern waiting to be composed.

## Pattern directory

```text
postgres/
  motif.yaml    # required — the Motif declaration
  README.md     # optional — shown in registry UI
  example/
    katalog.yaml  # optional — example Katalog importing this Motif
```

`motif.yaml` is the only required file.

## Writing a Motif
See full description in [Writing your first Motif](../getting-started/03-writing-your-first-motif.md).


## Publishing

Motifs publish to `ORK_MOTIFS_REGISTRY` (default: `ghcr.io/orkspace/orkestra-registry/patterns/motifs`):

```bash
ork push postgres:v16 ./motifs/postgres/
```

If a Katalog pattern directory contains a `motif.yaml`, `ork push` pushes both — the Katalog to the katalog registry and the Motif to the motif registry — in a single command.

## Importing into a Katalog

Motifs are imported at the CRD level, alongside `operatorBox`:

```yaml
spec:
  crds:
    myapp:
      imports:
        - motif: oci://ghcr.io/orkspace/orkestra-registry/patterns/motifs/postgres:v16
          oci: true
          with:
            image: "{{ .spec.database.image }}"
            volumeSize: "{{ .spec.storage | default \"10Gi\" }}"
```

`with:` values are Go templates evaluated in the CR's reconcile context. Required inputs not present in `with:` are caught at startup — not at reconcile time.

## Conditional resources

A Motif can declare resources that only apply when a specific input is set:

```yaml
resources:
  services:
    - name: "{{ .metadata.name }}-admin"
      port: 5050
      when:
        - field: "{{ inputs.adminEnabled }}"
          equals: "true"
```

If `adminEnabled` is not provided or is `"false"`, Orkestra skips this resource entirely. Conditions are evaluated at reconcile time against the bound input values.

## Composing multiple Motifs

A Katalog can import any number of Motifs. Each is independent — resources don't conflict because each uses `.metadata.name` as a prefix:

```yaml
spec:
  crds:
    platform:
      imports:
        - motif: postgres:v16
          with:
            image: "{{ .spec.db.image }}"
        - motif: redis:v7
          with:
            image: redis:7-alpine
            volumeSize: "{{ .spec.cache.size }}"
```
